package cloudflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flectolab/flecto-manager/cdn"
	"github.com/stretchr/testify/assert"
)

func validSettings() cdn.Values {
	return cdn.Values{
		fieldZoneID:    "abc123",
		fieldPlan:      planBusiness,
		cdn.FieldHosts: []string{"www.example.com"},
	}
}

func validSecrets() cdn.Values {
	return cdn.Values{fieldAPIToken: "cf-token"}
}

// newTestProvider builds a provider talking to a stub API.
func newTestProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	built, err := New(validSettings(), validSecrets(), cdn.Options{})
	assert.NoError(t, err)

	provider := built.(*Provider)
	provider.client.baseURL = server.URL
	return provider
}

func TestDescriptor(t *testing.T) {
	descriptor := Descriptor()

	t.Run("successDeclaresEveryField", func(t *testing.T) {
		for _, name := range []string{fieldZoneID, fieldAPIToken, cdn.FieldHosts, fieldPlan} {
			field, ok := descriptor.Field(name)
			assert.True(t, ok, name)
			assert.True(t, field.Required, name)
		}
	})

	t.Run("successKeepsTokenSecret", func(t *testing.T) {
		token, ok := descriptor.Field(fieldAPIToken)

		assert.True(t, ok)
		assert.True(t, token.IsSecret())
	})

	t.Run("successAcceptsMinimalConfiguration", func(t *testing.T) {
		// The plan carries a default, so a submission may leave it out.
		values := cdn.Values{
			fieldZoneID:    "abc123",
			fieldAPIToken:  "cf-token",
			cdn.FieldHosts: []string{"www.example.com"},
		}

		assert.NoError(t, descriptor.Validate(descriptor.ApplyDefaults(values)))
	})

	t.Run("successIsRegistered", func(t *testing.T) {
		got, err := cdn.GetDescriptor(Type)

		assert.NoError(t, err)
		assert.Equal(t, Type, got.Type)
	})
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		plan    string
		want    cdn.Limits
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "successWithFree",
			plan:    planFree,
			want:    cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 800},
			wantErr: assert.NoError,
		},
		{
			name:    "successWithBusiness",
			plan:    planBusiness,
			want:    cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 1500},
			wantErr: assert.NoError,
		},
		{
			name:    "successWithEnterprise",
			plan:    planEnterprise,
			want:    cdn.Limits{MaxTargetsPerRequest: 500, TargetsPerSecond: 3000},
			wantErr: assert.NoError,
		},
		{
			name:    "failWithUnknownPlan",
			plan:    "platinum",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := validSettings()
			settings[fieldPlan] = tt.plan

			got, err := New(settings, validSecrets(), cdn.Options{})

			tt.wantErr(t, err)
			if err != nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.want, got.Limits())
		})
	}
}

func TestProvider_Type(t *testing.T) {
	provider, err := New(validSettings(), validSecrets(), cdn.Options{})

	assert.NoError(t, err)
	assert.Equal(t, Type, provider.Type())
}

func TestProvider_Capabilities(t *testing.T) {
	provider, err := New(validSettings(), validSecrets(), cdn.Options{})
	assert.NoError(t, err)

	got := provider.Capabilities()

	assert.True(t, got.Supports(cdn.KindURL))
	// Prefix and hostname purges exist on every Cloudflare plan but are not offered:
	// they drop cached objects nobody published a change for.
	assert.False(t, got.Supports(cdn.KindPrefix))
	assert.False(t, got.Supports(cdn.KindHostname))
}

func TestProvider_Verify(t *testing.T) {
	t.Run("successWhenZoneIsReadable", func(t *testing.T) {
		var paths []string
		provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			writeAPIResponse(t, w, http.StatusOK, apiResponse{Success: true})
		})

		err := provider.Verify(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, []string{"/zones/abc123"}, paths)
	})

	t.Run("successFallsBackWhenTokenCannotReadZone", func(t *testing.T) {
		// The recommended token scope is Cache Purge alone, which cannot read the
		// zone. Verifying the token is then the best the API allows.
		var paths []string
		provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			if r.URL.Path == "/zones/abc123" {
				writeAPIResponse(t, w, http.StatusForbidden, apiResponse{
					Errors: []apiError{{Code: 9109, Message: "Unauthorized to access requested resource"}},
				})
				return
			}
			writeAPIResponse(t, w, http.StatusOK, apiResponse{Success: true})
		})

		err := provider.Verify(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, []string{"/zones/abc123", "/user/tokens/verify"}, paths)
	})

	t.Run("failWithInvalidToken", func(t *testing.T) {
		provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusUnauthorized, apiResponse{
				Errors: []apiError{{Code: 10000, Message: "Authentication error"}},
			})
		})

		err := provider.Verify(context.Background())

		assert.ErrorContains(t, err, "Authentication error")
	})

	t.Run("failWhenFallbackAlsoFails", func(t *testing.T) {
		provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusForbidden, apiResponse{
				Errors: []apiError{{Code: 9109, Message: "Unauthorized to access requested resource"}},
			})
		})

		err := provider.Verify(context.Background())

		assert.ErrorContains(t, err, "9109")
	})

	t.Run("failWithUnknownZone", func(t *testing.T) {
		provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusBadRequest, apiResponse{
				Errors: []apiError{{Code: 7003, Message: "Could not route to /zones/abc123"}},
			})
		})

		err := provider.Verify(context.Background())

		// A malformed or unknown zone id must not be hidden by the token fallback.
		assert.ErrorContains(t, err, "7003")
	})
}

func TestProvider_Purge(t *testing.T) {
	t.Run("successSendsBatchToZone", func(t *testing.T) {
		var (
			path string
			body purgeRequest
		)
		provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			path = r.URL.Path
			raw, _ := io.ReadAll(r.Body)
			assert.NoError(t, json.Unmarshal(raw, &body))
			writeAPIResponse(t, w, http.StatusOK, apiResponse{Success: true})
		})

		err := provider.Purge(context.Background(), []cdn.Target{
			{Kind: cdn.KindURL, Host: "www.example.com", Path: "/promo"},
			{Kind: cdn.KindURL, Host: "example.com", Path: "/robots.txt"},
		})

		assert.NoError(t, err)
		assert.Equal(t, "/zones/abc123/purge_cache", path)
		assert.Equal(t, []string{"https://www.example.com/promo", "https://example.com/robots.txt"}, body.Files)
	})

	t.Run("successWithEmptyBatchMakesNoCall", func(t *testing.T) {
		called := false
		provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

		err := provider.Purge(context.Background(), nil)

		assert.NoError(t, err)
		assert.False(t, called)
	})

	t.Run("failWithUnsupportedTargetKind", func(t *testing.T) {
		called := false
		provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

		err := provider.Purge(context.Background(), []cdn.Target{{Kind: cdn.KindPrefix, Host: "a.com", Path: "/x"}})

		assert.ErrorContains(t, err, "cannot purge a PREFIX target")
		assert.False(t, called)
	})

	t.Run("failWhenApiRejectsTheBatch", func(t *testing.T) {
		provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusBadRequest, apiResponse{
				Errors: []apiError{{Code: 1012, Message: "Request failed"}},
			})
		})

		err := provider.Purge(context.Background(), []cdn.Target{{Kind: cdn.KindURL, Host: "a.com", Path: "/x"}})

		assert.ErrorContains(t, err, "1012")
	})
}

func TestRenderURL(t *testing.T) {
	tests := []struct {
		name    string
		target  cdn.Target
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "success",
			target:  cdn.Target{Kind: cdn.KindURL, Host: "www.example.com", Path: "/promo"},
			want:    "https://www.example.com/promo",
			wantErr: assert.NoError,
		},
		{
			name:    "successWithPort",
			target:  cdn.Target{Kind: cdn.KindURL, Host: "www.example.com:8443", Path: "/promo"},
			want:    "https://www.example.com:8443/promo",
			wantErr: assert.NoError,
		},
		{
			name:    "failWithoutHost",
			target:  cdn.Target{Kind: cdn.KindURL, Path: "/promo"},
			wantErr: assert.Error,
		},
		{
			name:    "failWithoutPath",
			target:  cdn.Target{Kind: cdn.KindURL, Host: "www.example.com"},
			wantErr: assert.Error,
		},
		{
			name:    "failWithHostnameKind",
			target:  cdn.Target{Kind: cdn.KindHostname, Host: "www.example.com", Path: "/"},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderURL(tt.target)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

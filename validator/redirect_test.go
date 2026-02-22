package validator

import (
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestContainsScheme(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "http", input: "http://example.com", want: true},
		{name: "https", input: "https://example.com", want: true},
		{name: "HTTP uppercase", input: "HTTP://example.com", want: true},
		{name: "HTTPS uppercase", input: "HTTPS://example.com", want: true},
		{name: "http in middle", input: "^(http://example.com)", want: true},
		{name: "https after caret", input: "^https://example.com", want: true},
		{name: "ftp scheme", input: "ftp://example.com", want: true},
		{name: "protocol relative", input: "//example.com", want: false},
		{name: "no scheme", input: "example.com/path", want: false},
		{name: "colon without double slash", input: "example.com:80/path", want: false},
		{name: "empty string", input: "", want: false},
		{name: "path only", input: "/path/to/resource", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, containsScheme(tt.input))
		})
	}
}

func TestHasSchemePrefix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "http", input: "http://example.com", want: true},
		{name: "https", input: "https://example.com", want: true},
		{name: "HTTP uppercase", input: "HTTP://example.com", want: true},
		{name: "HTTPS uppercase", input: "HTTPS://example.com", want: true},
		{name: "protocol relative", input: "//example.com", want: true},
		{name: "no scheme", input: "example.com/path", want: false},
		{name: "colon without double slash", input: "example.com:80/path", want: false},
		{name: "empty string", input: "", want: false},
		{name: "path only", input: "/path/to/resource", want: false},
		{name: "http in middle", input: "^http://example.com", want: false},
		{name: "scheme after parenthesis", input: "(https://example.com)", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasSchemePrefix(tt.input))
		})
	}
}

func TestValidateRedirect(t *testing.T) {
	validate := validator.New()
	validate.RegisterStructValidation(ValidateRedirect, commonTypes.Redirect{})
	tests := []struct {
		name     string
		redirect *commonTypes.Redirect
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "successWithBasic",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithBasicHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "example.com/source",
				Target: "https://example.com/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithBasicHostAndPort",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "example.com:80/source",
				Target: "https://example.com/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithRegex",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegex,
				Source: "^/source/[0-9]+$",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithRegex2",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegex,
				Source: "/source/[0-9]+",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithRegexWithGroup",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegex,
				Source: "/source/([0-9]+)",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithRegexHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegexHost,
				Source: "^example\\.com/source/[0-9]+$",
				Target: "https://example.com/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "failedStatusEmpty",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
			},
			wantErr: assert.Error,
		},
		{
			name: "failedTypeEmpty",
			redirect: &commonTypes.Redirect{
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedTargetEmpty",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceInvalidWithBasic",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceInvalidWithBasicHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "source",
				Target: "https://example.com/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceWithProtocolRelativeBasicHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "//example.com/source",
				Target: "https://example.com/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceWithSchemeHttpBasicHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "http://example.com/source",
				Target: "https://example.com/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceWithSchemeHttpsBasicHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "https://example.com/source",
				Target: "https://example.com/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedTargetWithoutSchemeBasicHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "example.com/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceWithSchemeRegexHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegexHost,
				Source: "http://example.com/source/[0-9]+",
				Target: "https://example.com/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedTargetWithoutSchemeRegexHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegexHost,
				Source: "example.com/source/[0-9]+",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceInvalidWithRegexHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegexHost,
				Source: "example.com/source[",
				Target: "https://example.com/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceInvalidWithRegex",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegex,
				Source: "source[",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.redirect)
			tt.wantErr(t, err, "Redirect is not valid")
		})
	}
}

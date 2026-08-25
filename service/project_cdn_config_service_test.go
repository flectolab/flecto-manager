package service

import (
	"context"
	"errors"
	"testing"

	"github.com/flectolab/flecto-manager/cdn"
	_ "github.com/flectolab/flecto-manager/cdn/cloudflare"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/crypt"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func setupProjectCdnConfigServiceTest(t *testing.T) (
	*gomock.Controller,
	*mockFlectoRepository.MockProjectCdnConfigRepository,
	*mockFlectoRepository.MockActivityEventRepository,
	*appContext.Context,
	ProjectCdnConfigService,
) {
	ctrl := gomock.NewController(t)
	repo := mockFlectoRepository.NewMockProjectCdnConfigRepository(ctrl)
	activityRepo := mockFlectoRepository.NewMockActivityEventRepository(ctrl)

	ctx := appContext.TestContext(nil)
	key, err := crypt.GenerateKey()
	assert.NoError(t, err)
	cipher, err := crypt.NewAESGCM(key)
	assert.NoError(t, err)
	ctx.Cipher = cipher

	svc := NewProjectCdnConfigService(ctx, repo, NewActivityService(ctx, activityRepo))
	return ctrl, repo, activityRepo, ctx, svc
}

func cloudflareInput() model.ProjectCdnConfigInput {
	return model.ProjectCdnConfigInput{
		Name: "prod",
		Type: "cloudflare",
		Values: cdn.Values{
			"zone_id":      "abc123",
			"api_token":    "cf-token",
			cdn.FieldHosts: []string{"www.example.com"},
		},
	}
}

func storedCloudflareConfig() *model.ProjectCdnConfig {
	return &model.ProjectCdnConfig{
		ID:            7,
		NamespaceCode: "ns1",
		ProjectCode:   "proj1",
		Name:          "prod",
		Type:          "cloudflare",
		Enabled:       types.Ptr(true),
		Settings: cdn.Values{
			"zone_id":      "abc123",
			"plan":         "business",
			cdn.FieldHosts: []string{"www.example.com"},
		},
		Secrets: cdn.Values{"api_token": "cf-token"},
	}
}

func TestProjectCdnConfigService_Descriptors(t *testing.T) {
	ctrl, _, _, _, svc := setupProjectCdnConfigServiceTest(t)
	defer ctrl.Finish()

	got := svc.Descriptors()

	assert.NotEmpty(t, got)
}

func TestProjectCdnConfigService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, activityRepo, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, gorm.ErrRecordNotFound)
		repo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		activityRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		got, err := svc.Create(ctx, "ns1", "proj1", cloudflareInput())

		assert.NoError(t, err)
		assert.Equal(t, "prod", got.Name)
		assert.True(t, got.IsEnabled())
	})

	t.Run("successSplitsSecretsFromSettings", func(t *testing.T) {
		ctrl, repo, activityRepo, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, gorm.ErrRecordNotFound)
		repo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		activityRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		got, err := svc.Create(ctx, "ns1", "proj1", cloudflareInput())

		assert.NoError(t, err)
		assert.Equal(t, "abc123", got.Settings.String("zone_id"))
		assert.Equal(t, "cf-token", got.Secrets.String("api_token"))
		// The token must never reach the column stored in clear.
		assert.False(t, got.Settings.Has("api_token"))
	})

	t.Run("successAppliesProviderDefaults", func(t *testing.T) {
		ctrl, repo, activityRepo, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, gorm.ErrRecordNotFound)
		repo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		activityRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		got, err := svc.Create(ctx, "ns1", "proj1", cloudflareInput())

		assert.NoError(t, err)
		assert.Equal(t, "business", got.Settings.String("plan"))
	})

	t.Run("successKeepsDisabledWhenAsked", func(t *testing.T) {
		ctrl, repo, activityRepo, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		input := cloudflareInput()
		input.Enabled = types.Ptr(false)

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, gorm.ErrRecordNotFound)
		repo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		activityRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		got, err := svc.Create(ctx, "ns1", "proj1", input)

		assert.NoError(t, err)
		assert.False(t, got.IsEnabled())
	})

	t.Run("failWithUnknownType", func(t *testing.T) {
		ctrl, _, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()

		input := cloudflareInput()
		input.Type = "akamai"

		got, err := svc.Create(context.Background(), "ns1", "proj1", input)

		assert.ErrorContains(t, err, "unknown cdn provider type")
		assert.Nil(t, got)
	})

	t.Run("failWithNameAlreadyUsed", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(storedCloudflareConfig(), nil)

		got, err := svc.Create(ctx, "ns1", "proj1", cloudflareInput())

		assert.ErrorIs(t, err, ErrCdnNameAlreadyUsed)
		assert.Nil(t, got)
	})

	t.Run("failWhenLookupFails", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, errors.New("boom"))

		_, err := svc.Create(ctx, "ns1", "proj1", cloudflareInput())

		assert.ErrorContains(t, err, "boom")
	})

	t.Run("failWithInvalidProviderValues", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		input := cloudflareInput()
		delete(input.Values, "zone_id")

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, gorm.ErrRecordNotFound)

		_, err := svc.Create(ctx, "ns1", "proj1", input)

		assert.ErrorContains(t, err, "zone_id is required")
	})

	t.Run("failWithInvalidName", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		input := cloudflareInput()
		input.Name = "not a code"

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "not a code").Return(nil, gorm.ErrRecordNotFound)

		_, err := svc.Create(ctx, "ns1", "proj1", input)

		assert.Error(t, err)
	})

	t.Run("failWithoutEncryptionKey", func(t *testing.T) {
		ctrl, repo, _, appCtx, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		appCtx.Cipher = crypt.NewUnavailable()
		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, gorm.ErrRecordNotFound)

		_, err := svc.Create(ctx, "ns1", "proj1", cloudflareInput())

		// Storing an API token in clear is not an acceptable fallback.
		assert.ErrorIs(t, err, ErrCdnEncryptionUnavailable)
	})

	t.Run("failWhenCreateFails", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, gorm.ErrRecordNotFound)
		repo.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("boom"))

		_, err := svc.Create(ctx, "ns1", "proj1", cloudflareInput())

		assert.ErrorContains(t, err, "boom")
	})

	t.Run("successEvenWhenActivityFails", func(t *testing.T) {
		ctrl, repo, activityRepo, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, gorm.ErrRecordNotFound)
		repo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		activityRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("boom"))

		// The configuration exists, so failing to journal it must not report failure.
		got, err := svc.Create(ctx, "ns1", "proj1", cloudflareInput())

		assert.NoError(t, err)
		assert.NotNil(t, got)
	})
}

func TestProjectCdnConfigService_Update(t *testing.T) {
	t.Run("successKeepsStoredSecretWhenNotSubmitted", func(t *testing.T) {
		ctrl, repo, activityRepo, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(storedCloudflareConfig(), nil)
		repo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		activityRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		got, err := svc.Update(ctx, "ns1", "proj1", "prod", model.ProjectCdnConfigInput{
			Values: cdn.Values{
				"zone_id":      "new-zone",
				cdn.FieldHosts: []string{"www.example.com"},
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, "new-zone", got.Settings.String("zone_id"))
		// The form never receives the token, so it cannot send it back.
		assert.Equal(t, "cf-token", got.Secrets.String("api_token"))
	})

	t.Run("successReplacesSubmittedSecret", func(t *testing.T) {
		ctrl, repo, activityRepo, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(storedCloudflareConfig(), nil)
		repo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		activityRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		got, err := svc.Update(ctx, "ns1", "proj1", "prod", model.ProjectCdnConfigInput{
			Values: cdn.Values{
				"zone_id":      "abc123",
				"api_token":    "rotated",
				cdn.FieldHosts: []string{"www.example.com"},
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, "rotated", got.Secrets.String("api_token"))
	})

	t.Run("successTogglesEnabled", func(t *testing.T) {
		ctrl, repo, activityRepo, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(storedCloudflareConfig(), nil)
		repo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		activityRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		got, err := svc.Update(ctx, "ns1", "proj1", "prod", model.ProjectCdnConfigInput{
			Enabled: types.Ptr(false),
			Values: cdn.Values{
				"zone_id":      "abc123",
				cdn.FieldHosts: []string{"www.example.com"},
			},
		})

		assert.NoError(t, err)
		// Disabling keeps the zone and the credentials, it only stops the purges.
		assert.False(t, got.IsEnabled())
		assert.Equal(t, "cf-token", got.Secrets.String("api_token"))
	})

	t.Run("failWithUnknownConfig", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "nope").Return(nil, gorm.ErrRecordNotFound)

		_, err := svc.Update(ctx, "ns1", "proj1", "nope", model.ProjectCdnConfigInput{})

		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("failWhenTypeChanges", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(storedCloudflareConfig(), nil)

		_, err := svc.Update(ctx, "ns1", "proj1", "prod", model.ProjectCdnConfigInput{Type: "akamai"})

		// The stored values belong to the original provider.
		assert.ErrorIs(t, err, ErrCdnTypeImmutable)
	})

	t.Run("failWhenUpdateFails", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(storedCloudflareConfig(), nil)
		repo.EXPECT().Update(ctx, gomock.Any()).Return(errors.New("boom"))

		_, err := svc.Update(ctx, "ns1", "proj1", "prod", model.ProjectCdnConfigInput{
			Values: cdn.Values{"zone_id": "abc123", cdn.FieldHosts: []string{"www.example.com"}},
		})

		assert.ErrorContains(t, err, "boom")
	})
}

func TestProjectCdnConfigService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, activityRepo, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(storedCloudflareConfig(), nil)
		repo.EXPECT().Delete(ctx, "ns1", "proj1", "prod").Return(nil)
		activityRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		got, err := svc.Delete(ctx, "ns1", "proj1", "prod")

		assert.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("successWithUnknownConfig", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "nope").Return(nil, gorm.ErrRecordNotFound)

		got, err := svc.Delete(ctx, "ns1", "proj1", "nope")

		assert.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("failWhenLookupFails", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(nil, errors.New("boom"))

		got, err := svc.Delete(ctx, "ns1", "proj1", "prod")

		assert.ErrorContains(t, err, "boom")
		assert.False(t, got)
	})

	t.Run("failWhenDeleteFails", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(storedCloudflareConfig(), nil)
		repo.EXPECT().Delete(ctx, "ns1", "proj1", "prod").Return(errors.New("boom"))

		got, err := svc.Delete(ctx, "ns1", "proj1", "prod")

		assert.ErrorContains(t, err, "boom")
		assert.False(t, got)
	})
}

func TestProjectCdnConfigService_Provider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, _, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()

		got, err := svc.Provider(storedCloudflareConfig())

		assert.NoError(t, err)
		assert.Equal(t, "cloudflare", got.Type())
	})

	t.Run("failWithNilConfig", func(t *testing.T) {
		ctrl, _, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()

		_, err := svc.Provider(nil)

		assert.Error(t, err)
	})

	t.Run("failWithUnknownType", func(t *testing.T) {
		ctrl, _, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()

		config := storedCloudflareConfig()
		config.Type = "akamai"

		_, err := svc.Provider(config)

		assert.ErrorContains(t, err, "unknown cdn provider type")
	})
}

func TestProjectCdnConfigService_Test(t *testing.T) {
	t.Run("failWithUnknownConfig", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "nope").Return(nil, gorm.ErrRecordNotFound)

		err := svc.Test(ctx, "ns1", "proj1", "nope")

		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("failWithUnknownType", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		config := storedCloudflareConfig()
		config.Type = "akamai"
		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(config, nil)

		err := svc.Test(ctx, "ns1", "proj1", "prod")

		assert.ErrorContains(t, err, "unknown cdn provider type")
	})

	t.Run("successWhenProviderAcceptsTheCredentials", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		registerFakeProvider(nil)
		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "fake").Return(fakeConfig(), nil)

		err := svc.Test(ctx, "ns1", "proj1", "fake")

		assert.NoError(t, err)
	})

	t.Run("failWhenProviderRejectsTheCredentials", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		registerFakeProvider(errors.New("authentication error"))
		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "fake").Return(fakeConfig(), nil)

		err := svc.Test(ctx, "ns1", "proj1", "fake")

		assert.ErrorContains(t, err, "authentication error")
	})
}

func TestProjectCdnConfigService_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByProject(ctx, "ns1", "proj1").Return([]model.ProjectCdnConfig{*storedCloudflareConfig()}, nil)

		got, err := svc.List(ctx, "ns1", "proj1")

		assert.NoError(t, err)
		assert.Len(t, got, 1)
	})
}

func TestProjectCdnConfigService_ListEnabled(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindEnabledByProject(ctx, "ns1", "proj1").Return([]model.ProjectCdnConfig{*storedCloudflareConfig()}, nil)

		got, err := svc.ListEnabled(ctx, "ns1", "proj1")

		assert.NoError(t, err)
		assert.Len(t, got, 1)
	})
}

func TestProjectCdnConfigService_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(storedCloudflareConfig(), nil)

		got, err := svc.Get(ctx, "ns1", "proj1", "prod")

		assert.NoError(t, err)
		assert.Equal(t, "prod", got.Name)
	})
}

func TestProjectCdnConfigService_GetQuery(t *testing.T) {
	ctrl, repo, _, _, svc := setupProjectCdnConfigServiceTest(t)
	defer ctrl.Finish()
	ctx := context.Background()

	repo.EXPECT().GetTx(ctx).Return(nil)
	repo.EXPECT().GetQuery(ctx).Return(nil)

	assert.Nil(t, svc.GetTx(ctx))
	assert.Nil(t, svc.GetQuery(ctx))
}

// fakeProviderType is a provider registered by the tests alone, so that the paths
// going all the way through a provider are exercised without a network call.
const fakeProviderType = "test-fake"

type fakeProvider struct {
	verifyErr error
}

func (p *fakeProvider) Type() string                              { return fakeProviderType }
func (p *fakeProvider) Capabilities() cdn.Capabilities            { return cdn.Capabilities{URL: true} }
func (p *fakeProvider) Limits() cdn.Limits                        { return cdn.Limits{MaxTargetsPerRequest: 10} }
func (p *fakeProvider) Verify(context.Context) error              { return p.verifyErr }
func (p *fakeProvider) Purge(context.Context, []cdn.Target) error { return nil }

// registerFakeProvider installs the fake under its own type, leaving the real
// providers registered alongside it.
func registerFakeProvider(verifyErr error) {
	cdn.Register(cdn.Descriptor{
		Type:  fakeProviderType,
		Label: "Fake",
		Fields: []cdn.Field{
			{Name: cdn.FieldHosts, Label: "Hosts", Kind: cdn.FieldKindStringList, Required: true},
		},
		Capabilities: cdn.Capabilities{URL: true},
	}, func(settings, secrets cdn.Values, options cdn.Options) (cdn.Provider, error) {
		return &fakeProvider{verifyErr: verifyErr}, nil
	})
}

func fakeConfig() *model.ProjectCdnConfig {
	return &model.ProjectCdnConfig{
		ID:            8,
		NamespaceCode: "ns1",
		ProjectCode:   "proj1",
		Name:          "fake",
		Type:          fakeProviderType,
		Enabled:       types.Ptr(true),
		Settings:      cdn.Values{cdn.FieldHosts: []string{"www.example.com"}},
	}
}

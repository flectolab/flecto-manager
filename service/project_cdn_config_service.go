package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/flectolab/flecto-manager/cdn"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/crypt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/types"
	"gorm.io/gorm"
)

var (
	// ErrCdnEncryptionUnavailable refuses secrets without an encryption key, rather
	// than storing them in clear.
	ErrCdnEncryptionUnavailable = errors.New("cdn configuration needs security.encryption_key to be set")
	// ErrCdnNameAlreadyUsed is returned on a duplicate name within a project.
	ErrCdnNameAlreadyUsed = errors.New("cdn configuration name is already used in this project")
	// ErrCdnTypeImmutable is returned when an update changes the provider type, whose
	// stored values would mean nothing to another one.
	ErrCdnTypeImmutable = errors.New("cdn configuration type cannot be changed")
)

type ProjectCdnConfigService interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	// Descriptors lists the provider types with the fields the admin UI renders.
	Descriptors() []cdn.Descriptor
	// EncryptionAvailable reports whether security.encryption_key is set.
	EncryptionAvailable() bool
	List(ctx context.Context, namespaceCode, projectCode string) ([]model.ProjectCdnConfig, error)
	// ListEnabled returns the configurations a publish purges through.
	ListEnabled(ctx context.Context, namespaceCode, projectCode string) ([]model.ProjectCdnConfig, error)
	Get(ctx context.Context, namespaceCode, projectCode, name string) (*model.ProjectCdnConfig, error)
	Create(ctx context.Context, namespaceCode, projectCode string, input model.ProjectCdnConfigInput) (*model.ProjectCdnConfig, error)
	Update(ctx context.Context, namespaceCode, projectCode, name string, input model.ProjectCdnConfigInput) (*model.ProjectCdnConfig, error)
	Delete(ctx context.Context, namespaceCode, projectCode, name string) (bool, error)
	// Test checks the credentials against the CDN API.
	Test(ctx context.Context, namespaceCode, projectCode, name string) error
	Provider(config *model.ProjectCdnConfig) (cdn.Provider, error)
}

type projectCdnConfigService struct {
	ctx      *appContext.Context
	repo     repository.ProjectCdnConfigRepository
	activity ActivityService
}

func NewProjectCdnConfigService(
	ctx *appContext.Context,
	repo repository.ProjectCdnConfigRepository,
	activity ActivityService,
) ProjectCdnConfigService {
	return &projectCdnConfigService{ctx: ctx, repo: repo, activity: activity}
}

func (s *projectCdnConfigService) GetTx(ctx context.Context) *gorm.DB {
	return s.repo.GetTx(ctx)
}

func (s *projectCdnConfigService) GetQuery(ctx context.Context) *gorm.DB {
	return s.repo.GetQuery(ctx)
}

func (s *projectCdnConfigService) Descriptors() []cdn.Descriptor {
	return cdn.Descriptors()
}

func (s *projectCdnConfigService) EncryptionAvailable() bool {
	return !crypt.IsUnavailable(s.ctx.Cipher)
}

func (s *projectCdnConfigService) List(ctx context.Context, namespaceCode, projectCode string) ([]model.ProjectCdnConfig, error) {
	return s.repo.FindByProject(ctx, namespaceCode, projectCode)
}

func (s *projectCdnConfigService) ListEnabled(ctx context.Context, namespaceCode, projectCode string) ([]model.ProjectCdnConfig, error) {
	return s.repo.FindEnabledByProject(ctx, namespaceCode, projectCode)
}

func (s *projectCdnConfigService) Get(ctx context.Context, namespaceCode, projectCode, name string) (*model.ProjectCdnConfig, error) {
	return s.repo.FindByName(ctx, namespaceCode, projectCode, name)
}

func (s *projectCdnConfigService) Create(ctx context.Context, namespaceCode, projectCode string, input model.ProjectCdnConfigInput) (*model.ProjectCdnConfig, error) {
	descriptor, err := cdn.GetDescriptor(input.Type)
	if err != nil {
		return nil, err
	}

	if _, errExisting := s.repo.FindByName(ctx, namespaceCode, projectCode, input.Name); errExisting == nil {
		return nil, ErrCdnNameAlreadyUsed
	} else if !errors.Is(errExisting, gorm.ErrRecordNotFound) {
		return nil, errExisting
	}

	config := &model.ProjectCdnConfig{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		Name:          input.Name,
		Type:          input.Type,
		Enabled:       enabledOrDefault(input.Enabled),
	}

	if err = s.applyValues(config, descriptor, cdn.Values{}, input.Values); err != nil {
		return nil, err
	}

	if err = s.ctx.Validator.Struct(config); err != nil {
		return nil, err
	}

	if err = s.repo.Create(ctx, config); err != nil {
		s.ctx.Logger.Error("failed to create cdn configuration",
			"namespace", namespaceCode, "project", projectCode, "name", input.Name, "error", err)
		return nil, err
	}

	s.recordActivity(ctx, config, model.ActivityActionCreate, model.ActivityChange[model.CdnConfigSnapshot]{
		After: model.NewCdnConfigSnapshot(config),
	})

	s.ctx.Logger.Info("cdn configuration created",
		"namespace", namespaceCode, "project", projectCode, "name", config.Name, "type", config.Type)
	return config, nil
}

func (s *projectCdnConfigService) Update(ctx context.Context, namespaceCode, projectCode, name string, input model.ProjectCdnConfigInput) (*model.ProjectCdnConfig, error) {
	config, err := s.repo.FindByName(ctx, namespaceCode, projectCode, name)
	if err != nil {
		return nil, err
	}

	if input.Type != "" && input.Type != config.Type {
		return nil, ErrCdnTypeImmutable
	}

	descriptor, err := cdn.GetDescriptor(config.Type)
	if err != nil {
		return nil, err
	}

	before := model.NewCdnConfigSnapshot(config)
	storedSecrets := config.Secrets

	if input.Enabled != nil {
		config.Enabled = input.Enabled
	}

	if err = s.applyValues(config, descriptor, storedSecrets, input.Values); err != nil {
		return nil, err
	}

	if err = s.ctx.Validator.Struct(config); err != nil {
		return nil, err
	}

	if err = s.repo.Update(ctx, config); err != nil {
		s.ctx.Logger.Error("failed to update cdn configuration",
			"namespace", namespaceCode, "project", projectCode, "name", name, "error", err)
		return nil, err
	}

	s.recordActivity(ctx, config, model.ActivityActionUpdate, model.ActivityChange[model.CdnConfigSnapshot]{
		Before: before,
		After:  model.NewCdnConfigSnapshot(config),
	})

	s.ctx.Logger.Info("cdn configuration updated",
		"namespace", namespaceCode, "project", projectCode, "name", name, "enabled", config.IsEnabled())
	return config, nil
}

func (s *projectCdnConfigService) Delete(ctx context.Context, namespaceCode, projectCode, name string) (bool, error) {
	config, err := s.repo.FindByName(ctx, namespaceCode, projectCode, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	if err = s.repo.Delete(ctx, namespaceCode, projectCode, name); err != nil {
		return false, err
	}

	s.recordActivity(ctx, config, model.ActivityActionDelete, model.ActivityChange[model.CdnConfigSnapshot]{
		Before: model.NewCdnConfigSnapshot(config),
	})

	s.ctx.Logger.Info("cdn configuration deleted",
		"namespace", namespaceCode, "project", projectCode, "name", name)
	return true, nil
}

func (s *projectCdnConfigService) Test(ctx context.Context, namespaceCode, projectCode, name string) error {
	config, err := s.repo.FindByName(ctx, namespaceCode, projectCode, name)
	if err != nil {
		return err
	}

	provider, err := s.Provider(config)
	if err != nil {
		return err
	}

	return provider.Verify(ctx)
}

func (s *projectCdnConfigService) Provider(config *model.ProjectCdnConfig) (cdn.Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("cdn configuration is required")
	}
	return cdn.New(config.Type, config.Settings, config.Secrets, cdnProviderOptions(s.ctx))
}

func cdnProviderOptions(ctx *appContext.Context) cdn.Options {
	return cdn.Options{HTTPTimeout: ctx.Config.Cdn.HTTPTimeout}
}

// applyValues keeps the stored secrets left out of a submission: the form never
// receives them, so it cannot send them back.
func (s *projectCdnConfigService) applyValues(
	config *model.ProjectCdnConfig,
	descriptor cdn.Descriptor,
	storedSecrets cdn.Values,
	submitted cdn.Values,
) error {
	values := descriptor.ApplyDefaults(descriptor.MergeSecrets(storedSecrets, submitted))

	if err := descriptor.Validate(values); err != nil {
		return err
	}

	settings, secrets := descriptor.Split(values)

	if len(secrets) > 0 && crypt.IsUnavailable(s.ctx.Cipher) {
		return ErrCdnEncryptionUnavailable
	}

	config.Settings = settings
	config.Secrets = secrets
	return nil
}

// recordActivity logs a failure rather than returning it: the change already happened.
func (s *projectCdnConfigService) recordActivity(
	ctx context.Context,
	config *model.ProjectCdnConfig,
	action model.ActivityAction,
	change model.ActivityChange[model.CdnConfigSnapshot],
) {
	err := s.activity.Record(ctx, nil, model.ActivityInput{
		NamespaceCode: config.NamespaceCode,
		ProjectCode:   config.ProjectCode,
		Resource:      model.ActivityResourceCdn,
		Action:        action,
		ResourceID:    types.Ptr(config.ID),
		Data:          change,
	})
	if err != nil {
		s.ctx.Logger.Error("failed to record cdn activity",
			"namespace", config.NamespaceCode, "project", config.ProjectCode, "name", config.Name, "error", err)
	}
}

func enabledOrDefault(enabled *bool) *bool {
	if enabled == nil {
		return types.Ptr(true)
	}
	return enabled
}

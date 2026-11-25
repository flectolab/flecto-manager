package repository

import (
	"context"
	"testing"

	"github.com/flectolab/flecto-manager/cdn"
	"github.com/flectolab/flecto-manager/crypt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProjectCdnConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// The secrets column needs a cipher registered, as at boot.
	key, err := crypt.GenerateKey()
	assert.NoError(t, err)
	cipher, err := crypt.NewAESGCM(key)
	assert.NoError(t, err)
	crypt.RegisterSerializer(cipher)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.ProjectCdnConfig{}))

	assert.NoError(t, db.Create(&model.Namespace{NamespaceCode: "ns1", Name: "ns1"}).Error)
	assert.NoError(t, db.Create(&model.Project{NamespaceCode: "ns1", ProjectCode: "proj1", Name: "proj1"}).Error)

	return db
}

func newTestCdnConfig(name string, enabled bool) *model.ProjectCdnConfig {
	return &model.ProjectCdnConfig{
		NamespaceCode: "ns1",
		ProjectCode:   "proj1",
		Name:          name,
		Type:          "cloudflare",
		Enabled:       types.Ptr(enabled),
		Settings:      cdn.Values{"zone_id": "abc123", cdn.FieldHosts: []string{"www.example.com"}},
		Secrets:       cdn.Values{"api_token": "cf-token"},
	}
}

func TestProjectCdnConfigRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		config := newTestCdnConfig("prod", true)

		err := repo.Create(context.Background(), config)

		assert.NoError(t, err)
		assert.NotZero(t, config.ID)
	})

	t.Run("successStoresSecretsEncrypted", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("prod", true)))

		var stored string
		assert.NoError(t, db.Raw("SELECT secrets FROM project_cdn_configs LIMIT 1").Scan(&stored).Error)

		assert.NotContains(t, stored, "cf-token")
	})

	t.Run("successKeepsSettingsReadable", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("prod", true)))

		var stored string
		assert.NoError(t, db.Raw("SELECT settings FROM project_cdn_configs LIMIT 1").Scan(&stored).Error)

		assert.Contains(t, stored, "abc123")
	})
}

func TestProjectCdnConfigRepository_MultipleSecrets(t *testing.T) {
	t.Run("successRoundTripsEverySecretThroughOneColumn", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)

		config := newTestCdnConfig("prod", true)
		config.Secrets = cdn.Values{"access_key": "AKIA", "secret_key": "s3cret"}
		assert.NoError(t, repo.Create(context.Background(), config))

		got, err := repo.FindByName(context.Background(), "ns1", "proj1", "prod")

		assert.NoError(t, err)
		assert.Equal(t, "AKIA", got.Secrets.String("access_key"))
		assert.Equal(t, "s3cret", got.Secrets.String("secret_key"))
	})

	t.Run("successKeepsEverySecretOutOfTheStoredColumn", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)

		config := newTestCdnConfig("prod", true)
		config.Secrets = cdn.Values{"access_key": "AKIA", "secret_key": "s3cret"}
		assert.NoError(t, repo.Create(context.Background(), config))

		var stored string
		assert.NoError(t, db.Raw("SELECT secrets FROM project_cdn_configs LIMIT 1").Scan(&stored).Error)

		assert.NotContains(t, stored, "AKIA")
		assert.NotContains(t, stored, "s3cret")
		// The whole object is sealed, field names included.
		assert.NotContains(t, stored, "access_key")
	})
}

func TestProjectCdnConfigRepository_FindByProject(t *testing.T) {
	t.Run("successSortsByName", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("zone-b", true)))
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("zone-a", false)))

		got, err := repo.FindByProject(context.Background(), "ns1", "proj1")

		assert.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, "zone-a", got[0].Name)
		assert.Equal(t, "zone-b", got[1].Name)
	})

	t.Run("successRoundTripsValues", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("prod", true)))

		got, err := repo.FindByProject(context.Background(), "ns1", "proj1")

		assert.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "abc123", got[0].Settings.String("zone_id"))
		assert.Equal(t, []string{"www.example.com"}, got[0].Hosts())
		assert.Equal(t, "cf-token", got[0].Secrets.String("api_token"))
	})

	t.Run("successWithoutConfig", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)

		got, err := repo.FindByProject(context.Background(), "ns1", "proj1")

		assert.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("successIgnoresOtherProjects", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("prod", true)))

		got, err := repo.FindByProject(context.Background(), "ns1", "other")

		assert.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestProjectCdnConfigRepository_FindEnabledByProject(t *testing.T) {
	t.Run("successKeepsOnlyEnabled", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("on", true)))
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("off", false)))

		got, err := repo.FindEnabledByProject(context.Background(), "ns1", "proj1")

		assert.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "on", got[0].Name)
	})
}

func TestProjectCdnConfigRepository_FindByName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("prod", true)))

		got, err := repo.FindByName(context.Background(), "ns1", "proj1", "prod")

		assert.NoError(t, err)
		assert.Equal(t, "prod", got.Name)
	})

	t.Run("failWithUnknownName", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)

		got, err := repo.FindByName(context.Background(), "ns1", "proj1", "nope")

		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, got)
	})
}

func TestProjectCdnConfigRepository_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		config := newTestCdnConfig("prod", true)
		assert.NoError(t, repo.Create(context.Background(), config))

		config.Enabled = types.Ptr(false)
		config.Secrets = cdn.Values{"api_token": "rotated"}
		err := repo.Update(context.Background(), config)

		assert.NoError(t, err)
		got, errFind := repo.FindByName(context.Background(), "ns1", "proj1", "prod")
		assert.NoError(t, errFind)
		assert.False(t, got.IsEnabled())
		assert.Equal(t, "rotated", got.Secrets.String("api_token"))
	})
}

func TestProjectCdnConfigRepository_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestCdnConfig("prod", true)))

		err := repo.Delete(context.Background(), "ns1", "proj1", "prod")

		assert.NoError(t, err)
		_, errFind := repo.FindByName(context.Background(), "ns1", "proj1", "prod")
		assert.ErrorIs(t, errFind, gorm.ErrRecordNotFound)
	})

	t.Run("failWithUnknownName", func(t *testing.T) {
		db := setupProjectCdnConfigTestDB(t)
		repo := NewProjectCdnConfigRepository(db)

		err := repo.Delete(context.Background(), "ns1", "proj1", "nope")

		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestProjectCdnConfigRepository_GetQuery(t *testing.T) {
	db := setupProjectCdnConfigTestDB(t)
	repo := NewProjectCdnConfigRepository(db)

	assert.NotNil(t, repo.GetTx(context.Background()))
	assert.NotNil(t, repo.GetQuery(context.Background()))
}

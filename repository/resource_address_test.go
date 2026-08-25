package repository

import (
	"context"
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupResourceAddressTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.Page{}))

	assert.NoError(t, db.Create(&model.Namespace{NamespaceCode: "ns1", Name: "ns1"}).Error)
	assert.NoError(t, db.Create(&model.Project{NamespaceCode: "ns1", ProjectCode: "proj1", Name: "proj1"}).Error)

	return db
}

func createAddressRedirect(t *testing.T, db *gorm.DB, source string) *model.Redirect {
	t.Helper()

	redirect := &model.Redirect{
		NamespaceCode: "ns1",
		ProjectCode:   "proj1",
		IsPublished:   types.Ptr(true),
		Redirect: &commonTypes.Redirect{
			Type:   commonTypes.RedirectTypeBasic,
			Source: source,
			Target: "/target",
			Status: commonTypes.RedirectStatusMovedPermanent,
		},
	}
	assert.NoError(t, db.Create(redirect).Error)
	return redirect
}

func TestRedirectRepository_FindAddressesByIDs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupResourceAddressTestDB(t)
		repo := NewRedirectRepository(db)
		first := createAddressRedirect(t, db, "/a")
		second := createAddressRedirect(t, db, "/b")

		got, err := repo.FindAddressesByIDs(context.Background(), nil, []int64{first.ID, second.ID}, 500)

		assert.NoError(t, err)
		assert.Equal(t, []model.ResourceAddress{
			{ID: first.ID, Type: "BASIC", Source: "/a"},
			{ID: second.ID, Type: "BASIC", Source: "/b"},
		}, got)
	})

	t.Run("successWithoutIDs", func(t *testing.T) {
		db := setupResourceAddressTestDB(t)
		repo := NewRedirectRepository(db)

		got, err := repo.FindAddressesByIDs(context.Background(), nil, nil, 500)

		assert.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("successInBatches", func(t *testing.T) {
		db := setupResourceAddressTestDB(t)
		repo := NewRedirectRepository(db)

		ids := make([]int64, 0, 5)
		for _, source := range []string{"/1", "/2", "/3", "/4", "/5"} {
			ids = append(ids, createAddressRedirect(t, db, source).ID)
		}

		// A single IN clause holding one placeholder per row breaks past the database
		// placeholder limit, which a large publish reaches.
		got, err := repo.FindAddressesByIDs(context.Background(), nil, ids, 2)

		assert.NoError(t, err)
		assert.Len(t, got, 5)
	})

	t.Run("successWithoutBatchSize", func(t *testing.T) {
		db := setupResourceAddressTestDB(t)
		repo := NewRedirectRepository(db)
		redirect := createAddressRedirect(t, db, "/a")

		got, err := repo.FindAddressesByIDs(context.Background(), nil, []int64{redirect.ID}, 0)

		assert.NoError(t, err)
		assert.Len(t, got, 1)
	})

	t.Run("successWithinATransaction", func(t *testing.T) {
		db := setupResourceAddressTestDB(t)
		repo := NewRedirectRepository(db)
		redirect := createAddressRedirect(t, db, "/a")

		err := db.Transaction(func(tx *gorm.DB) error {
			got, errFind := repo.FindAddressesByIDs(context.Background(), tx, []int64{redirect.ID}, 500)
			assert.NoError(t, errFind)
			assert.Len(t, got, 1)
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("successIgnoresUnknownIDs", func(t *testing.T) {
		db := setupResourceAddressTestDB(t)
		repo := NewRedirectRepository(db)

		got, err := repo.FindAddressesByIDs(context.Background(), nil, []int64{999}, 500)

		assert.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestPageRepository_FindAddressesByIDs(t *testing.T) {
	t.Run("successReadsThePathWithoutTheContent", func(t *testing.T) {
		db := setupResourceAddressTestDB(t)
		repo := NewPageRepository(db)

		page := &model.Page{
			NamespaceCode: "ns1",
			ProjectCode:   "proj1",
			IsPublished:   types.Ptr(true),
			Page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        "/robots.txt",
				Content:     "User-agent: *",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		assert.NoError(t, db.Create(page).Error)

		got, err := repo.FindAddressesByIDs(context.Background(), nil, []int64{page.ID}, 500)

		assert.NoError(t, err)
		assert.Equal(t, []model.ResourceAddress{{ID: page.ID, Type: "BASIC", Source: "/robots.txt"}}, got)
	})

	t.Run("successWithoutIDs", func(t *testing.T) {
		db := setupResourceAddressTestDB(t)
		repo := NewPageRepository(db)

		got, err := repo.FindAddressesByIDs(context.Background(), nil, nil, 500)

		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}

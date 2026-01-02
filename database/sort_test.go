package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSortTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	return db
}

func TestApplySort(t *testing.T) {
	allowedColumns := map[string]string{
		"name":      "name",
		"createdAt": "created_at",
		"updatedAt": "updated_at",
		"code":      "code",
	}

	t.Run("empty sorts returns query unchanged", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		result := ApplySort(query, allowedColumns, []SortInput{}, "")

		assert.NotNil(t, result)
	})

	t.Run("nil sorts returns query unchanged", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		result := ApplySort(query, allowedColumns, nil, "")

		assert.NotNil(t, result)
	})

	t.Run("single sort ASC", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		sorts := []SortInput{
			{Column: "name", Direction: SortASC},
		}

		result := ApplySort(query, allowedColumns, sorts, "")

		stmt := result.Statement
		result.Statement.Build("ORDER BY")
		assert.Contains(t, stmt.SQL.String(), "name ASC")
	})

	t.Run("single sort DESC", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		sorts := []SortInput{
			{Column: "name", Direction: SortDESC},
		}

		result := ApplySort(query, allowedColumns, sorts, "")

		stmt := result.Statement
		result.Statement.Build("ORDER BY")
		assert.Contains(t, stmt.SQL.String(), "name DESC")
	})

	t.Run("multiple sorts", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		sorts := []SortInput{
			{Column: "name", Direction: SortASC},
			{Column: "createdAt", Direction: SortDESC},
		}

		result := ApplySort(query, allowedColumns, sorts, "")

		stmt := result.Statement
		result.Statement.Build("ORDER BY")
		sql := stmt.SQL.String()
		assert.Contains(t, sql, "name ASC")
		assert.Contains(t, sql, "created_at DESC")
	})

	t.Run("invalid column is ignored", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		sorts := []SortInput{
			{Column: "invalid_column", Direction: SortASC},
		}

		result := ApplySort(query, allowedColumns, sorts, "")

		stmt := result.Statement
		result.Statement.Build("ORDER BY")
		assert.NotContains(t, stmt.SQL.String(), "invalid_column")
	})

	t.Run("mixed valid and invalid columns", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		sorts := []SortInput{
			{Column: "name", Direction: SortASC},
			{Column: "invalid", Direction: SortDESC},
			{Column: "code", Direction: SortDESC},
		}

		result := ApplySort(query, allowedColumns, sorts, "")

		stmt := result.Statement
		result.Statement.Build("ORDER BY")
		sql := stmt.SQL.String()
		assert.Contains(t, sql, "name ASC")
		assert.Contains(t, sql, "code DESC")
		assert.NotContains(t, sql, "invalid")
	})

	t.Run("with table prefix", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		sorts := []SortInput{
			{Column: "name", Direction: SortASC},
		}

		result := ApplySort(query, allowedColumns, sorts, "users")

		stmt := result.Statement
		result.Statement.Build("ORDER BY")
		assert.Contains(t, stmt.SQL.String(), "users.name ASC")
	})

	t.Run("maps json column name to db column name", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		sorts := []SortInput{
			{Column: "createdAt", Direction: SortASC},
		}

		result := ApplySort(query, allowedColumns, sorts, "")

		stmt := result.Statement
		result.Statement.Build("ORDER BY")
		assert.Contains(t, stmt.SQL.String(), "created_at ASC")
	})

	t.Run("default direction is ASC for invalid direction", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		sorts := []SortInput{
			{Column: "name", Direction: SortDirection("INVALID")},
		}

		result := ApplySort(query, allowedColumns, sorts, "")

		stmt := result.Statement
		result.Statement.Build("ORDER BY")
		assert.Contains(t, stmt.SQL.String(), "name ASC")
	})

	t.Run("empty allowed columns ignores all sorts", func(t *testing.T) {
		db := setupSortTestDB(t)
		query := db.Model(&struct{}{})

		sorts := []SortInput{
			{Column: "name", Direction: SortASC},
		}

		result := ApplySort(query, map[string]string{}, sorts, "")

		stmt := result.Statement
		result.Statement.Build("ORDER BY")
		assert.NotContains(t, stmt.SQL.String(), "name")
	})
}

package model

import (
	"time"

	"github.com/flectolab/flecto-manager/common/types"
)

const (
	ColumnProjectCode = "project_code"
)

var ProjectSortableColumns = map[string]string{
	"id":             "id",
	"namespace_code": ColumnNamespaceCode,
	"code":           ColumnProjectCode,
	"name":           "name",
	"createdAt":      "created_at",
	"updatedAt":      "updated_at",
}

type Project struct {
	ID            int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	ProjectCode   string     `json:"code" gorm:"size:50;uniqueIndex:idx_project_namespace" validate:"required,code"`
	NamespaceCode string     `json:"-" gorm:"size:50;uniqueIndex:idx_project_namespace;index:idx_namespace"`
	Namespace     *Namespace `json:"namespace" gorm:"foreignKey:NamespaceCode;references:NamespaceCode;"`
	Name          string     `json:"name" validate:"required"`
	Version       int        `json:"version" gorm:"default:1"`
	CreatedAt     time.Time  `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt     time.Time  `json:"UpdatedAt" gorm:"type:timestamp"`
	PublishedAt   time.Time  `json:"publishedAt" gorm:"type:timestamp"`
}

type ProjectList = types.PaginatedResult[Project]

package model

import (
	"time"

	"github.com/flectolab/flecto-manager/common/types"
)

const (
	ColumnNamespaceCode = "namespace_code"
)

var NamespaceSortableColumns = map[string]string{
	"id":             "id",
	"namespace_code": ColumnNamespaceCode,
	"name":           "name",
	"createdAt":      "created_at",
	"updatedAt":      "updated_at",
}

type Namespace struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string    `json:"namespace_code" gorm:"size:50;uniqueIndex:idx_namespace_namespace_code;" validate:"required,code"`
	Name          string    `json:"name" validate:"required"`
	CreatedAt     time.Time `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt     time.Time `json:"updatedAt" gorm:"type:timestamp"`
}

type NamespaceList = types.PaginatedResult[Namespace]

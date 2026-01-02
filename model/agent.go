package model

import (
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
)

var AgentSortableColumns = map[string]string{
	"name":      "name",
	"status":    "status",
	"type":      "type",
	"createAt":  "created_at",
	"updatedAt": "updated_at",
	"lastHitAt": "last_hit_at",
}

type Agent struct {
	ID            int64    `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string   `json:"-" gorm:"size:50;index:idx_pages_namespace_project"`
	ProjectCode   string   `json:"-" gorm:"size:50;index:idx_pages_namespace_project"`
	Project       *Project `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;"`
	commonTypes.Agent
	CreatedAt time.Time `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"type:timestamp"`
	LastHitAt time.Time `json:"lastHitAt" gorm:"type:timestamp"`
}

type AgentList = commonTypes.PaginatedResult[Agent]

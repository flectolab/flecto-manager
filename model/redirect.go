package model

import (
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
)

const (
	DraftChangeTypeCreate    DraftChangeType = "CREATE"
	DraftChangeTypeUpdate    DraftChangeType = "UPDATE"
	DraftChangeTypeDelete    DraftChangeType = "DELETE"
	DraftChangeTypePublished DraftChangeType = "PUBLISHED"
)

type DraftChangeType string

var RedirectSortableColumns = map[string]string{
	"source":    "source",
	"target":    "target",
	"type":      "type",
	"status":    "status",
	"updatedAt": "updated_at",
}

type Redirect struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string    `json:"-" gorm:"size:50;index:idx_redirects_namespace_project"`
	ProjectCode   string    `json:"-" gorm:"size:50;index:idx_redirects_namespace_project"`
	Project       *Project  `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;"`
	IsPublished   *bool     `json:"is_published" gorm:"default:false;not null"`
	PublishedAt   time.Time `json:"publishedAt" gorm:"type:timestamp"`
	*commonTypes.Redirect
	RedirectDraft *RedirectDraft `json:"draft" gorm:"foreignKey:OldRedirectID;references:ID"`
	CreatedAt     time.Time      `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt     time.Time      `json:"updatedAt" gorm:"type:timestamp"`
}

type RedirectList = commonTypes.PaginatedResult[Redirect]

type RedirectDraft struct {
	ID            int64                 `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string                `json:"-" gorm:"size:50;index:idx_redirect_drafts_namespace_project"`
	ProjectCode   string                `json:"-" gorm:"size:50;index:idx_redirect_drafts_namespace_project"`
	Project       *Project              `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;"`
	ChangeType    DraftChangeType       `json:"changeType" gorm:"size:50;" validate:"required"`
	OldRedirectID *int64                `json:"-" gorm:"index:idx_redirect_drafts_old_redirect_id"`
	OldRedirect   *Redirect             `json:"oldRedirect" gorm:"foreignKey:OldRedirectID;"`
	NewRedirect   *commonTypes.Redirect `gorm:"embedded;embeddedPrefix:new_"`
	CreatedAt     time.Time             `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt     time.Time             `json:"updatedAt" gorm:"type:timestamp"`
}

type RedirectDraftList = commonTypes.PaginatedResult[RedirectDraft]

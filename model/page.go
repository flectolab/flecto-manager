package model

import (
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
)

var PageSortableColumns = map[string]string{
	"path":        "path",
	"contentType": "content_type",
	"type":        "type",
	"updatedAt":   "updated_at",
}

type Page struct {
	ID            int64    `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string   `json:"-" gorm:"size:255;index:idx_pages_namespace_project"`
	ProjectCode   string   `json:"-" gorm:"size:255;index:idx_pages_namespace_project"`
	Project       *Project `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;-:migration"`
	IsPublished   *bool    `json:"is_published" gorm:"default:false;not null"`
	ContentSize   int64    `json:"contentSize" gorm:"default:0;not null"`
	*commonTypes.Page
	PageDraft *PageDraft `json:"draft" gorm:"foreignKey:OldPageID;references:ID"`
	CreatedAt time.Time  `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt time.Time  `json:"updatedAt" gorm:"type:timestamp"`
}

type PageList = commonTypes.PaginatedResult[Page]

type PageDraft struct {
	ID            int64             `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string            `json:"-" gorm:"size:255;index:idx_page_drafts_namespace_project"`
	ProjectCode   string            `json:"-" gorm:"size:255;index:idx_page_drafts_namespace_project"`
	Project       *Project          `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;-:migration"`
	ChangeType    DraftChangeType   `json:"changeType" validate:"required"`
	OldPageID     *int64            `json:"-" gorm:"index:idx_page_drafts_old_page_id"`
	OldPage       *Page             `json:"oldPage" gorm:"foreignKey:OldPageID;references:ID;constraint:OnDelete:CASCADE;"`
	ContentSize   int64             `json:"contentSize" gorm:"default:0;not null"`
	NewPage       *commonTypes.Page `gorm:"embedded;embeddedPrefix:new_"`
	CreatedAt     time.Time         `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt     time.Time         `json:"updatedAt" gorm:"type:timestamp"`
}

type PageDraftList = commonTypes.PaginatedResult[PageDraft]

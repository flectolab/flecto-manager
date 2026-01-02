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
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string    `json:"-" gorm:"size:50;index:idx_pages_namespace_project"`
	ProjectCode   string    `json:"-" gorm:"size:50;index:idx_pages_namespace_project"`
	Project       *Project  `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;"`
	IsPublished   *bool     `json:"is_published" gorm:"default:false;not null"`
	PublishedAt   time.Time `json:"publishedAt" gorm:"type:timestamp"`
	ContentSize   int64     `json:"contentSize" gorm:"default:0;not null"`
	*commonTypes.Page
	PageDraft *PageDraft `json:"draft" gorm:"foreignKey:OldPageID;references:ID"`
	CreatedAt time.Time  `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt time.Time  `json:"updatedAt" gorm:"type:timestamp"`
}

type PageList = commonTypes.PaginatedResult[Page]

type PageDraft struct {
	ID            int64             `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string            `json:"-" gorm:"size:50;index:idx_page_drafts_namespace_project"`
	ProjectCode   string            `json:"-" gorm:"size:50;index:idx_page_drafts_namespace_project"`
	Project       *Project          `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;"`
	ChangeType    DraftChangeType   `json:"changeType" gorm:"size:50;" validate:"required"`
	OldPageID     *int64            `json:"-" gorm:"index:idx_page_drafts_old_page_id"`
	OldPage       *Page             `json:"oldPage" gorm:"foreignKey:OldPageID;"`
	ContentSize   int64             `json:"contentSize" gorm:"default:0;not null"`
	NewPage       *commonTypes.Page `gorm:"embedded;embeddedPrefix:new_"`
	CreatedAt     time.Time         `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt     time.Time         `json:"updatedAt" gorm:"type:timestamp"`
}

type PageDraftList = commonTypes.PaginatedResult[PageDraft]

package model

import (
	"time"

	"github.com/flectolab/flecto-manager/common/types"
)

const (
	TokenPrefix        = "flecto_"
	TokenNameMaxLength = 300
	TokenPreviewChars  = 4 // Number of characters to show at start and end of preview
)

var TokenSortableColumns = map[string]string{
	"id":        "id",
	"name":      "name",
	"createdAt": "created_at",
	"updatedAt": "updated_at",
	"expiresAt": "expires_at",
}

type Token struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string     `json:"name" gorm:"uniqueIndex;size:300;not null" validate:"required,max=300"`
	TokenHash    string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	TokenPreview string     `json:"tokenPreview" gorm:"size:30;not null"` // e.g., "flecto_abcd...wxyz"
	ExpiresAt    *time.Time `json:"expiresAt" gorm:"type:timestamp"`
	CreatedAt    time.Time  `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt time.Time  `json:"updatedAt" gorm:"type:timestamp"`
}

type TokenList = types.PaginatedResult[Token]

// IsExpired checks if the token has expired
func (t *Token) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}

// GetRoleCode returns the role code for this token's personal role
func (t *Token) GetRoleCode() string {
	return "token_" + t.Name
}

// GenerateTokenPreview creates a preview string like "flecto_abcd...wxyz" from the full token
func GenerateTokenPreview(plainToken string) string {
	if len(plainToken) <= len(TokenPrefix)+TokenPreviewChars*2 {
		return plainToken
	}
	start := plainToken[:len(TokenPrefix)+TokenPreviewChars]
	end := plainToken[len(plainToken)-TokenPreviewChars:]
	return start + "..." + end
}

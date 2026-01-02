package model

import (
	"time"

	"github.com/flectolab/flecto-manager/common/types"
)

var UserSortableColumns = map[string]string{
	"id":        "id",
	"username":  "username",
	"lastname":  "lastname",
	"firstname": "firstname",
	"createdAt": "created_at",
	"updatedAt": "updated_at",
}

type User struct {
	ID               int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Username         string    `json:"username" gorm:"unique;size:100;not null" validate:"required,code"`
	Password         string    `json:"-" gorm:"size:255"`
	Lastname         string    `json:"lastname"  validate:"required"`
	Firstname        string    `json:"firstname"  validate:"required"`
	Active           *bool     `json:"active" gorm:"default:true;not null"`
	RefreshTokenHash string    `json:"-" gorm:"size:255"`
	CreatedAt        time.Time `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt        time.Time `json:"updatedAt" gorm:"type:timestamp"`
}

func (u *User) IsActive() bool {
	return u.Active != nil && *u.Active
}

// HasPassword returns true if the user can use basic auth
func (u *User) HasPassword() bool {
	return u.Password != ""
}

type UserList = types.PaginatedResult[User]

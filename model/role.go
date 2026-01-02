package model

import (
	"regexp"
	"time"

	"github.com/flectolab/flecto-manager/common/types"
)

var ValidRoleNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type RoleType string

const (
	RoleTypeUser  RoleType = "user"
	RoleTypeRole  RoleType = "role"
	RoleTypeToken RoleType = "token"
)

var RoleSortableColumns = map[string]string{
	"id":        "id",
	"code":      "code",
	"type":      "type",
	"createdAt": "created_at",
	"updatedAt": "updated_at",
}

type Role struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Code      string    `json:"code" gorm:"uniqueIndex:idx_role_code_type;size:100;not null" validate:"required,code"`
	Type      RoleType  `json:"type" gorm:"uniqueIndex:idx_role_code_type;size:100;not null"`
	CreatedAt time.Time `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"type:timestamp"`

	Users []User `json:"users,omitempty" gorm:"many2many:user_roles;"`

	Resources []ResourcePermission `json:"resources,omitempty" gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE;"`
	Admin     []AdminPermission    `json:"admin,omitempty" gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE;"`
}

type UserRole struct {
	UserID    int64     `json:"userId" gorm:"primaryKey"`
	RoleID    int64     `json:"roleId" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"type:timestamp"`

	User User `json:"user" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	Role Role `json:"role" gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE;"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

type RoleList = types.PaginatedResult[Role]

type SubjectPermissions struct {
	Resources []ResourcePermission `json:"resources,omitempty"`
	Admin     []AdminPermission    `json:"admin,omitempty"`
}

func (s *SubjectPermissions) Append(permission *SubjectPermissions) {
	if len(permission.Resources) > 0 {
		s.Resources = append(s.Resources, permission.Resources...)
	}
	if len(permission.Admin) > 0 {
		s.Admin = append(s.Admin, permission.Admin...)
	}
}

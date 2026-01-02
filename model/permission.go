package model

import "time"

type SectionType string
type ActionType string

const (
	AdminSectionUsers      SectionType = "users"
	AdminSectionRoles      SectionType = "roles"
	AdminSectionProjects   SectionType = "projects"
	AdminSectionNamespaces SectionType = "namespaces"
	AdminSectionAll        SectionType = "*"

	ActionRead   ActionType = "read"
	ActionWrite  ActionType = "write"
	ActionDelete ActionType = "delete"
	ActionAll    ActionType = "*"
)

type ResourcePermission struct {
	ID        int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	Namespace string     `json:"namespace" gorm:"size:100;not null;index:idx_res_perm_namespace"`
	Project   string     `json:"project" gorm:"size:100;index:idx_res_perm_project"`
	Action    ActionType `json:"action" gorm:"size:50;not null"`
	RoleID    int64
	Role      Role      `json:"role,omitempty" gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE;"`
	CreatedAt time.Time `json:"createdAt" gorm:"type:timestamp"`
}

func (ResourcePermission) TableName() string {
	return "resource_permissions"
}

type AdminPermission struct {
	ID        int64       `json:"id" gorm:"primaryKey;autoIncrement"`
	Section   SectionType `json:"section" gorm:"size:100;not null;index:idx_admin_perm_section"`
	Action    ActionType  `json:"action" gorm:"size:50;not null"`
	RoleID    int64
	Role      Role      `json:"role,omitempty" gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE;"`
	CreatedAt time.Time `json:"createdAt" gorm:"type:timestamp"`
}

func (AdminPermission) TableName() string {
	return "admin_permissions"
}

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidRoleNameRegex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "alphanumeric only",
			input: "admin123",
			want:  true,
		},
		{
			name:  "with underscore",
			input: "admin_role",
			want:  true,
		},
		{
			name:  "with hyphen",
			input: "admin-role",
			want:  true,
		},
		{
			name:  "mixed valid characters",
			input: "Admin_Role-123",
			want:  true,
		},
		{
			name:  "with space is invalid",
			input: "admin role",
			want:  false,
		},
		{
			name:  "with special char is invalid",
			input: "admin@role",
			want:  false,
		},
		{
			name:  "empty string is invalid",
			input: "",
			want:  false,
		},
		{
			name:  "with dot is invalid",
			input: "admin.role",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidRoleNameRegex.MatchString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoleTypeConstants(t *testing.T) {
	assert.Equal(t, RoleType("user"), RoleTypeUser)
	assert.Equal(t, RoleType("role"), RoleTypeRole)
	assert.Equal(t, RoleType("token"), RoleTypeToken)
}

func TestRoleSortableColumns(t *testing.T) {
	expected := map[string]string{
		"id":        "id",
		"code":      "code",
		"type":      "type",
		"createdAt": "created_at",
		"updatedAt": "updated_at",
	}
	assert.Equal(t, expected, RoleSortableColumns)
}

func TestUserRole_TableName(t *testing.T) {
	userRole := UserRole{}
	assert.Equal(t, "user_roles", userRole.TableName())
}

func TestSubjectPermissions_Append(t *testing.T) {
	tests := []struct {
		name           string
		initial        *SubjectPermissions
		toAppend       *SubjectPermissions
		wantResources  int
		wantAdmin      int
	}{
		{
			name: "append to empty permissions",
			initial: &SubjectPermissions{
				Resources: []ResourcePermission{},
				Admin:     []AdminPermission{},
			},
			toAppend: &SubjectPermissions{
				Resources: []ResourcePermission{{Namespace: "ns1"}},
				Admin:     []AdminPermission{{Section: "users"}},
			},
			wantResources: 1,
			wantAdmin:     1,
		},
		{
			name: "append to existing permissions",
			initial: &SubjectPermissions{
				Resources: []ResourcePermission{{Namespace: "ns1"}},
				Admin:     []AdminPermission{{Section: "users"}},
			},
			toAppend: &SubjectPermissions{
				Resources: []ResourcePermission{{Namespace: "ns2"}, {Namespace: "ns3"}},
				Admin:     []AdminPermission{{Section: "roles"}},
			},
			wantResources: 3,
			wantAdmin:     2,
		},
		{
			name: "append empty permissions",
			initial: &SubjectPermissions{
				Resources: []ResourcePermission{{Namespace: "ns1"}},
				Admin:     []AdminPermission{{Section: "users"}},
			},
			toAppend: &SubjectPermissions{
				Resources: []ResourcePermission{},
				Admin:     []AdminPermission{},
			},
			wantResources: 1,
			wantAdmin:     1,
		},
		{
			name: "append nil slices",
			initial: &SubjectPermissions{
				Resources: []ResourcePermission{{Namespace: "ns1"}},
				Admin:     []AdminPermission{{Section: "users"}},
			},
			toAppend: &SubjectPermissions{
				Resources: nil,
				Admin:     nil,
			},
			wantResources: 1,
			wantAdmin:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.Append(tt.toAppend)
			assert.Len(t, tt.initial.Resources, tt.wantResources)
			assert.Len(t, tt.initial.Admin, tt.wantAdmin)
		})
	}
}

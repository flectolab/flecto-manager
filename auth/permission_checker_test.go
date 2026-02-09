package auth

import (
	"context"
	"errors"
	"testing"

	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPermissionCheckerTest(t *testing.T) (*gomock.Controller, *mockFlectoService.MockRoleService, *PermissionChecker) {
	ctrl := gomock.NewController(t)
	mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
	checker := NewPermissionChecker(mockRoleService)
	return ctrl, mockRoleService, checker
}

func TestNewPermissionChecker(t *testing.T) {
	ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, checker)
	assert.Equal(t, mockRoleService, checker.roleService)
}

func TestPermissionChecker_CanResource(t *testing.T) {
	ctrl, _, checker := setupPermissionCheckerTest(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		permissions *model.SubjectPermissions
		namespace   string
		project     string
		resource    model.ResourceType
		action      model.ActionType
		expected    bool
	}{
		{
			name: "exact match",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				},
			},
			namespace: "ns1",
			project:   "proj1",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionRead,
			expected:  true,
		},
		{
			name: "wildcard namespace",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				},
			},
			namespace: "any-ns",
			project:   "proj1",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionRead,
			expected:  true,
		},
		{
			name: "wildcard project",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "*", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				},
			},
			namespace: "ns1",
			project:   "any-proj",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionRead,
			expected:  true,
		},
		{
			name: "wildcard action",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionAll},
				},
			},
			namespace: "ns1",
			project:   "proj1",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionWrite,
			expected:  true,
		},
		{
			name: "all wildcards",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeAll, Action: model.ActionAll},
				},
			},
			namespace: "any-ns",
			project:   "any-proj",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionWrite,
			expected:  true,
		},
		{
			name: "all wildcards with any resource",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeRedirect, Action: model.ActionWrite},
				},
			},
			namespace: "any-ns",
			project:   "any-proj",
			resource:  model.ResourceTypeAny,
			action:    model.ActionWrite,
			expected:  true,
		},
		{
			name: "no match - wrong namespace",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				},
			},
			namespace: "ns2",
			project:   "proj1",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionRead,
			expected:  false,
		},
		{
			name: "no match - wrong project",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				},
			},
			namespace: "ns1",
			project:   "proj2",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionRead,
			expected:  false,
		},
		{
			name: "no match - wrong action",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				},
			},
			namespace: "ns1",
			project:   "proj1",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionWrite,
			expected:  false,
		},
		{
			name: "empty permissions",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{},
			},
			namespace: "ns1",
			project:   "proj1",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionRead,
			expected:  false,
		},
		{
			name: "multiple permissions - match second",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
					{Namespace: "ns2", Project: "proj2", Resource: model.ResourceTypeAll, Action: model.ActionWrite},
				},
			},
			namespace: "ns2",
			project:   "proj2",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionWrite,
			expected:  true,
		},
		{
			name: "resource type match - redirect",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeRedirect, Action: model.ActionRead},
				},
			},
			namespace: "ns1",
			project:   "proj1",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionRead,
			expected:  true,
		},
		{
			name: "resource type mismatch",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeRedirect, Action: model.ActionRead},
				},
			},
			namespace: "ns1",
			project:   "proj1",
			resource:  model.ResourceTypePage,
			action:    model.ActionRead,
			expected:  false,
		},
		{
			name: "wildcard resource matches specific",
			permissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				},
			},
			namespace: "ns1",
			project:   "proj1",
			resource:  model.ResourceTypeRedirect,
			action:    model.ActionRead,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.CanResource(tt.permissions, tt.namespace, tt.project, tt.resource, tt.action)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPermissionChecker_CanAdmin(t *testing.T) {
	ctrl, _, checker := setupPermissionCheckerTest(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		permissions *model.SubjectPermissions
		section     model.SectionType
		action      model.ActionType
		expected    bool
	}{
		{
			name: "exact match",
			permissions: &model.SubjectPermissions{
				Admin: []model.AdminPermission{
					{Section: model.AdminSectionUsers, Action: model.ActionRead},
				},
			},
			section:  model.AdminSectionUsers,
			action:   model.ActionRead,
			expected: true,
		},
		{
			name: "wildcard section",
			permissions: &model.SubjectPermissions{
				Admin: []model.AdminPermission{
					{Section: "*", Action: model.ActionRead},
				},
			},
			section:  model.AdminSectionRoles,
			action:   model.ActionRead,
			expected: true,
		},
		{
			name: "wildcard action",
			permissions: &model.SubjectPermissions{
				Admin: []model.AdminPermission{
					{Section: model.AdminSectionUsers, Action: model.ActionAll},
				},
			},
			section:  model.AdminSectionUsers,
			action:   model.ActionWrite,
			expected: true,
		},
		{
			name: "all wildcards",
			permissions: &model.SubjectPermissions{
				Admin: []model.AdminPermission{
					{Section: "*", Action: model.ActionAll},
				},
			},
			section:  model.AdminSectionProjects,
			action:   model.ActionWrite,
			expected: true,
		},
		{
			name: "no match - wrong section",
			permissions: &model.SubjectPermissions{
				Admin: []model.AdminPermission{
					{Section: model.AdminSectionUsers, Action: model.ActionRead},
				},
			},
			section:  model.AdminSectionRoles,
			action:   model.ActionRead,
			expected: false,
		},
		{
			name: "no match - wrong action",
			permissions: &model.SubjectPermissions{
				Admin: []model.AdminPermission{
					{Section: model.AdminSectionUsers, Action: model.ActionRead},
				},
			},
			section:  model.AdminSectionUsers,
			action:   model.ActionWrite,
			expected: false,
		},
		{
			name: "empty permissions",
			permissions: &model.SubjectPermissions{
				Admin: []model.AdminPermission{},
			},
			section:  model.AdminSectionUsers,
			action:   model.ActionRead,
			expected: false,
		},
		{
			name: "multiple permissions - match second",
			permissions: &model.SubjectPermissions{
				Admin: []model.AdminPermission{
					{Section: model.AdminSectionUsers, Action: model.ActionRead},
					{Section: model.AdminSectionRoles, Action: model.ActionWrite},
				},
			},
			section:  model.AdminSectionRoles,
			action:   model.ActionWrite,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.CanAdmin(tt.permissions, tt.section, tt.action)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPermissionChecker_CanResourceForUsername(t *testing.T) {
	t.Run("success - permission granted", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "testuser").
			Return(permissions, nil)

		result, err := checker.CanResourceForUsername(ctx, "testuser", "ns1", "proj1", model.ResourceTypeAll, model.ActionRead)

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("success - permission denied", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "testuser").
			Return(permissions, nil)

		result, err := checker.CanResourceForUsername(ctx, "testuser", "ns2", "proj1", model.ResourceTypeAll, model.ActionRead)

		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("error from service", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "testuser").
			Return(nil, expectedErr)

		result, err := checker.CanResourceForUsername(ctx, "testuser", "ns1", "proj1", model.ResourceTypeAll, model.ActionRead)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})
}

func TestPermissionChecker_CanAdminForUsername(t *testing.T) {
	t.Run("success - permission granted", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Admin: []model.AdminPermission{
				{Section: model.AdminSectionUsers, Action: model.ActionWrite},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "admin").
			Return(permissions, nil)

		result, err := checker.CanAdminForUsername(ctx, "admin", model.AdminSectionUsers, model.ActionWrite)

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("success - permission denied", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Admin: []model.AdminPermission{
				{Section: model.AdminSectionUsers, Action: model.ActionRead},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "user").
			Return(permissions, nil)

		result, err := checker.CanAdminForUsername(ctx, "user", model.AdminSectionUsers, model.ActionWrite)

		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("error from service", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("user not found")

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "unknown").
			Return(nil, expectedErr)

		result, err := checker.CanAdminForUsername(ctx, "unknown", model.AdminSectionUsers, model.ActionRead)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})
}

func TestPermissionChecker_CanResourceForRoleCode(t *testing.T) {
	t.Run("success - permission granted", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "*", Project: "*", Resource: model.ResourceTypeAll, Action: model.ActionAll},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "admin").
			Return(permissions, nil)

		result, err := checker.CanResourceForRoleCode(ctx, "admin", "any-ns", "any-proj", model.ResourceTypeAll, model.ActionWrite)

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("success - permission denied", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "reader").
			Return(permissions, nil)

		result, err := checker.CanResourceForRoleCode(ctx, "reader", "ns1", "proj1", model.ResourceTypeAll, model.ActionWrite)

		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("error from service", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("role not found")

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "unknown").
			Return(nil, expectedErr)

		result, err := checker.CanResourceForRoleCode(ctx, "unknown", "ns1", "proj1", model.ResourceTypeAll, model.ActionRead)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})
}

func TestPermissionChecker_CanAdminForRoleCode(t *testing.T) {
	t.Run("success - permission granted", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Admin: []model.AdminPermission{
				{Section: "*", Action: model.ActionAll},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "superadmin").
			Return(permissions, nil)

		result, err := checker.CanAdminForRoleCode(ctx, "superadmin", model.AdminSectionNamespaces, model.ActionWrite)

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("success - permission denied", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Admin: []model.AdminPermission{
				{Section: model.AdminSectionUsers, Action: model.ActionRead},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "viewer").
			Return(permissions, nil)

		result, err := checker.CanAdminForRoleCode(ctx, "viewer", model.AdminSectionRoles, model.ActionRead)

		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("error from service", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "broken").
			Return(nil, expectedErr)

		result, err := checker.CanAdminForRoleCode(ctx, "broken", model.AdminSectionUsers, model.ActionRead)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})
}

// --- Tests for Must methods ---

func TestPermissionChecker_MustCanResourceForUsername(t *testing.T) {
	t.Run("returns true when permission granted", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "testuser").
			Return(permissions, nil)

		result := checker.MustCanResourceForUsername(ctx, "testuser", "ns1", "proj1", model.ResourceTypeAll, model.ActionRead)

		assert.True(t, result)
	})

	t.Run("returns false on error", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "unknown").
			Return(nil, errors.New("user not found"))

		result := checker.MustCanResourceForUsername(ctx, "unknown", "ns1", "proj1", model.ResourceTypeAll, model.ActionRead)

		assert.False(t, result)
	})
}

func TestPermissionChecker_MustCanAdminForUsername(t *testing.T) {
	t.Run("returns true when permission granted", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Admin: []model.AdminPermission{
				{Section: model.AdminSectionUsers, Action: model.ActionWrite},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "admin").
			Return(permissions, nil)

		result := checker.MustCanAdminForUsername(ctx, "admin", model.AdminSectionUsers, model.ActionWrite)

		assert.True(t, result)
	})

	t.Run("returns false on error", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockRoleService.EXPECT().
			GetPermissionsByUsername(ctx, "unknown").
			Return(nil, errors.New("user not found"))

		result := checker.MustCanAdminForUsername(ctx, "unknown", model.AdminSectionUsers, model.ActionRead)

		assert.False(t, result)
	})
}

func TestPermissionChecker_MustCanResourceForRoleCode(t *testing.T) {
	t.Run("returns true when permission granted", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "*", Project: "*", Resource: model.ResourceTypeAll, Action: model.ActionAll},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "admin").
			Return(permissions, nil)

		result := checker.MustCanResourceForRoleCode(ctx, "admin", "any-ns", "any-proj", model.ResourceTypeAll, model.ActionWrite)

		assert.True(t, result)
	})

	t.Run("returns false on error", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "unknown").
			Return(nil, errors.New("role not found"))

		result := checker.MustCanResourceForRoleCode(ctx, "unknown", "ns1", "proj1", model.ResourceTypeAll, model.ActionRead)

		assert.False(t, result)
	})
}

func TestPermissionChecker_MustCanAdminForRoleCode(t *testing.T) {
	t.Run("returns true when permission granted", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Admin: []model.AdminPermission{
				{Section: "*", Action: model.ActionAll},
			},
		}

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "superadmin").
			Return(permissions, nil)

		result := checker.MustCanAdminForRoleCode(ctx, "superadmin", model.AdminSectionNamespaces, model.ActionWrite)

		assert.True(t, result)
	})

	t.Run("returns false on error", func(t *testing.T) {
		ctrl, mockRoleService, checker := setupPermissionCheckerTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockRoleService.EXPECT().
			GetPermissionsByRoleCode(ctx, "broken").
			Return(nil, errors.New("database error"))

		result := checker.MustCanAdminForRoleCode(ctx, "broken", model.AdminSectionUsers, model.ActionRead)

		assert.False(t, result)
	})
}

// --- Tests for Query Filtering methods ---

func TestPermissionChecker_filterPermissionsByAction(t *testing.T) {
	ctrl, _, checker := setupPermissionCheckerTest(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		permissions []model.ResourcePermission
		action      model.ActionType
		expected    int
	}{
		{
			name:        "empty permissions",
			permissions: []model.ResourcePermission{},
			action:      model.ActionRead,
			expected:    0,
		},
		{
			name: "exact action match",
			permissions: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
				{Namespace: "ns2", Project: "proj2", Action: model.ActionWrite},
			},
			action:   model.ActionRead,
			expected: 1,
		},
		{
			name: "wildcard action matches all",
			permissions: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Action: model.ActionAll},
			},
			action:   model.ActionWrite,
			expected: 1,
		},
		{
			name: "mixed actions",
			permissions: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
				{Namespace: "ns2", Project: "proj2", Action: model.ActionAll},
				{Namespace: "ns3", Project: "proj3", Action: model.ActionWrite},
			},
			action:   model.ActionRead,
			expected: 2, // ActionRead and ActionAll
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.filterPermissionsByAction(tt.permissions, tt.action)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestPermissionChecker_extractAllowedNamespaces(t *testing.T) {
	ctrl, _, checker := setupPermissionCheckerTest(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		permissions []model.ResourcePermission
		action      model.ActionType
		expected    []string // nil = no permissions, empty = full access
		isNil       bool
	}{
		{
			name:        "no permissions - returns nil",
			permissions: []model.ResourcePermission{},
			action:      model.ActionRead,
			expected:    nil,
			isNil:       true,
		},
		{
			name: "no matching action - returns nil",
			permissions: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Action: model.ActionWrite},
			},
			action:   model.ActionRead,
			expected: nil,
			isNil:    true,
		},
		{
			name: "wildcard namespace - returns empty slice for full access",
			permissions: []model.ResourcePermission{
				{Namespace: "*", Project: "proj1", Action: model.ActionRead},
			},
			action:   model.ActionRead,
			expected: []string{},
			isNil:    false,
		},
		{
			name: "specific namespaces - returns list",
			permissions: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
				{Namespace: "ns2", Project: "proj2", Action: model.ActionRead},
			},
			action:   model.ActionRead,
			expected: []string{"ns1", "ns2"},
			isNil:    false,
		},
		{
			name: "duplicate namespaces - returns unique list",
			permissions: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
				{Namespace: "ns1", Project: "proj2", Action: model.ActionRead},
				{Namespace: "ns2", Project: "proj3", Action: model.ActionRead},
			},
			action:   model.ActionRead,
			expected: []string{"ns1", "ns2"},
			isNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.extractAllowedNamespaces(tt.permissions, tt.action)
			if tt.isNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.ElementsMatch(t, tt.expected, result)
			}
		})
	}
}

func TestPermissionChecker_FilterQueryByNamespace(t *testing.T) {
	ctrl, _, checker := setupPermissionCheckerTest(t)
	defer ctrl.Finish()

	t.Run("no permissions - adds false condition", func(t *testing.T) {
		permissions := []model.ResourcePermission{}

		sql := toSQL(checker.FilterQueryByNamespace(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		assert.Contains(t, sql, "1 = 0")
	})

	t.Run("wildcard namespace - no filter added", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "*", Project: "proj1", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByNamespace(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		assert.NotContains(t, sql, "WHERE")
		assert.NotContains(t, sql, "1 = 0")
	})

	t.Run("specific namespaces - adds IN clause", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
			{Namespace: "ns2", Project: "proj2", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByNamespace(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		assert.Contains(t, sql, ColumnNamespaceCode+" IN")
	})

	t.Run("wrong action - adds false condition", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "proj1", Action: model.ActionWrite},
		}

		sql := toSQL(checker.FilterQueryByNamespace(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		assert.Contains(t, sql, "1 = 0")
	})
}

func TestPermissionChecker_FilterQueryByProject(t *testing.T) {
	ctrl, _, checker := setupPermissionCheckerTest(t)
	defer ctrl.Finish()

	t.Run("no permissions - adds false condition", func(t *testing.T) {
		permissions := []model.ResourcePermission{}

		sql := toSQL(checker.FilterQueryByProject(
			mockDB(),
			permissions,
			"ns1",
			model.ActionRead,
		))

		assert.Contains(t, sql, "1 = 0")
	})

	t.Run("wildcard namespace with specific project - filters by namespace and project", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "*", Project: "proj1", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByProject(
			mockDB(),
			permissions,
			"ns1",
			model.ActionRead,
		))

		assert.Contains(t, sql, ColumnNamespaceCode+" =")
		assert.Contains(t, sql, ColumnProjectCode+" IN")
	})

	t.Run("wildcard project for namespace - filters by namespace only", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "*", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByProject(
			mockDB(),
			permissions,
			"ns1",
			model.ActionRead,
		))

		assert.Contains(t, sql, ColumnNamespaceCode+" =")
		assert.NotContains(t, sql, ColumnProjectCode)
		assert.NotContains(t, sql, "1 = 0")
	})

	t.Run("wildcard namespace and wildcard project - filters by namespace only", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "*", Project: "*", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByProject(
			mockDB(),
			permissions,
			"ns1",
			model.ActionRead,
		))

		assert.Contains(t, sql, ColumnNamespaceCode+" =")
		assert.NotContains(t, sql, ColumnProjectCode)
		assert.NotContains(t, sql, "1 = 0")
	})

	t.Run("specific projects - filters by namespace and project", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
			{Namespace: "ns1", Project: "proj2", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByProject(
			mockDB(),
			permissions,
			"ns1",
			model.ActionRead,
		))

		assert.Contains(t, sql, ColumnNamespaceCode+" =")
		assert.Contains(t, sql, ColumnProjectCode+" IN")
	})

	t.Run("no access to namespace - adds false condition", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns2", Project: "proj1", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByProject(
			mockDB(),
			permissions,
			"ns1",
			model.ActionRead,
		))

		assert.Contains(t, sql, "1 = 0")
	})

	t.Run("wrong action - adds false condition", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "proj1", Action: model.ActionWrite},
		}

		sql := toSQL(checker.FilterQueryByProject(
			mockDB(),
			permissions,
			"ns1",
			model.ActionRead,
		))

		assert.Contains(t, sql, "1 = 0")
	})

	t.Run("mixed permissions - only matching namespace", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
			{Namespace: "ns2", Project: "proj2", Action: model.ActionRead}, // Different namespace
			{Namespace: "ns1", Project: "proj3", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByProject(
			mockDB(),
			permissions,
			"ns1",
			model.ActionRead,
		))

		assert.Contains(t, sql, ColumnNamespaceCode+" =")
		assert.Contains(t, sql, ColumnProjectCode+" IN")
	})
}

func TestPermissionChecker_FilterQueryByNamespaceProject(t *testing.T) {
	ctrl, _, checker := setupPermissionCheckerTest(t)
	defer ctrl.Finish()

	t.Run("no permissions - adds false condition", func(t *testing.T) {
		permissions := []model.ResourcePermission{}

		sql := toSQL(checker.FilterQueryByNamespaceProject(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		assert.Contains(t, sql, "1 = 0")
	})

	t.Run("wildcard namespace - no filter added", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "*", Project: "*", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByNamespaceProject(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		assert.NotContains(t, sql, "1 = 0")
	})

	t.Run("wildcard project - filters by namespace only", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "*", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByNamespaceProject(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		assert.Contains(t, sql, ColumnNamespaceCode+" IN")
	})

	t.Run("specific projects - filters by namespace and project", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
			{Namespace: "ns1", Project: "proj2", Action: model.ActionRead},
		}

		sql := toSQL(checker.FilterQueryByNamespaceProject(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		assert.Contains(t, sql, ColumnNamespaceCode+" =")
		assert.Contains(t, sql, ColumnProjectCode+" IN")
	})

	t.Run("mixed wildcard and specific - consolidates correctly", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "*", Action: model.ActionRead},     // Full access to ns1
			{Namespace: "ns1", Project: "proj1", Action: model.ActionRead}, // Should be ignored (ns1 has full)
			{Namespace: "ns2", Project: "proj2", Action: model.ActionRead}, // Specific project in ns2
			{Namespace: "ns2", Project: "proj3", Action: model.ActionRead}, // Another project in ns2
		}

		sql := toSQL(checker.FilterQueryByNamespaceProject(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		// ns1 should be in the full namespace access list
		// ns2 should have specific project filter
		assert.Contains(t, sql, ColumnNamespaceCode)
	})

	t.Run("wrong action - adds false condition", func(t *testing.T) {
		permissions := []model.ResourcePermission{
			{Namespace: "ns1", Project: "proj1", Action: model.ActionWrite},
		}

		sql := toSQL(checker.FilterQueryByNamespaceProject(
			mockDB(),
			permissions,
			model.ActionRead,
		))

		assert.Contains(t, sql, "1 = 0")
	})
}

// testDB is a test table for SQL generation tests
type testDB struct {
	ID            int64  `gorm:"primaryKey"`
	NamespaceCode string `gorm:"column:namespace_code"`
	ProjectCode   string `gorm:"column:project_code"`
}

func (testDB) TableName() string {
	return "test_table"
}

// mockDB creates a SQLite DB for testing SQL generation
func mockDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	return db.Model(&testDB{})
}

// toSQL generates the SQL string from a query using ToSQL
func toSQL(db *gorm.DB) string {
	return db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&[]testDB{})
	})
}

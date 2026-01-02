package project

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flectolab/flecto-manager/auth"
	"github.com/flectolab/flecto-manager/http/route"
	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/flectolab/flecto-manager/model"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetVersion(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockProjectService := mockFlectoService.NewMockProjectService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		project := &model.Project{
			ID:            1,
			NamespaceCode: "ns1",
			ProjectCode:   "proj1",
			Version:       42,
		}

		mockProjectService.EXPECT().
			GetByCode(gomock.Any(), "ns1", "proj1").
			Return(project, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects/ns1/proj1/version", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "proj1")

		// Set user context with permissions
		userCtx := &auth.UserContext{
			UserID:   1,
			Username: "testuser",
			SubjectPermissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := GetVersion(permissionChecker, mockProjectService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "42\n", rec.Body.String())
	})

	t.Run("missing namespace code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockProjectService := mockFlectoService.NewMockProjectService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects//proj1/version", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("", "proj1")

		handler := GetVersion(permissionChecker, mockProjectService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("missing project code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockProjectService := mockFlectoService.NewMockProjectService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects/ns1//version", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "")

		handler := GetVersion(permissionChecker, mockProjectService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("permission denied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockProjectService := mockFlectoService.NewMockProjectService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects/ns1/proj1/version", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "proj1")

		// Set user context without permissions
		userCtx := &auth.UserContext{
			UserID:             1,
			Username:           "testuser",
			SubjectPermissions: &model.SubjectPermissions{},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := GetVersion(permissionChecker, mockProjectService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockProjectService := mockFlectoService.NewMockProjectService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		mockProjectService.EXPECT().
			GetByCode(gomock.Any(), "ns1", "proj1").
			Return(nil, errors.New("database error"))

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects/ns1/proj1/version", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "proj1")

		// Set user context with permissions
		userCtx := &auth.UserContext{
			UserID:   1,
			Username: "testuser",
			SubjectPermissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := GetVersion(permissionChecker, mockProjectService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
	})
}

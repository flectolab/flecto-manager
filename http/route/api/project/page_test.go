package project

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flectolab/flecto-manager/auth"
	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/http/route"
	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/flectolab/flecto-manager/model"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetPages(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		pages := []model.Page{
			{
				ID:            1,
				NamespaceCode: "ns1",
				ProjectCode:   "proj1",
				Page:          &commonTypes.Page{Path: "/index.html", ContentType: commonTypes.PageContentTypeTextPlain},
			},
		}

		mockPageService.EXPECT().
			FindByProjectPublished(gomock.Any(), "ns1", "proj1", gomock.Any()).
			Return(pages, int64(1), nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects/ns1/proj1/pages", nil)
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
					{Namespace: "*", Project: "*", Resource: model.ResourceTypePage, Action: model.ActionRead},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := GetPages(permissionChecker, mockPageService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"Total":1`)
		assert.Contains(t, rec.Body.String(), `"/index.html"`)
		assert.Contains(t, rec.Body.String(), `"TEXT_PLAIN"`)
	})

	t.Run("success empty list", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		mockPageService.EXPECT().
			FindByProjectPublished(gomock.Any(), "ns1", "proj1", gomock.Any()).
			Return([]model.Page{}, int64(0), nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects/ns1/proj1/pages", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "proj1")

		userCtx := &auth.UserContext{
			UserID:   1,
			Username: "testuser",
			SubjectPermissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypePage, Action: model.ActionRead},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := GetPages(permissionChecker, mockPageService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"Total":0`)
	})

	t.Run("missing namespace code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects//proj1/pages", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("", "proj1")

		handler := GetPages(permissionChecker, mockPageService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("missing project code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects/ns1//pages", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "")

		handler := GetPages(permissionChecker, mockPageService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("permission denied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects/ns1/proj1/pages", nil)
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

		handler := GetPages(permissionChecker, mockPageService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		mockPageService.EXPECT().
			FindByProjectPublished(gomock.Any(), "ns1", "proj1", gomock.Any()).
			Return(nil, int64(0), errors.New("database error"))

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/projects/ns1/proj1/pages", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "proj1")

		userCtx := &auth.UserContext{
			UserID:   1,
			Username: "testuser",
			SubjectPermissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypePage, Action: model.ActionRead},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := GetPages(permissionChecker, mockPageService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
	})
}

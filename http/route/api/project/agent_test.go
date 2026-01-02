package project

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestPostAgent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		mockAgentService.EXPECT().
			Upsert(gomock.Any(), gomock.Any()).
			Return(nil)

		e := echo.New()
		body := `{"name":"test-agent","status":"success","type":"default","version":1}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/ns1/proj1/agents", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
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
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeAgent, Action: model.ActionWrite},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := PostAgent(permissionChecker, mockAgentService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing namespace code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		body := `{"name":"test-agent","status":"success","type":"default","version":1}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects//proj1/agents", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("", "proj1")

		handler := PostAgent(permissionChecker, mockAgentService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("missing project code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		body := `{"name":"test-agent","status":"success","type":"default","version":1}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/ns1//agents", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "")

		handler := PostAgent(permissionChecker, mockAgentService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("permission denied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		body := `{"name":"test-agent","status":"success","type":"default","version":1}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/ns1/proj1/agents", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
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

		handler := PostAgent(permissionChecker, mockAgentService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid json body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		body := `invalid json`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/ns1/proj1/agents", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "proj1")

		userCtx := &auth.UserContext{
			UserID:   1,
			Username: "testuser",
			SubjectPermissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeAgent, Action: model.ActionWrite},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := PostAgent(permissionChecker, mockAgentService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("validation error - missing name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		body := `{"status":"success","type":"default","version":1}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/ns1/proj1/agents", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "proj1")

		userCtx := &auth.UserContext{
			UserID:   1,
			Username: "testuser",
			SubjectPermissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeAgent, Action: model.ActionWrite},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := PostAgent(permissionChecker, mockAgentService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		mockAgentService.EXPECT().
			Upsert(gomock.Any(), gomock.Any()).
			Return(errors.New("database error"))

		e := echo.New()
		body := `{"name":"test-agent","status":"success","type":"default","version":1}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/ns1/proj1/agents", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey)
		c.SetParamValues("ns1", "proj1")

		userCtx := &auth.UserContext{
			UserID:   1,
			Username: "testuser",
			SubjectPermissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeAgent, Action: model.ActionWrite},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := PostAgent(permissionChecker, mockAgentService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
	})
}

func TestPatchAgentHit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		mockAgentService.EXPECT().
			UpdateLastHit(gomock.Any(), "ns1", "proj1", "agent1").
			Return(nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/projects/ns1/proj1/agents/agent1/hit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey, route.NameKey)
		c.SetParamValues("ns1", "proj1", "agent1")

		// Set user context with permissions
		userCtx := &auth.UserContext{
			UserID:   1,
			Username: "testuser",
			SubjectPermissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeAgent, Action: model.ActionWrite},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := PatchAgentHit(permissionChecker, mockAgentService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing namespace code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/projects//proj1/agents/agent1/hit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey, route.NameKey)
		c.SetParamValues("", "proj1", "agent1")

		handler := PatchAgentHit(permissionChecker, mockAgentService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("missing project code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/projects/ns1//agents/agent1/hit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey, route.NameKey)
		c.SetParamValues("ns1", "", "agent1")

		handler := PatchAgentHit(permissionChecker, mockAgentService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("missing name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/projects/ns1/proj1/agents//hit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey, route.NameKey)
		c.SetParamValues("ns1", "proj1", "")

		handler := PatchAgentHit(permissionChecker, mockAgentService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})

	t.Run("permission denied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/projects/ns1/proj1/agents/agent1/hit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey, route.NameKey)
		c.SetParamValues("ns1", "proj1", "agent1")

		// Set user context without permissions
		userCtx := &auth.UserContext{
			UserID:             1,
			Username:           "testuser",
			SubjectPermissions: &model.SubjectPermissions{},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := PatchAgentHit(permissionChecker, mockAgentService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAgentService := mockFlectoService.NewMockAgentService(ctrl)
		mockRoleService := mockFlectoService.NewMockRoleService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockRoleService)

		mockAgentService.EXPECT().
			UpdateLastHit(gomock.Any(), "ns1", "proj1", "agent1").
			Return(errors.New("database error"))

		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/projects/ns1/proj1/agents/agent1/hit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames(route.NamespaceCodeKey, route.ProjectCodeKey, route.NameKey)
		c.SetParamValues("ns1", "proj1", "agent1")

		userCtx := &auth.UserContext{
			UserID:   1,
			Username: "testuser",
			SubjectPermissions: &model.SubjectPermissions{
				Resources: []model.ResourcePermission{
					{Namespace: "*", Project: "*", Resource: model.ResourceTypeAgent, Action: model.ActionWrite},
				},
			},
		}
		ctx := auth.SetUserContext(req.Context(), userCtx)
		c.SetRequest(req.WithContext(ctx))

		handler := PatchAgentHit(permissionChecker, mockAgentService)
		err := handler(c)

		require.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
	})
}

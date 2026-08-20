package project

import (
	"context"
	"encoding/json"
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
			FindByProjectPublished(gomock.Any(), "ns1", "proj1", gomock.Any(), gomock.Any()).
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
			FindByProjectPublished(gomock.Any(), "ns1", "proj1", gomock.Any(), gomock.Any()).
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
			FindByProjectPublished(gomock.Any(), "ns1", "proj1", gomock.Any(), gomock.Any()).
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

func TestGetPagesCursor(t *testing.T) {
	newContext := func(e *echo.Echo, target string) (echo.Context, *httptest.ResponseRecorder) {
		req := httptest.NewRequest(http.MethodGet, target, nil)
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
		c.SetRequest(req.WithContext(auth.SetUserContext(req.Context(), userCtx)))
		return c, rec
	}

	pages := func(count int, firstID int64) []model.Page {
		items := make([]model.Page, 0, count)
		for i := 0; i < count; i++ {
			items = append(items, model.Page{
				ID:            firstID + int64(i),
				NamespaceCode: "ns1",
				ProjectCode:   "proj1",
				Page:          &commonTypes.Page{Path: "/robots.txt"},
			})
		}
		return items
	}

	t.Run("a full page hands out a cursor", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockFlectoService.NewMockRoleService(ctrl))

		mockPageService.EXPECT().
			FindByProjectPublished(gomock.Any(), "ns1", "proj1", gomock.Any(), nil).
			Return(pages(2, 10), int64(7), nil)

		c, rec := newContext(echo.New(), "/?limit=2")
		require.NoError(t, GetPages(permissionChecker, mockPageService)(c))

		var body commonTypes.PageList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, 7, body.Total)
		assert.Equal(t, 0, body.Offset)
		require.NotEmpty(t, body.Next)

		cursor, err := route.DecodeCursor(body.Next)
		require.NoError(t, err)
		// The cursor carries the last id of the page, the total measured once, and how
		// many rows have been handed out so far.
		assert.Equal(t, route.Cursor{AfterID: 11, Total: 7, Delivered: 2}, cursor)
	})

	t.Run("a short page ends the walk", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockFlectoService.NewMockRoleService(ctrl))

		mockPageService.EXPECT().
			FindByProjectPublished(gomock.Any(), "ns1", "proj1", gomock.Any(), nil).
			Return(pages(1, 10), int64(1), nil)

		c, rec := newContext(echo.New(), "/?limit=2")
		require.NoError(t, GetPages(permissionChecker, mockPageService)(c))

		var body commonTypes.PageList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Empty(t, body.Next)
		assert.False(t, body.HasMore())
	})

	t.Run("a cursor page reports the total and offset it carries", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockFlectoService.NewMockRoleService(ctrl))

		encoded, err := route.EncodeCursor(route.Cursor{AfterID: 11, Total: 7, Delivered: 2})
		require.NoError(t, err)

		var gotAfterID *int64
		mockPageService.EXPECT().
			FindByProjectPublished(gomock.Any(), "ns1", "proj1", gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _, _ string, _ *commonTypes.PaginationInput, afterID *int64) ([]model.Page, int64, error) {
				gotAfterID = afterID
				// The service does not count in cursor mode, so it reports no total.
				return pages(2, 12), 0, nil
			})

		c, rec := newContext(echo.New(), "/?limit=2&cursor="+encoded)
		require.NoError(t, GetPages(permissionChecker, mockPageService)(c))

		require.NotNil(t, gotAfterID)
		assert.Equal(t, int64(11), *gotAfterID)

		var body commonTypes.PageList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		// Total and Offset come from the cursor, so HasMore keeps working for clients
		// that never look at Next.
		assert.Equal(t, 7, body.Total)
		assert.Equal(t, 2, body.Offset)
		assert.True(t, body.HasMore())

		next, err := route.DecodeCursor(body.Next)
		require.NoError(t, err)
		assert.Equal(t, route.Cursor{AfterID: 13, Total: 7, Delivered: 4}, next)
	})

	t.Run("a malformed cursor is a client error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockPageService := mockFlectoService.NewMockPageService(ctrl)
		permissionChecker := auth.NewPermissionChecker(mockFlectoService.NewMockRoleService(ctrl))

		c, _ := newContext(echo.New(), "/?cursor=not-a-cursor")
		err := GetPages(permissionChecker, mockPageService)(c)

		require.Error(t, err)
		httpErr := &echo.HTTPError{}
		require.ErrorAs(t, err, &httpErr)
		// Restarting the listing from the top would loop forever, so the request is
		// refused rather than silently falling back to offset pagination.
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})
}

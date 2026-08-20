package project

import (
	"fmt"
	"net/http"

	"github.com/flectolab/flecto-manager/auth"
	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/http/route"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/labstack/echo/v4"
)

func GetPages(permissionChecker *auth.PermissionChecker, pageService service.PageService) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		namespaceCode := c.Param(route.NamespaceCodeKey)
		projectCode := c.Param(route.ProjectCodeKey)
		if namespaceCode == "" || projectCode == "" {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("namespaceCode and projectCode are required"))
		}
		userCtx := auth.GetUser(ctx)
		if !permissionChecker.CanResource(userCtx.SubjectPermissions, namespaceCode, projectCode, model.ResourceTypePage, model.ActionRead) {
			return c.NoContent(http.StatusForbidden)
		}
		pagination := &commonTypes.PaginationInput{Limit: types.Ptr(500), Offset: types.Ptr(0)}
		err := c.Bind(pagination)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		// A cursor replaces the offset: the listing walks from a position rather than
		// skipping rows, and the total travels inside the cursor so it is measured
		// once instead of on every page.
		var cursor *route.Cursor
		var afterID *int64
		if pagination.GetCursor() != "" {
			decoded, errCursor := route.DecodeCursor(pagination.GetCursor())
			if errCursor != nil {
				return echo.NewHTTPError(http.StatusBadRequest, errCursor)
			}
			cursor = &decoded
			afterID = &decoded.AfterID
		}

		pagesDB, total, err := pageService.FindByProjectPublished(ctx, namespaceCode, projectCode, pagination, afterID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		pages := make([]commonTypes.Page, 0)
		for _, page := range pagesDB {
			pages = append(pages, *page.Page)
		}
		var lastID int64
		if len(pagesDB) > 0 {
			lastID = pagesDB[len(pagesDB)-1].ID
		}
		listPage, err := route.NewListPage(pagination, cursor, total, len(pagesDB), lastID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		pageList := &commonTypes.PageList{
			Total:  listPage.Total,
			Offset: listPage.Offset,
			Limit:  listPage.Limit,
			Next:   listPage.Next,
			Items:  pages,
		}
		return c.JSON(http.StatusOK, pageList)
	}
}

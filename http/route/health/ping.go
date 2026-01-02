package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func GetPing() func(echo.Context) error {
	return func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}
}

package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health/ping", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := GetPing()
	err := handler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

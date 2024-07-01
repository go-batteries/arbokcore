package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type ShareHandler struct {
}

func (sh *ShareHandler) ShareFile(c echo.Context) error {
	return c.NoContent(http.StatusForbidden)
}

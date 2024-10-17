package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/services/utils"
	"github.com/nbittich/wtm/types"
)

func HomeRouter(e *echo.Echo) {
	e.GET("/", homeHandler).Name = "home.root"
}

func homeHandler(c echo.Context) error {
	request := c.Request()
	return c.JSON(http.StatusOK, types.Message{Type: types.INFO, Message: utils.Translate(request.Context(), "home.hello")})
}

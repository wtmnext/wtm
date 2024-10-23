package admin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/services"
	"github.com/nbittich/wtm/types"
)

func AdminRouter(e *echo.Echo) {
	adminGroup := e.Group("/admin/users")
	adminGroup.POST("/new", newUserHandler).Name = "admin.users.New"
	adminGroup.GET("", listUserHandler).Name = "admin.users.List"
}

func listUserHandler(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	users, err := services.AllUsers(ctx, adminUser.Group, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, users)
}

func newUserHandler(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}

	newUserForm := types.NewUserForm{}
	if err = c.Bind(&newUserForm); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	c.Logger().Debug(newUserForm)
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	user, err := services.NewUser(ctx, &newUserForm, adminUser.Group)
	if err != nil {
		if err, ok := err.(types.InvalidFormError); ok {
			err.Form = newUserForm
			return c.JSON(http.StatusBadRequest, err)
		}
		c.Logger().Error("Unexpected error when creating a new user:", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error while creating new user")
	}
	c.Logger().Debug(user)

	return c.JSON(http.StatusOK, user)
}

package user

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/services"
)

func UserPlanningRoute(e *echo.Echo) {
	planningGroup := e.Group("/planning")
	planningGroup.GET("/assignments", getPlanningAssignments).Name = "user.planning.GetAssignments"
}

func getPlanningAssignments(c echo.Context) error {
	user, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf(" user not found in context"))
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	assignments, err := services.GetPlanningAssignments(ctx, user.ID, user.Group)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, assignments)
}

package admin

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/services/superadmin"
	"github.com/nbittich/wtm/types"
)

func SuperAdminRouter(e *echo.Echo) {
	superGroup := e.Group("/organizations")
	superGroup.POST("", upsertOrgHandler).Name = "superadmin.organizations.Upsert"
	superGroup.GET("", listOrgHandler).Name = "superadmin.organizations.List"
}

func listOrgHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	orgs, err := superadmin.ListOrgs(ctx)
	if err != nil {
		c.Logger().Errorf("could not list all orgs %s", err.Error())
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}
	return c.JSON(http.StatusOK, orgs)
}

func upsertOrgHandler(c echo.Context) error {
	orgForm := types.OrganizationForm{}
	if err := c.Bind(&orgForm); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	c.Logger().Debug(orgForm)
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	org, err := superadmin.AddOrUpdateOrg(ctx, &orgForm)
	if err != nil {
		if err, ok := err.(types.InvalidFormError); ok {
			err.Form = orgForm
			return c.JSON(http.StatusBadRequest, err)
		}
		c.Logger().Error("Unexpected error when creating a new organization:", err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error while creating new organization")
	}
	c.Logger().Debug(org)

	return c.JSON(http.StatusOK, org)
}

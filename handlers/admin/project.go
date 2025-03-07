package admin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/services"
	projectService "github.com/nbittich/wtm/services/project"
	"github.com/nbittich/wtm/types"
)

func AdminProjectRouter(e *echo.Echo) {
	projectsGroup := e.Group("/admin/projects")
	projectsGroup.POST("/:id/planning/cycle", upsertPlanningCycle).Name = "admin.planning.UpsertPlanningCycle"
	projectsGroup.POST("/:id/planning/cycle/validate", validatePlanningCycle).Name = "admin.planning.ValidatePlanningCycle"
	projectsGroup.POST("/:id/planning/validate", validatePlanningEntry).Name = "admin.planning.Validate"
	projectsGroup.POST("/:id/planning", upsertPlanningEntry).Name = "admin.planning.UpsertPlanning"
	projectsGroup.GET("/:id/planning", getPlanning).Name = "admin.planning.Get"
	projectsGroup.GET("/:id", getProject).Name = "admin.project.Get"
	projectsGroup.POST("", upsertProject).Name = "admin.planning.UpsertProject"
	projectsGroup.GET("", listProjects).Name = "admin.project.ListProject"
}

func validatePlanningCycle(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}
	cycle := types.PlanningCycle{}
	if err = c.Bind(&cycle); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()

	entries, err := projectService.GeneratePlanningEntriesFromCycle(ctx, &cycle, adminUser.Group)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	valid, err := projectService.CheckEntriesValid(ctx, entries, adminUser.Group)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, valid)
}

func validatePlanningEntry(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}
	entry := types.PlanningEntry{}
	if err = c.Bind(&entry); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	valid, err := projectService.CheckEntriesValid(ctx, []types.PlanningEntry{entry}, adminUser.Group)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, valid)
}

func upsertPlanningCycle(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}
	cycle := types.PlanningCycle{}
	if err = c.Bind(&cycle); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	entries, err := projectService.MakePlanningCycle(ctx, &cycle, adminUser.Group)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, entries)
}

func upsertPlanningEntry(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}
	planningEntry := types.PlanningEntry{}
	if err = c.Bind(&planningEntry); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	entry, err := projectService.AddOrUpdatePlanningEntry(ctx, planningEntry, true, adminUser.Group)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, entry)
}

func upsertProject(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}

	project := types.Project{}
	if err := c.Bind(&project); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	if _, err := projectService.AddOrUpdateProject(ctx, &project, adminUser.Group); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, project)
}

func listProjects(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	projects, err := projectService.GetProjects(ctx, adminUser.Group)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, projects)
}

func getPlanning(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	projectID := c.Param("id")
	planning, err := projectService.GetPlanning(ctx, projectID, adminUser.Group)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, planning)
}

func getProject(c echo.Context) error {
	adminUser, err := services.GetUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("admin user not found in context"))
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()
	projectID := c.Param("id")
	project, err := projectService.GetProject(ctx, projectID, adminUser.Group)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, project)
}

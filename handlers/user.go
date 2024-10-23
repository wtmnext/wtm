package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/services"
	"github.com/nbittich/wtm/services/utils"
	"github.com/nbittich/wtm/types"
)

func UserRouter(e *echo.Echo) {
	userGroup := e.Group("/users")
	userGroup.GET("/activate", activateUserHandler).Name = "users.Activate"
	userGroup.POST("/login", loginHandler).Name = "users.Login"
	userGroup.GET("/logout", logoutHandler).Name = "users.Logout"
}

func handleGeneralFormError(c echo.Context, invalidFormError types.InvalidFormError) error {
	invalidFormError.Messages["general"] = utils.Translate(c.Request().Context(), invalidFormError.Messages["general"].(string))
	return c.JSON(http.StatusBadRequest, invalidFormError)
}

func logoutHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, &types.Message{
		Type:    types.WARNING,
		Message: "please delete your token. it will be removed automatically once it expires'",
	})
}

func loginHandler(c echo.Context) error {
	username := strings.TrimSpace(c.FormValue("username"))
	password := strings.TrimSpace(c.FormValue("password"))
	group := types.Group(strings.TrimSpace(c.FormValue("group")))
	invalidFormError := types.InvalidFormError{Messages: types.InvalidMessage{"general": "home.signin.invalidCredentials"}}
	if len(username) == 0 || len(password) == 0 {
		return handleGeneralFormError(c, invalidFormError)
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)
	defer cancel()

	user, error := services.FindByUsernameOrEmail(ctx, username, group)

	if error != nil || !user.Enabled {
		return handleGeneralFormError(c, invalidFormError)
	}
	passwordMatches := services.CheckPasswordHash(password, *user.Password)
	if !passwordMatches {
		fmt.Println("passwords don't match")
		return handleGeneralFormError(c, invalidFormError)
	}
	userClaims := &types.UserClaims{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Profile:  user.Profile,
		Settings: user.Settings,
		Roles:    user.Roles,
		Group:    *user.Group,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.JWTExpiresAFterMinutes)),
			Issuer:    config.JWTIssuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)
	ss, err := token.SignedString(config.JWTSecretKey)
	if err != nil {
		c.Logger().Error("error writing jwt", err)
		return handleGeneralFormError(c, invalidFormError)
	}
	jwt := map[string]string{"jwt": ss}
	return c.JSON(http.StatusOK, jwt)
}

func activateUserHandler(c echo.Context) error {
	hash := c.QueryParam("hash")
	group := c.QueryParam("group")
	ctx, cancel := context.WithTimeout(c.Request().Context(), config.MongoCtxTimeout)

	defer cancel()
	active, err := services.ActivateUser(ctx, hash, types.Group(group))
	if err != nil {
		c.Logger().Error("could not activate user: ", err)
	}
	message := types.Message{}
	if active {
		message.Type = types.SUCCESS
		message.Message = "home.signup.user.activated"
	} else {
		message.Type = types.ERROR
		message.Message = "home.signup.user.notActivated"
	}

	message.Message = utils.Translate(c.Request().Context(), message.Message)
	return c.JSON(http.StatusOK, message)
}

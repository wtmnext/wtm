package utils

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/types"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func GeneralFormError(c echo.Context, invalidFormError types.InvalidFormError) error {
	invalidFormError.Messages["general"] = Translate(c.Request().Context(), invalidFormError.Messages["general"].(string))
	return c.JSON(http.StatusBadRequest, invalidFormError)
}

func Translate(c context.Context, id string) string {
	msg := &i18n.Message{ID: id}
	lz, ok := c.Value(types.I18nKey).(*i18n.Localizer)
	if ok {
		msg, e := lz.LocalizeMessage(msg)
		if e == nil {
			return msg
		}
	}

	return id
}

func RenderHTML(statusCode int, c echo.Context, tpl templ.Component) error {
	c.Response().Status = statusCode
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	ctx := c.Request().Context()

	msg := types.Message{}
	if err := c.Bind(&msg); err == nil && msg.Message != "" {
		ctx = context.WithValue(ctx, types.MessageKey, msg)
	}
	if tok, ok := c.Get("user").(*jwt.Token); ok {
		if user, ok := tok.Claims.(*types.UserClaims); ok {
			ctx = context.WithValue(ctx, types.UserKey, *user)
		}
	}

	return tpl.Render(ctx, c.Response().Writer)
}

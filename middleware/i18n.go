package middleware

import (
	"context"

	"github.com/BurntSushi/toml"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/types"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var bundle *i18n.Bundle

func init() {
	bundle = i18n.NewBundle(language.French)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.LoadMessageFile("i18n/fr.toml")
	bundle.LoadMessageFile("i18n/en.toml")
}

func I18n(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		r := c.Request()
		lang := r.FormValue("lang")
		accept := r.Header.Get("Accept-Language")
		ctx := r.Context()
		if tok, ok := c.Get("user").(*jwt.Token); ok {
			if user, ok := tok.Claims.(*types.UserClaims); ok {
				if user.Settings.Lang != "" {
					lang = user.Settings.Lang
				}
			}
		}

		if lang != "" {
			ctx = context.WithValue(ctx, types.LangKey, lang)
		} else {
			ctx = context.WithValue(ctx, types.LangKey, accept)
		}

		localizer := i18n.NewLocalizer(bundle, lang, accept)
		c.SetRequest(r.WithContext(context.WithValue(ctx, types.I18nKey, localizer)))
		return next(c)
	}
}

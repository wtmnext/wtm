package main

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/handlers"
	adminHandlers "github.com/nbittich/wtm/handlers/admin"
	superadminHandlers "github.com/nbittich/wtm/handlers/superadmin"
	userHandlers "github.com/nbittich/wtm/handlers/user"
	appMidleware "github.com/nbittich/wtm/middleware"
	"github.com/nbittich/wtm/services/db"
	"github.com/nbittich/wtm/services/email"
	"github.com/nbittich/wtm/types"
)

//go:embed banner.txt
var BANNER string

func main() {
	loc, err := time.LoadLocation(config.TZ)
	if err != nil {
		panic(err)
	}
	fmt.Println("will use tz", loc)
	time.Local = loc

	defer db.Disconnect()
	defer close(email.MailChan)

	e := echo.New()

	// static assets
	e.Static("/assets", "assets")

	// static ext
	if _, err := os.Stat(config.StaticDirectory); os.IsNotExist(err) {
		os.MkdirAll(config.StaticDirectory, 0o755)
	}
	e.Static("/static", config.StaticDirectory)

	// middleware
	// e.Pre(middleware.AddTrailingSlash()) interfer with POST form

	if config.GoEnv == config.DEVELOPMENT {
		e.Use(middleware.CORS())
	}

	if config.GoEnv == config.PRODUCTION {
		e.Use(middleware.Secure())
	}
	// csrf
	// e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
	// 	TokenLookup: "form:csrf",
	// 	Skipper: func(c echo.Context) bool {
	// 		request := c.Request()
	// 		accept := request.Header.Get(echo.HeaderAccept)
	// 		return strings.Contains(c.Path(), "assets") || accept == echo.MIMEApplicationJSON
	// 	},
	// }))
	e.Use(middleware.Gzip())
	e.Use(middleware.Logger())

	// JWT

	e.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: config.JWTSecretKey,
		// TokenLookup:            fmt.Sprintf("header:Authorization:Bearer ,cookie:%s", config.JWTCookie),
		TokenLookupFuncs:       []middleware.ValuesExtractor{appMidleware.JWTTokenExtractor},
		ContinueOnIgnoredError: true,
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(types.UserClaims)
		},
		ErrorHandler: appMidleware.JWTErrorHandler,
	}))

	e.Use(appMidleware.ValidateAuth)

	e.Use(appMidleware.I18n)
	// end middleware

	e.HideBanner = true
	e.Logger.SetLevel(config.LogLevel)

	// email consume logs
	go func() {
		for msg := range email.MailChan {
			switch msg.(type) {
			case string:
				e.Logger.Info(msg)
			case error:
				e.Logger.Error(e)
			}
		}
	}()

	fmt.Println()

	fmt.Println(BANNER)

	handlers.UserRouter(e)
	handlers.HomeRouter(e)
	adminHandlers.AdminUserRouter(e)
	adminHandlers.AdminProjectRouter(e)
	userHandlers.UserPlanningRoute(e)
	superadminHandlers.SuperAdminRouter(e)
	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%s", config.Host, config.Port)))
}

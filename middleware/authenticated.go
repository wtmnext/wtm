package middleware

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/services"
	"github.com/nbittich/wtm/services/db"
	"github.com/nbittich/wtm/services/utils"
	"github.com/nbittich/wtm/types"
	"github.com/nbittich/wtm/views"
	"go.mongodb.org/mongo-driver/bson"
)

type AuthConfig struct {
	Pattern         string       `json:"pattern"`
	Authenticated   bool         `json:"authenticated"`
	Unauthenticated bool         `json:"unauthenticated"`
	Roles           []types.Role `json:"roles"`
	Group           types.Group  `json:"group"`
}

//go:embed auth_config.json
var authConfigFile []byte
var authConfigs []AuthConfig

func init() {
	if err := json.Unmarshal(authConfigFile, &authConfigs); err != nil {
		panic(err)
	}
}

func JWTTokenExtractor(c echo.Context) ([]string, error) {
	tok := c.Request().Header.Get("Authorization")
	if strings.Contains(tok, "Bearer ") {
		split := strings.Split(tok, "Bearer ")
		if len(split) != 2 {
			return nil, fmt.Errorf("invalid token %s", split)
		}
		return []string{split[1]}, nil
	}
	return nil, fmt.Errorf("invalid token %s", tok)
}

func JWTErrorHandler(c echo.Context, err error) error {
	for _, ac := range authConfigs {
		if m, _ := regexp.MatchString(ac.Pattern, c.Path()); m {
			if ac.Authenticated {
				c.Logger().Warnf("login error %s", err.Error())
				e := echo.ErrUnauthorized
				e.SetInternal(err)
				return e
			}
		}
	}
	return nil
}

func forbidden(c echo.Context) error {
	request := c.Request()
	accept := request.Header.Get(echo.HeaderAccept)
	if accept == echo.MIMEApplicationJSON {
		return c.JSON(http.StatusForbidden, types.Message{Type: types.ERROR, Message: "Forbidden"})
	} else {
		message := types.Message{}
		message.Type = types.ERROR
		message.Message = "common.forbidden"
		ctx := request.Context()
		c.SetRequest(request.WithContext(context.WithValue(ctx, types.MessageKey, message)))

		return utils.RenderHTML(http.StatusForbidden, c, views.Error())

	}
}

func ValidateAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		accept := c.Request().Header.Get(echo.HeaderAccept)
		if accept == echo.MIMETextHTML {
			return forbidden(c)
		}

		user := services.GetUser(c)

		for _, ac := range authConfigs {
			if m, _ := regexp.MatchString(ac.Pattern, c.Path()); m {
				if ac.Unauthenticated && user != nil {
					return forbidden(c)
				}
				if ac.Authenticated && user == nil {
					return forbidden(c)
				}
				if ac.Authenticated {
					// check user token wasn't forged
					// fixme maybe add a special hash with a combination of what we have above in db
					filter := bson.M{
						"$and": []bson.M{
							{"username": user.Username},
							{"group": user.Group},
							{"_id": user.ID},
						},
					}
					collection, err := db.GetCollection(services.UserCollection, user.Group)
					if err != nil {
						return forbidden(c)
					}
					if exist, err := db.Exist(c.Request().Context(), filter, collection); !exist || err != nil {
						return forbidden(c)
					}

					// vaidate roles
					if len(ac.Roles) > 0 {
						mapElt := make(map[types.Role]bool, len(user.Roles))
						for _, r := range user.Roles {
							mapElt[r] = true
						}
						for _, r := range ac.Roles {
							if !mapElt[r] {
								return forbidden(c)
							}
						}

					}
				}

			}
		}
		return next(c)
	}
}

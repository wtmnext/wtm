package types

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserActivationURL struct {
	ID        string    `bson:"_id" json:"_id"`
	UserID    string    `json:"userId" bson:"userId"`
	Hash      string    `json:"hash" bson:"hash"`
	Group     Group     `json:"group" bson:"group"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type NewUserForm struct {
	Username        string `json:"username" form:"username" validate:"required,min=3,max=15"`
	Password        string `json:"password" form:"password" validate:"required,min=6,max=18,password"`
	ConfirmPassword string `json:"confirmPassword" form:"confirmPassword" validate:"eqcsfield=Password"`
	Email           string `json:"email" form:"email" validate:"required,email"`
	ConfirmEmail    string `json:"confirmEmail" form:"confirmEmail" validate:"eqcsfield=Email"`
	Group           Group  `json:"group" form:"gorup" validate:"required"`
}

type User struct {
	ID       string      `bson:"_id" json:"_id"`
	Username string      `json:"username"`
	Password string      `json:"password"`
	Enabled  bool        `json:"enabled"`
	Email    string      `json:"email"`
	Profile  UserProfile `json:"profile"`
	Roles    []Role      `json:"roles"`
	Group    Group       `json:"group"`
	Settings UserSetting `json:"settings"`
}

type UserClaims struct {
	Username string      `json:"username"`
	Email    string      `json:"email"`
	Profile  UserProfile `json:"profile"`
	Settings UserSetting `json:"settings"`
	Roles    []Role      `json:"roles"`
	Group    Group       `json:"group"`
	jwt.RegisteredClaims
}

type UserProfile struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type UserSetting struct {
	Lang string `json:"lang"`
}

type Role uint8

type Group string

const (
	USER Role = iota + 1
	ADMIN
)

func (user User) GetID() string {
	return user.ID
}

func (user *User) SetID(id string) {
	user.ID = id
}

func (userActivationURL UserActivationURL) GetID() string {
	return userActivationURL.ID
}

func (userActivationURL *UserActivationURL) SetID(id string) {
	userActivationURL.ID = id
}

func (user *UserActivationURL) GenerateURL(baseURL string) string {
	return fmt.Sprintf("%s?hash=%s&group=%s", baseURL, user.Hash, string(user.Group))
}

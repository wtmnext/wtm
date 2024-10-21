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
	Username        string `json:"username" form:"username" validate:"required,min=3,max=15,alphanum,startswithalpha"`
	Password        string `json:"password" form:"password" validate:"required,min=6,max=18,password"`
	ConfirmPassword string `json:"confirmPassword" form:"confirmPassword" validate:"eqcsfield=Password"`
	Email           string `json:"email" form:"email" validate:"required,email"`
	ConfirmEmail    string `json:"confirmEmail" form:"confirmEmail" validate:"eqcsfield=Email"`
	Role            *Role  `json:"role" form:"role" validate:"omitempty"`
}

type Organization struct {
	ID             string            `bson:"_id" json:"_id"`
	Group          Group             `bson:"group" json:"group"`
	FullName       string            `bson:"fullName" json:"fullName"`
	AdditionalInfo map[string]string `bson:"additionalInfo" json:"additionalInfo"`
	Email          string            `json:"email"`
}

type OrganizationForm struct {
	ID             string            `json:"_id"`
	Group          string            `json:"group" validate:"required,min=2,max=24,alpha"`
	FullName       string            `json:"fullName" validate:"required, min=2,max=255"`
	AdditionalInfo map[string]string `json:"additionalInfo"`
	NewUser        *NewUserForm      `json:"newUser" validate:"omitempty,dive"`
	Email          *string           `json:"email" validate:"omitempty,email"`
}

type User struct {
	ID       string      `bson:"_id" json:"_id"`
	Username string      `json:"username"`
	Password string      `json:"password"`
	Enabled  bool        `json:"enabled"`
	Email    string      `json:"email"`
	Profile  UserProfile `json:"profile"`
	Roles    []Role      `json:"roles"`
	Group    *Group      `json:"group"`
	Settings UserSetting `json:"settings"`
}

type UserClaims struct {
	ID       string      `json:"id"`
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

type Role string

type Group string

const (
	USER       Role = "USER"
	ADMIN      Role = "ADMIN"
	SUPERADMIN Role = "SUPERADMIN"
)

func (user User) GetID() string {
	return user.ID
}

func (user *User) SetID(id string) {
	user.ID = id
}

func (org Organization) GetID() string {
	return org.ID
}

func (org *Organization) SetID(id string) {
	org.ID = id
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

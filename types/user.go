package types

import (
	"fmt"
	"slices"
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

type Organization struct {
	ID             string           `bson:"_id" json:"_id"`
	Group          Group            `bson:"group" json:"group"`
	FullName       string           `bson:"fullName" json:"fullName"`
	AdditionalInfo []AdditionalInfo `bson:"additionalInfo" json:"additionalInfo"`
	Email          string           `json:"email"`
}

type OrganizationForm struct {
	ID             string           `json:"_id"`
	Group          string           `json:"group" validate:"required,min=2,max=24,alpha"`
	FullName       string           `json:"fullName" validate:"required,min=2,max=255"`
	AdditionalInfo []AdditionalInfo `json:"additionalInfo" validate:"omitempty"`
	NewUser        *NewUserForm     `json:"newUser" validate:"omitempty"`
	Email          *string          `json:"email" validate:"omitempty,email"`
}

type AdditionalInfo struct {
	Key   string `json:"key" validate:"required,min=2"`
	Value string `json:"value" validate:"required,min=2"`
}
type Role string

type Group string

const (
	USER       Role = "USER"
	ADMIN      Role = "ADMIN"
	SUPERADMIN Role = "SUPERADMIN"
)

type NewUserForm struct {
	Username        string `json:"username" form:"username" validate:"required,min=3,max=15,alphanum,startswithalpha"`
	Password        string `json:"password" form:"password" validate:"required,min=6,max=18,password"`
	ConfirmPassword string `json:"confirmPassword" form:"confirmPassword" validate:"eqcsfield=Password"`
	Email           string `json:"email" form:"email" validate:"required,email"`
	ConfirmEmail    string `json:"confirmEmail" form:"confirmEmail" validate:"eqcsfield=Email"`
	Role            *Role  `json:"role" form:"role" validate:"omitempty"`
}
type User struct {
	ID       string      `bson:"_id" json:"_id"`
	Username string      `json:"username"`
	Password *string     `json:"-" bson:"password"`
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
	FirstName    string                  `json:"firstName"`
	LastName     string                  `json:"lastName"`
	Availability *UserNormalAvailability `json:"availability" bson:"availability,omitempty"`
}

type UserSetting struct {
	Lang string `json:"lang"`
}

type UserNormalAvailability struct {
	Days        []time.Weekday `json:"days"`
	MinHour     int            `json:"minHour"`
	MaxHour     int            `json:"maxHour"`
	HoursPerDay int            `json:"hoursPerday"`
}

func (availability *UserNormalAvailability) IsAvailable(startStr string, endStr string) (bool, error) {
	var (
		start, end time.Time
		err        error
	)

	if start, err = time.Parse(BelgianDateTimeFormat, startStr); err != nil {
		return false, err
	}
	if end, err = time.Parse(BelgianDateTimeFormat, endStr); err != nil {
		return false, err
	}
	if availability != nil {
		if !slices.Contains(availability.Days, start.Weekday()) || !slices.Contains(availability.Days, end.Weekday()) {
			return false, nil
		}
		difference := end.Sub(start).Abs()
		if availability.HoursPerDay < int(difference.Hours()) {
			return false, nil
		}
		minTime := time.Date(start.Year(), start.Month(), start.Day(), availability.MinHour, 0, 0, 0, start.Location())
		maxTime := time.Date(start.Year(), start.Month(), start.Day(), availability.MaxHour, 0, 0, 0, start.Location())

		if minTime.After(maxTime) || minTime.Equal(maxTime) { // e.g 00:00 -> 00:00 should be 24h then
			maxTime = maxTime.Add(time.Hour * 24)
		}

		return (start.After(minTime) || start.Equal(minTime)) &&
			(end.Before(maxTime) || end.Equal(maxTime)), nil
	}
	return false, nil
}

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

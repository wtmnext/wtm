package types

import (
	"encoding/json"
	"time"
)

type HasID interface {
	GetID() string
}
type Identifiable interface {
	HasID
	SetID(id string)
}

const (
	I18nKey = CtxKey("localizer")
	LangKey = CtxKey("lang")
	// CsrfKey            = CtxKey("csrf")
	SignupFormErrorKey = CtxKey("signupFormError")
	SigninFormErrorKey = CtxKey("signinFormError")
	MessageKey         = CtxKey("message")
	UserKey            = CtxKey("user")
)

type StatusType int8

const (
	INFO StatusType = iota + 1
	SUCCESS
	WARNING
	ERROR
)

type Message struct {
	Type    StatusType `json:"type" url:"type" param:"type" form:"type" query:"type"`
	Message string     `json:"message" url:"message" param:"message" form:"message" query:"message"`
}

type (
	CtxKey           string
	InvalidMessage   = map[string]interface{}
	InvalidFormError struct {
		Form     interface{}
		Messages InvalidMessage `json:"messages"`
	}
)

func (apiError InvalidFormError) Error() string {
	val, e := json.Marshal(apiError.Messages)
	if e != nil {
		return e.Error()
	}
	return string(val)
}

type TimeISO8601 struct {
	time.Time
}

func (date *TimeISO8601) UnmarshalCSV(csv string) (err error) {
	if csv == "" {
		return nil
	}
	date.Time, err = time.Parse(time.RFC3339, csv)
	return err
}

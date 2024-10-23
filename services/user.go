package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/services/db"
	"github.com/nbittich/wtm/services/email"
	"github.com/nbittich/wtm/services/utils"
	"github.com/nbittich/wtm/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

const (
	UserCollection              = "user"
	UserActivationURLCollection = "userActivationUrl"
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), config.DefaultBCryptCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GetUser(c echo.Context) *types.UserClaims {
	if tok, ok := c.Get("user").(*jwt.Token); ok {
		if user, ok := tok.Claims.(*types.UserClaims); ok {
			return user
		}
	}
	return nil
}

func NewUser(ctx context.Context, newUserForm *types.NewUserForm, group types.Group) (*types.User, error) {
	lang := ctx.Value(types.LangKey).(string)
	var err error
	collection, err := db.GetCollection(UserCollection, group)
	if err != nil {
		return nil, err
	}
	err = utils.ValidateStruct(newUserForm)
	if err != nil {
		return nil, err
	}

	password, err := hashPassword(newUserForm.Password)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"$or": []bson.M{
			{"email": newUserForm.Email},
			{"username": newUserForm.Username},
		},
	}

	exist, err := db.Exist(ctx, filter, collection)
	if err != nil {
		return nil, err
	}

	if exist {
		m := types.InvalidMessage{"general": "home.signup.user.exist"}
		return nil, types.InvalidFormError{Form: newUserForm, Messages: m}
	}

	var roles []types.Role
	if newUserForm.Role != nil && *newUserForm.Role != types.USER {
		roles = make([]types.Role, 0, 2)
		roles = append(roles, types.USER)
		roles = append(roles, *newUserForm.Role)
	} else {
		roles = []types.Role{types.USER}
	}

	user := &types.User{
		Username: newUserForm.Username,
		Password: password,
		Email:    newUserForm.Email,
		Enabled:  false,
		Settings: types.UserSetting{Lang: lang},
		Profile:  types.UserProfile{},
		Roles:    roles,
		Group:    &group,
	}

	go sendActivationEmail(user, true)
	return user, nil
}

func sendActivationEmail(user *types.User, createUser bool) {
	collection, err := db.GetCollection(UserCollection, *user.Group)
	if err != nil {
		log.Println("could not create user:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.MongoCtxTimeout)
	defer cancel()
	if createUser {
		if _, e := db.InsertOrUpdate(ctx, user, collection); e != nil {
			log.Println("could not create user:", e)
			return
		}
	}
	activateURL, e := GenerateActivateURL(ctx, config.BaseURL+"/users/activate", user.ID, *user.Group)
	if e != nil {
		log.Println("error while generating validation url", e)
		return
	}
	email.SendAsync([]string{user.Email}, []string{}, "Activate your account", fmt.Sprintf(`<a href="%s">Activate your account now!</p>`, activateURL))
}

func FindByUsernameOrEmail(ctx context.Context, username string, group types.Group) (types.User, error) {
	userCollection, err := db.GetCollection(UserCollection, group)
	if err != nil {
		return types.User{}, err
	}
	filter := bson.M{
		"$or": []bson.M{
			{"email": username},
			{"username": username},
		},
	}
	return db.FindOneBy[types.User](ctx, filter, userCollection)
}

func ActivateUser(ctx context.Context, hash string, group types.Group) (bool, error) {
	var (
		userCollection              *mongo.Collection
		userActivationURLCollection *mongo.Collection
		err                         error
	)
	if userCollection, err = db.GetCollection(UserCollection, group); err != nil {
		return false, err
	}

	if userActivationURLCollection, err = db.GetCollection(UserActivationURLCollection, group); err != nil {
		return false, err
	}
	userActivationURL, err := db.FindOneBy[types.UserActivationURL](ctx, bson.M{
		"hash": hash,
	}, userActivationURLCollection)
	if err != nil {
		return false, err
	}
	user, err := db.FindOneByID[types.User](ctx, userCollection, userActivationURL.UserID)
	if err != nil {
		return false, err
	}
	if user.Enabled {
		return false, fmt.Errorf("user already enabled")
	}
	now := time.Now()
	duration := now.Sub(userActivationURL.UpdatedAt)
	if duration > config.ActivationExpiration {
		log.Println("activation link no longer valid")
		go sendActivationEmail(&user, false)
		return false, fmt.Errorf("invalid hash")
	}
	userActivationURL.UpdatedAt = now

	user.Enabled = true
	_, err = db.InsertOrUpdate(ctx, &user, userCollection)
	if err != nil {
		return false, err
	}
	_, _ = db.InsertOrUpdate(ctx, &userActivationURL, userActivationURLCollection)
	return true, nil
}

func GenerateActivateURL(ctx context.Context, baseURL string, userID string, group types.Group) (string, error) {
	var (
		userCollection              *mongo.Collection
		userActivationURLCollection *mongo.Collection
		err                         error
		user                        types.User
	)
	if userCollection, err = db.GetCollection(UserCollection, group); err != nil {
		return "", err
	}
	if userActivationURLCollection, err = db.GetCollection(UserActivationURLCollection, group); err != nil {
		return "", err
	}
	if user, err = db.FindOneByID[types.User](ctx, userCollection, userID); err != nil {
		return "", err
	}
	if user.Enabled {
		return "", fmt.Errorf("user.alreadyEnabled")
	}
	filter := bson.M{
		"userId": user.ID,
	}
	userActivationURL, err := db.FindOneBy[types.UserActivationURL](ctx, filter, userActivationURLCollection)
	if err != nil {
		now := time.Now()
		duration := now.Sub(userActivationURL.UpdatedAt)
		if duration < config.ActivationExpiration {
			log.Println("activation link still valid")
			return userActivationURL.GenerateURL(baseURL), nil
		}
	}
	userActivationURL.Hash = uuid.New().String()
	userActivationURL.Group = group
	userActivationURL.UpdatedAt = time.Now()
	userActivationURL.UserID = userID
	_, err = db.InsertOrUpdate(ctx, &userActivationURL, userActivationURLCollection)
	if err != nil {
		return "", nil
	}
	return userActivationURL.GenerateURL(baseURL), nil
}

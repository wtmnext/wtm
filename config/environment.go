package config

import (
	"fmt"
	stdLog "log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
)

var (
	Host                     = loadEnvOrDefault("HOST", "0.0.0.0")
	HostName                 = loadEnvOrDefault("HOSTNAME", "localhost")
	Port                     = loadEnvOrDefault("PORT", "8080")
	BaseURL                  = loadEnvOrDefault("BASE_URL", fmt.Sprintf("http://%s:%s", HostName, Port))
	GoEnv                    = env(loadEnvOrDefault("GO_ENV", "development"))
	LogLevel                 = logLevel(loadEnvOrDefault("LOG_LEVEL", "INFO"))
	SMTPHost                 = loadEnvOrDefault("SMTP_HOST", "localhost")
	SMTPPort                 = loadIntEnvOrDefault("SMTP_PORT", 1025)
	SMTPFrom                 = loadEnvOrDefault("SMTP_FROM", "test@localhost")
	SMTPPassword             = loadEnvOrDefault("SMTP_PASSWORD", "")
	SMTPSSL                  = loadBoolOrDefault("SMTP_SSL", false)
	MongoHost                = loadEnvOrDefault("MONGO_HOST", "localhost")
	MongoPort                = loadEnvOrDefault("MONGO_PORT", "27017")
	MongoUser                = loadEnvOrDefault("MONGO_USER", "root")
	MongoMigrationCollection = loadEnvOrDefault("MONGO_MIGRATION_COLLECTION", "_migration")
	MongoPassword            = loadEnvOrDefault("MONGO_PASSWORD", "root")
	MongoAdminDBName         = loadEnvOrDefault("MONGO_ADMIN_DB_NAME", "wtm")
	MongoCtxTimeout          = time.Duration(loadIntEnvOrDefault("MONGO_CONTEXT_TIMEOUT_SECONDS", 60)) * time.Second
	MongoMaxConnectionPool   = loadIntEnvOrDefault("MONGO_MAX_CONNECTION_POOL", 200)
	ActivationExpiration     = time.Duration(loadIntEnvOrDefault("ACTIVATION_EXPIRATION", 20)) * time.Minute
	JWTSecretKey             = []byte(loadEnvOrDefault("JWT_SECRET_KEY", "secret"))
	JWTExpiresAFterMinutes   = time.Duration(loadIntEnvOrDefault("JWT_EXPIRES_AFTER_MINUTES", 30)) * time.Minute
	JWTIssuer                = loadEnvOrDefault("JWT_ISSUER", "WorkingTimeManagement")
	// JWTCookie                = loadEnvOrDefault("JWT_COOKIE", "jwt")
	TempDir         = loadEnvOrDefault("TMP_DIRECTORY", os.TempDir())
	StaticDirectory = loadEnvOrDefault("STATIC_DIRECTORY", fmt.Sprint(os.TempDir(), "/wtm/static"))
)

type EnvType uint8

const (
	DEVELOPMENT EnvType = iota + 1
	TEST
	PRODUCTION
)

func loadEnvOrDefault(key string, defaultValue string) string {
	value, exist := os.LookupEnv(key)
	if !exist {
		return defaultValue
	} else {
		return value
	}
}

func logLevel(level string) log.Lvl {
	var lvl log.Lvl
	switch strings.ToUpper(level) {
	case "DEBUG":
		lvl = log.DEBUG
	case "INFO":
		lvl = log.INFO
	case "WARN":
		lvl = log.WARN
	case "ERROR":
		lvl = log.ERROR
	case "OFF":
		lvl = log.OFF
	default:
		stdLog.Println("warning! invalid log level:", level)
		lvl = log.INFO
	}
	return lvl
}

func loadBoolOrDefault(key string, defaultValue bool) bool {
	value := loadEnvOrDefault(key, fmt.Sprint(defaultValue))
	b, err := strconv.ParseBool(value)
	if err != nil {
		stdLog.Println("warning! invalid key value (bool conversion):", key)
		return defaultValue
	}
	return b
}

func loadIntEnvOrDefault(key string, defaultValue int) int {
	value := loadEnvOrDefault(key, fmt.Sprint(defaultValue))
	num, err := strconv.Atoi(value)
	if err != nil {
		stdLog.Println("warning! invalid key value (int conversion):", key)
		return defaultValue
	}
	return num
}

func env(envType string) EnvType {
	switch strings.ToUpper(envType) {
	case "DEVELOPMENT":
		return DEVELOPMENT
	case "TEST":
		return TEST
	case "PRODUCTION":
		return PRODUCTION
	default:
		stdLog.Println("warning! invalid env type:", envType)
		return DEVELOPMENT

	}
}

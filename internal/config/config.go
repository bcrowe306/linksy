package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                  string
	DBType                string // "sqlite" or "postgres"
	DBDSN                 string
	StorageType           string // "local" or "r2" or "s3"
	StoragePath           string
	AppURL                string
	R2Endpoint            string
	R2AccessKeyID         string
	R2SecretAccessKey     string
	R2Bucket              string
	WorkOSAPIKey          string
	WorkOSClient          string
	WorkOSAPIKeyTest      string
	WorkOSClientIDStaging string
	WorkOSAPIKeyProd      string
	WorkOSClientIDProd    string
	WorkOSRedirectURI     string
	LogLevel              string
	// AppSecret salts analytics IP hashes. Required in production.
	AppSecret string
	// DevLogin enables a password-less local login. Only honoured when AppURL is localhost.
	DevLogin bool
}

// SecureCookies reports whether cookies should carry the Secure attribute.
func (c *Config) SecureCookies() bool {
	return strings.HasPrefix(c.AppURL, "https://")
}

// IsLocal reports whether the app is running on a loopback development URL.
func (c *Config) IsLocal() bool {
	u, err := url.Parse(c.AppURL)
	if err != nil {
		return false
	}
	h := u.Hostname()
	return h == "localhost" || h == "127.0.0.1"
}

func LoadConfig() *Config {
	port := getEnv("PORT", "8080")
	dbType := getEnv("DB_TYPE", "sqlite")
	dbDSN := getEnv("DATABASE_URL", getEnv("DB_DSN", "linksy.db"))
	storageType := getEnv("STORAGE_TYPE", "local")
	storagePath := getEnv("STORAGE_PATH", "./storage/uploads")
	appURL := getEnv("APP_URL", "http://localhost:"+port)
	r2Endpoint := getEnv("R2_ENDPOINT", getEnv("CLOUDFARE_S3_ENDPOINT", ""))
	r2AccessKeyID := getEnv("R2_ACCESS_KEY_ID", getEnv("CLOUDFARE_ACCESS_KEY_ID", ""))
	r2SecretAccessKey := getEnv("R2_SECRET_ACCESS_KEY", getEnv("CLOUDFARE_SECRET_KEY", ""))
	r2Bucket := getEnv("R2_BUCKET", "lynqr-storage")

	workosAPIKeyTest := getEnv("WORKOS_API_KEY_TEST", "")
	workosClientIDStaging := getEnv("WORKOS_CLIENT_ID_STAGING", "")
	workosAPIKeyProd := getEnv("WORKOS_API_KEY_PROD", "")
	workosClientIDProd := getEnv("WORKOS_CLIENT_ID_PROD", "")

	workosAPIKey := getEnv("WORKOS_API_KEY", "")
	if workosAPIKey == "" {
		if workosAPIKeyProd != "" {
			workosAPIKey = workosAPIKeyProd
		} else {
			workosAPIKey = workosAPIKeyTest
		}
	}

	workosClient := getEnv("WORKOS_CLIENT_ID", "")
	if workosClient == "" {
		if workosClientIDProd != "" {
			workosClient = workosClientIDProd
		} else {
			workosClient = workosClientIDStaging
		}
	}

	workosRedirectURI := getEnv("WORKOS_REDIRECT_URI", appURL+"/api/v1/auth/callback")
	logLevel := getEnv("LOG_LEVEL", "info")

	// If DATABASE_URL starts with postgres:// or postgresql://, automatically use postgres
	if len(dbDSN) > 10 && (dbDSN[:11] == "postgres://" || dbDSN[:13] == "postgresql://") {
		dbType = "postgres"
	}

	// If R2 credentials are present and STORAGE_TYPE is r2 or s3, select r2
	if (storageType == "r2" || storageType == "s3") && r2Endpoint != "" && r2AccessKeyID != "" {
		storageType = "r2"
	}

	return &Config{
		Port:                  port,
		DBType:                dbType,
		DBDSN:                 dbDSN,
		StorageType:           storageType,
		StoragePath:           storagePath,
		AppURL:                appURL,
		R2Endpoint:            r2Endpoint,
		R2AccessKeyID:         r2AccessKeyID,
		R2SecretAccessKey:     r2SecretAccessKey,
		R2Bucket:              r2Bucket,
		WorkOSAPIKey:          workosAPIKey,
		WorkOSClient:          workosClient,
		WorkOSAPIKeyTest:      workosAPIKeyTest,
		WorkOSClientIDStaging: workosClientIDStaging,
		WorkOSAPIKeyProd:      workosAPIKeyProd,
		WorkOSClientIDProd:    workosClientIDProd,
		WorkOSRedirectURI:     workosRedirectURI,
		LogLevel:              logLevel,
		AppSecret:             getEnv("APP_SECRET", ""),
		DevLogin:              getEnv("DEV_LOGIN", "") == "true",
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

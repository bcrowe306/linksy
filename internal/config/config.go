package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port         string
	DBType       string // "sqlite" or "postgres"
	DBDSN        string
	StorageType  string // "local" or "s3"
	StoragePath  string
	AppURL       string
	WorkOSAPIKey string
	WorkOSClient string
	LogLevel     string
}

func LoadConfig() *Config {
	port := getEnv("PORT", "8080")
	dbType := getEnv("DB_TYPE", "sqlite")
	dbDSN := getEnv("DATABASE_URL", getEnv("DB_DSN", "linksy.db"))
	storageType := getEnv("STORAGE_TYPE", "local")
	storagePath := getEnv("STORAGE_PATH", "./storage/uploads")
	appURL := getEnv("APP_URL", "http://localhost:"+port)
	workosAPIKey := getEnv("WORKOS_API_KEY", "")
	workosClient := getEnv("WORKOS_CLIENT_ID", "")
	logLevel := getEnv("LOG_LEVEL", "info")

	// If DATABASE_URL starts with postgres:// or postgresql://, automatically use postgres
	if len(dbDSN) > 10 && (dbDSN[:11] == "postgres://" || dbDSN[:13] == "postgresql://") {
		dbType = "postgres"
	}

	return &Config{
		Port:         port,
		DBType:       dbType,
		DBDSN:        dbDSN,
		StorageType:  storageType,
		StoragePath:  storagePath,
		AppURL:       appURL,
		WorkOSAPIKey: workosAPIKey,
		WorkOSClient: workosClient,
		LogLevel:     logLevel,
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

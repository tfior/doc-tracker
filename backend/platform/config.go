package platform

import "os"

type Config struct {
	DBUser     string
	DBPassword string
	DBName     string
	DBHost     string
	DBPort     string

	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageUseSSL    bool

	ServerPort     string
	SessionSecret  string
	MigrationsPath string
}

func LoadConfig() Config {
	return Config{
		DBUser:     getEnv("DB_USER", "doctracker"),
		DBPassword: getEnv("DB_PASSWORD", "changeme"),
		DBName:     getEnv("DB_NAME", "doctracker"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),

		StorageEndpoint:  getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey: getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey: getEnv("STORAGE_SECRET_KEY", "changeme"),
		StorageBucket:    getEnv("STORAGE_BUCKET", "doc-tracker"),
		StorageUseSSL:    getEnv("STORAGE_USE_SSL", "false") == "true",

		ServerPort:     getEnv("SERVER_PORT", "8080"),
		SessionSecret:  getEnv("SESSION_SECRET", "changeme"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "db/migrations"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

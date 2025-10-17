package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds application configuration.
type Config struct {
	DatabaseURL          string
	APIHost              string
	APIPort              string
	JWTSecret            string
	JWTExpiry            string
	StorageType          string
	UploadDir            string
	UploadURLPrefix      string // Added for configurable local storage URL path
	S3Bucket             string
	S3Region             string
	S3AccessKeyID        string
	S3SecretAccessKey    string
	S3BaseURL            string
	StorageTimeout       time.Duration // Added for configurable storage operation timeout
	RateLimitRequests    int
	RateLimitBurst       int
	MaxPostLength        int
	MaxTags              int
	DefaultMaxThreads    int
	DefaultMaxReplies    int
	DefaultMaxImageSize  int
	ArchiveDeleteDays    int
	LogLevel             string
	LogFile              string
	CORSAllowedOrigins   string
	CORSAllowedMethods   string
	CORSAllowedHeaders   string
	CORSAllowCredentials bool
}

// Load loads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL: fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			getEnv("POSTGRES_USER", "admin"),
			getEnv("POSTGRES_PASSWORD", "password"),
			getEnv("POSTGRES_HOST", "localhost"),
			getEnv("POSTGRES_PORT", "5432"),
			getEnv("POSTGRES_DB", "imageboard")),
		APIHost:              getEnv("API_HOST", "0.0.0.0"),
		APIPort:              getEnv("API_PORT", "8080"),
		JWTSecret:            getEnv("JWT_SECRET", "your-32-byte-secret-here"),
		JWTExpiry:            getEnv("JWT_EXPIRY", "24h"),
		StorageType:          getEnv("STORAGE_TYPE", "local"),
		UploadDir:            getEnv("UPLOAD_DIR", "/uploads"),
		UploadURLPrefix:      getEnv("UPLOAD_URL_PREFIX", "/uploads"), // Added default
		S3Bucket:             getEnv("S3_BUCKET", "my-bucket"),
		S3Region:             getEnv("S3_REGION", "us-east-1"),
		S3AccessKeyID:        getEnv("S3_ACCESS_KEY_ID", ""),
		S3SecretAccessKey:    getEnv("S3_SECRET_ACCESS_KEY", ""),
		S3BaseURL:            getEnv("S3_BASE_URL", "https://my-bucket.s3.amazonaws.com"),
		StorageTimeout:       time.Duration(getEnvAsInt("STORAGE_TIMEOUT_SECONDS", 10)) * time.Second, // Added default 10s
		RateLimitRequests:    getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitBurst:       getEnvAsInt("RATE_LIMIT_BURST", 10),
		MaxPostLength:        getEnvAsInt("MAX_POST_LENGTH", 5000),
		MaxTags:              getEnvAsInt("MAX_TAGS", 10),
		DefaultMaxThreads:    getEnvAsInt("DEFAULT_MAX_THREADS", 100),
		DefaultMaxReplies:    getEnvAsInt("DEFAULT_MAX_REPLIES", 500),
		DefaultMaxImageSize:  getEnvAsInt("DEFAULT_MAX_IMAGE_SIZE", 5242880),
		ArchiveDeleteDays:    getEnvAsInt("ARCHIVE_DELETE_DAYS", 30),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		LogFile:              getEnv("LOG_FILE", "stdout"),
		CORSAllowedOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "*"),
		CORSAllowedMethods:   getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		CORSAllowedHeaders:   getEnv("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Requested-With"),
		CORSAllowCredentials: getEnv("CORS_ALLOW_CREDENTIALS", "false") == "true",
	}
}

// getEnv retrieves an environment variable or returns a default.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default.
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

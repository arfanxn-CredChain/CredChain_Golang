package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GinPort                      *string
	InitialSuperAdminEmail       *string
	InitialSuperAdminPrivKey     *string
	InitialSuperAdminName        *string
	InitialSuperAdminNumber      *string
	InitialSuperAdminPhoneNumber *string
	InitialSuperAdminBirthDate   *time.Time
	InitialSuperAdminMeta        map[string]any
	WalletEncryptionKey          *string
	RPCURL                       *string
	RelayerPrivateKey            *string
	AuthorityContract            *string
	RegistryContract             *string
	JWTSecret                    *string
	JWTAccessExpiryMinutes       *int
	JWTRefreshExpiryHours        *int
	PostgresUser                 *string
	PostgresPassword             *string
	PostgresDB                   *string
	PostgresDSN                  *string
	DBMaxOpenConns               *int
	DBMaxIdleConns               *int
	DBConnMaxLifetime            *int
	MongoInitDBUsername          *string
	MongoInitPassword            *string
	MongoURI                     *string
	GeminiAPIKey                 *string
	GoogleClientID               *string
	GoogleClientSecret           *string
	GoogleRedirectURI            *string
	GinCorsAllowOrigins          []string
	GinCorsAllowMethods          []string
	GinCorsAllowHeaders          []string
	GinCorsExposeHeaders         []string
	GinCorsAllowCredentials      *bool
	GinCorsMaxAge                time.Duration
	LogLevel                     *string
	LogOutput                    *string
	I18nLocalesDir               *string
}

func getIntEnv(key string, defaultVal *int) *int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return &intVal
}

func getStringSliceEnv(key string, defaultVal []string) []string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return strings.Split(val, ",")
}

func getBoolEnv(key string, defaultVal *bool) *bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	result := val == "true"
	return &result
}

func NewConfig(envPath string) (*Config, error) {
	err := godotenv.Load(envPath)
	if err != nil {
		// Continue without .env file (env vars may be set directly)
	}

	defaultJWTAccessExpiry := 15
	defaultJWTRefreshExpiry := 168
	defaultDBMaxOpenConns := 25
	defaultDBMaxIdleConns := 25
	defaultDBConnMaxLifetime := 5
	defaultGinCorsMaxAge := 43200
	defaultGinCorsAllowCredentials := true
	defaultCORSOrigins := []string{"*"}
	defaultCORSMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	defaultCORSHeaders := []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	defaultCORSExposeHeaders := []string{"Content-Length"}
	defaultGoogleRedirectURI := "http://localhost:3000/google/callback"

	cfg := &Config{
		GinPort:                      getEnv("GIN_PORT", ptr("8080")),
		InitialSuperAdminEmail:       getEnv("INITIAL_SUPER_ADMIN_EMAIL", nil),
		InitialSuperAdminPrivKey:     getEnv("INITIAL_SUPER_ADMIN_PRIVATE_KEY", nil),
		InitialSuperAdminName:        getEnv("INITIAL_SUPER_ADMIN_NAME", nil),
		InitialSuperAdminNumber:      getEnv("INITIAL_SUPER_ADMIN_NUMBER", nil),
		InitialSuperAdminPhoneNumber: getEnv("INITIAL_SUPER_ADMIN_PHONE_NUMBER", nil),
		InitialSuperAdminBirthDate:   getTimeEnv("INITIAL_SUPER_ADMIN_BIRTH_DATE", nil),
		InitialSuperAdminMeta:        getJSONEnv("INITIAL_SUPER_ADMIN_META", nil),
		WalletEncryptionKey:          getEnv("WALLET_ENCRYPTION_KEY", nil),
		RPCURL:                       getEnv("RPC_URL", nil),
		RelayerPrivateKey:            getEnv("RELAYER_PRIVATE_KEY", nil),
		AuthorityContract:            getEnv("AUTHORITY_CONTRACT", nil),
		RegistryContract:             getEnv("REGISTRY_CONTRACT", nil),
		JWTSecret:                    getEnv("JWT_SECRET", nil),
		JWTAccessExpiryMinutes:       getIntEnv("JWT_ACCESS_EXPIRY_MINUTES", &defaultJWTAccessExpiry),
		JWTRefreshExpiryHours:        getIntEnv("JWT_REFRESH_EXPIRY_HOURS", &defaultJWTRefreshExpiry),
		PostgresUser:                 getEnv("POSTGRES_USER", nil),
		PostgresPassword:             getEnv("POSTGRES_PASSWORD", nil),
		PostgresDB:                   getEnv("POSTGRES_DB", nil),
		PostgresDSN:                  getEnv("POSTGRES_DSN", nil),
		DBMaxOpenConns:               getIntEnv("DB_MAX_OPEN_CONNS", &defaultDBMaxOpenConns),
		DBMaxIdleConns:               getIntEnv("DB_MAX_IDLE_CONNS", &defaultDBMaxIdleConns),
		DBConnMaxLifetime:            getIntEnv("DB_CONN_MAX_LIFETIME", &defaultDBConnMaxLifetime),
		MongoInitDBUsername:          getEnv("MONGO_INIT_DB_USERNAME", nil),
		MongoInitPassword:            getEnv("MONGO_INITDB_ROOT_PASSWORD", nil),
		MongoURI:                     getEnv("MONGO_URI", nil),
		GeminiAPIKey:                 getEnv("GEMINI_API_KEY", nil),
		GoogleClientID:               getEnv("GOOGLE_CLIENT_ID", nil),
		GoogleClientSecret:           getEnv("GOOGLE_CLIENT_SECRET", nil),
		GoogleRedirectURI:            getEnv("GOOGLE_REDIRECT_URI", &defaultGoogleRedirectURI),
		GinCorsAllowOrigins:          getStringSliceEnv("GIN_CORS_ALLOW_ORIGINS", defaultCORSOrigins),
		GinCorsAllowMethods:          getStringSliceEnv("GIN_CORS_ALLOW_METHODS", defaultCORSMethods),
		GinCorsAllowHeaders:          getStringSliceEnv("GIN_CORS_ALLOW_HEADERS", defaultCORSHeaders),
		GinCorsExposeHeaders:         getStringSliceEnv("GIN_CORS_EXPOSE_HEADERS", defaultCORSExposeHeaders),
		GinCorsAllowCredentials:      getBoolEnv("GIN_CORS_ALLOW_CREDENTIALS", &defaultGinCorsAllowCredentials),
		GinCorsMaxAge:                time.Duration(*getIntEnv("GIN_CORS_MAX_AGE", &defaultGinCorsMaxAge)) * time.Second,
		LogLevel:                     getEnv("LOG_LEVEL", ptr("info")),
		LogOutput:                    getEnv("LOG_OUTPUT", ptr("stdout")),
		I18nLocalesDir:               getEnv("I18N_LOCALES_DIR", ptr("./locales")),
	}

	if cfg.JWTSecret == nil {
		return nil, fmt.Errorf("jwt_secret is required")
	}

	if cfg.WalletEncryptionKey == nil {
		return nil, fmt.Errorf("wallet_encryption_key is required")
	}

	if keyLen := len([]byte(*cfg.WalletEncryptionKey)); keyLen != 32 {
		return nil, fmt.Errorf("wallet_encryption_key must be exactly 32 bytes (AES-256), got %d", keyLen)
	}

	return cfg, nil
}

func getEnv(key string, fallback *string) *string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return &val
}

// ptr returns a pointer to a copy of the given value.
// Useful for creating pointers from literals in struct initialization.
func ptr[T any](v T) *T {
	return &v
}

// getTimeEnv parses an environment variable as a time.Time in ISO 8601 format (YYYY-MM-DD).
// Returns nil if the environment variable is empty or parsing fails.
func getTimeEnv(key string, fallback *time.Time) *time.Time {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	t, err := time.Parse(time.DateOnly, val)
	if err != nil {
		return fallback
	}
	return &t
}

// getJSONEnv parses an environment variable as a JSON object into map[string]any.
// Returns nil if the environment variable is empty or parsing fails.
func getJSONEnv(key string, fallback map[string]any) map[string]any {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return fallback
	}
	return result
}

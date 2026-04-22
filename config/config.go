package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgresDSN              string
	MongoURI                 string
	JWTSecret                string
	WalletEncryptionKey      string
	GeminiAPIKey             string
	RPCURL                   string
	RegistryContract         string
	AuthorityContract        string
	RelayerPrivateKey        string
	InitialSuperAdminEmail   string
	InitialSuperAdminPrivKey string
	AppPort                  string

	// Database connection pool settings
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime int // in minutes
}

func getIntEnv(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

// NewConfig loads configuration from .env file
func NewConfig(envPath string) (*Config, error) {
	// Load .env file from specified path
	err := godotenv.Load(envPath)
	if err != nil {
		// Continue anyway - env vars might be set directly
	}

	cfg := &Config{
		PostgresDSN:              getEnv("POSTGRES_DSN", "postgres://root:rootpassword@localhost:5432/credchain?sslmode=disable"),
		MongoURI:                 getEnv("MONGO_URI", "mongodb://root:rootpassword@localhost:27017"),
		JWTSecret:                getEnv("JWT_SECRET", ""),
		WalletEncryptionKey:      getEnv("WALLET_ENCRYPTION_KEY", ""),
		GeminiAPIKey:             getEnv("GEMINI_API_KEY", ""),
		RPCURL:                   getEnv("RPC_URL", ""),
		RegistryContract:         getEnv("REGISTRY_CONTRACT", ""),
		AuthorityContract:        getEnv("AUTHORITY_CONTRACT", ""),
		RelayerPrivateKey:        getEnv("RELAYER_PRIVATE_KEY", ""),
		InitialSuperAdminEmail:   getEnv("INITIAL_SUPER_ADMIN_EMAIL", ""),
		InitialSuperAdminPrivKey: getEnv("INITIAL_SUPER_ADMIN_PRIVATE_KEY", ""),
		AppPort:                  getEnv("PORT", "8080"),

		// Database connection pool
		DBMaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetime: getIntEnv("DB_CONN_MAX_LIFETIME", 5),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("jwt_secret is required")
	}

	if cfg.WalletEncryptionKey == "" {
		return nil, fmt.Errorf("wallet_encryption_key is required")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		if fallback != "" {
			return fallback
		}
		return ""
	}
	return val
}

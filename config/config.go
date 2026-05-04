package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort                  string
	InitialSuperAdminEmail   string
	InitialSuperAdminPrivKey string
	WalletEncryptionKey      string
	RPCURL                   string
	RelayerPrivateKey        string
	AuthorityContract        string
	RegistryContract         string
	JWTSecret                string
	JWTAccessExpiryMinutes   int
	JWTRefreshExpiryHours    int
	PostgresUser             string
	PostgresPassword         string
	PostgresDB               string
	PostgresDSN              string
	DBMaxOpenConns           int
	DBMaxIdleConns           int
	DBConnMaxLifetime        int
	MongoInitDBUsername      string
	MongoInitPassword        string
	MongoURI                 string
	GeminiAPIKey             string
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

func NewConfig(envPath string) (*Config, error) {
	err := godotenv.Load(envPath)
	if err != nil {

	}

	cfg := &Config{
		AppPort:                  getEnv("APP_PORT", "8080"),
		InitialSuperAdminEmail:   getEnv("INIT_SUPER_ADMIN_EMAIL", ""),
		InitialSuperAdminPrivKey: getEnv("INIT_SUPER_ADMIN_PRIVATE_KEY", ""),
		WalletEncryptionKey:      getEnv("WALLET_ENCRYPTION_KEY", ""),
		RPCURL:                   getEnv("RPC_URL", ""),
		RelayerPrivateKey:        getEnv("RELAYER_PRIVATE_KEY", ""),
		AuthorityContract:        getEnv("AUTHORITY_CONTRACT", ""),
		RegistryContract:         getEnv("REGISTRY_CONTRACT", ""),
		JWTSecret:                getEnv("JWT_SECRET", ""),
		JWTAccessExpiryMinutes:   getIntEnv("JWT_ACCESS_EXPIRY_MINUTES", 15),
		JWTRefreshExpiryHours:    getIntEnv("JWT_REFRESH_EXPIRY_HOURS", 168),
		PostgresUser:             getEnv("POSTGRES_USER", ""),
		PostgresPassword:         getEnv("POSTGRES_PASSWORD", ""),
		PostgresDB:               getEnv("POSTGRES_DB", ""),
		PostgresDSN:              getEnv("POSTGRES_DSN", ""),
		DBMaxOpenConns:           getIntEnv("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:           getIntEnv("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetime:        getIntEnv("DB_CONN_MAX_LIFETIME", 5),
		MongoInitDBUsername:      getEnv("MONGO_INIT_DB_USERNAME", ""),
		MongoInitPassword:        getEnv("MONGO_INITDB_ROOT_PASSWORD", ""),
		MongoURI:                 getEnv("MONGO_URI", ""),
		GeminiAPIKey:             getEnv("GEMINI_API_KEY", ""),
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
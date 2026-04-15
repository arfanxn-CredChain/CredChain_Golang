package config

import (
	"fmt"
	"os"
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

func LoadConfig() (*Config, error) {
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
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("FATAL: JWT_SECRET is missing")
	}
	
	if cfg.WalletEncryptionKey == "" {
		return nil, fmt.Errorf("FATAL: WALLET_ENCRYPTION_KEY is missing")
	}

	return cfg, nil
}

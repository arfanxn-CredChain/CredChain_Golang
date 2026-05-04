package cmd

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"strings"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"

	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(initSuperAdminCmd)
}

func initSuperAdmin(db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if cfg.InitialSuperAdminEmail == "" || cfg.InitialSuperAdminPrivKey == "" || cfg.WalletEncryptionKey == "" {
		return fmt.Errorf("missing core environment variables for super admin initialization")
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.InitialSuperAdminPrivKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key format: %w", err)
	}

	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("failed to cast public key to ecdsa")
	}

	walletAddress := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	encryptionKey := make([]byte, 32)
	copy(encryptionKey, []byte(cfg.WalletEncryptionKey))

	var existingID string
	err = db.QueryRow("SELECT id FROM users WHERE email = $1", cfg.InitialSuperAdminEmail).Scan(&existingID)
	if err == nil {
		logger.Info("super admin already exists, skipping initialization")
		return nil
	}

	logger.Info("super admin not found, initializing")

	encryptedKey, err := cryptoInfra.Encrypt([]byte(cfg.InitialSuperAdminPrivKey), encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt super admin private key: %w", err)
	}

	id := ulid.Make().String()
	insertQuery := `
		INSERT INTO users (id, name, email, role, wallet_address, wallet_private_key)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = db.ExecContext(context.Background(), insertQuery,
		id, "Super Admin", cfg.InitialSuperAdminEmail, domain.RoleSuperAdmin, walletAddress, encryptedKey,
	)
	if err != nil {
		return fmt.Errorf("failed to insert super admin user: %w", err)
	}

	logger.Info("super admin initialized securely")
	return nil
}

var initSuperAdminCmd = &cobra.Command{
	Use:   "init-super-admin",
	Short: "Initializes the Super Admin based on .env config",
	Long:  "Creates the inaugural Super Admin securely in Postgres parsing the Ethereum Wallet automatically from the given initial Private Key.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := cmd.Context().Value(ConfigContextKey).(*config.Config)

		logger, _ := zap.NewProduction()
		defer logger.Sync()

		db, err := sql.Open("postgres", cfg.PostgresDSN)
		if err != nil {
			logger.Error("failed to connect to postgres", zap.Error(err))
			return err
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			logger.Error("postgres ping failed", zap.Error(err))
			return err
		}

		err = initSuperAdmin(db, cfg, logger)
		if err != nil {
			logger.Error("super admin initialization failed", zap.Error(err))
			return err
		}

		logger.Info("successfully ran init-super-admin")
		return nil
	},
}

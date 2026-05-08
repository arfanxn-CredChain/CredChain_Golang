package cmd

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/feature/user"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(initSuperAdminCmd)
}

// initSuperAdminValidateConfig checks required env vars for super admin initialization
func initSuperAdminValidateConfig(cfg *config.Config) error {
	if cfg.InitialSuperAdminEmail == nil || cfg.InitialSuperAdminPrivKey == nil || cfg.WalletEncryptionKey == nil {
		return fmt.Errorf("missing core environment variables for super admin initialization")
	}
	return nil
}

// initSuperAdminParseWallet derives wallet address from hex private key
func initSuperAdminParseWallet(privKeyHex string) (string, error) {
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(privKeyHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("invalid private key format: %w", err)
	}

	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("failed to cast public key to ecdsa")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA).Hex(), nil
}

// initSuperAdminEncryptKey encrypts private key for storage
func initSuperAdminEncryptKey(privKey, encryptionKey string) (string, error) {
	encryptionKeyBytes := make([]byte, 32)
	copy(encryptionKeyBytes, []byte(encryptionKey))

	encrypted, err := cryptoInfra.Encrypt([]byte(privKey), encryptionKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt private key: %w", err)
	}

	return encrypted, nil
}

// initSuperAdminBuildUser constructs domain.User with super admin fields
func initSuperAdminBuildUser(email, walletAddress, encryptedKey string) domain.User {
	name := "Super Admin"
	return domain.User{
		Name:                      &name,
		Email:                     email,
		Role:                      domain.RoleSuperAdmin,
		WalletAddress:             walletAddress,
		EncryptedWalletPrivateKey: encryptedKey,
	}
}

// initSuperAdmin is the main FX-invoked function
func initSuperAdmin(cfg *config.Config, userRepo domain.UserRepository, logger *zap.Logger) error {
	if err := initSuperAdminValidateConfig(cfg); err != nil {
		return err
	}

	walletAddress, err := initSuperAdminParseWallet(*cfg.InitialSuperAdminPrivKey)
	if err != nil {
		return err
	}

	encryptedKey, err := initSuperAdminEncryptKey(*cfg.InitialSuperAdminPrivKey, *cfg.WalletEncryptionKey)
	if err != nil {
		return err
	}

	existing, err := userRepo.FindByEmails(context.Background(), *cfg.InitialSuperAdminEmail)
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		msg := "super admin already exists"
		logger.Error(msg)
		return fmt.Errorf("%s", msg)
	}

	adminUser := initSuperAdminBuildUser(*cfg.InitialSuperAdminEmail, walletAddress, encryptedKey)

	_, err = userRepo.Store(context.Background(), adminUser)
	if err != nil {
		return err
	}

	logger.Info("super admin initialized")
	return nil
}

var initSuperAdminCmd = &cobra.Command{
	Use:   "init-super-admin",
	Short: "Initializes the Super Admin based on .env config",
	Long:  "Creates the inaugural Super Admin securely in Postgres, parsing the Ethereum Wallet automatically from the given initial Private Key.",
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			infraLogger.Module,
			fx.Provide(
				NewConfigFromCmd(cmd),
				gormInfra.NewGorm,
				user.NewGormUserRepository,
			),
			fx.Invoke(initSuperAdmin),
		).Run()
	},
}

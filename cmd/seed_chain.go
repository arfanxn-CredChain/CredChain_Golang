package cmd

import (
	"context"
	"fmt"
	"log"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var seedChainNames []string

func init() {
	rootCmd.AddCommand(seedChainCmd)
	seedChainCmd.Flags().StringArrayVar(&seedChainNames, "names", nil, "Specific seeder names to run (default: run all)")
}

var seedChainCmd = &cobra.Command{
	Use:   "seed-chain",
	Short: "Register seeded users on the blockchain",
	Long: `Reads users from the database and registers their roles on-chain via
CredentialAuthority.BatchUpdateUserRoleWithSignature.

The SuperAdmin wallet (derived from the Hardhat mnemonic at index 1) signs all
role-update transactions. Index 0 is reserved for the relayer.

Prerequisites:
  1. The 'seed' command must have been run first (users in DB)
  2. The SuperAdmin wallet must already have the SuperAdmin role on-chain
     (set during contract deployment)
  3. The RPC_URL must point to a running Hardhat node

Examples:
  go run main.go seed-chain --env .env
  go run main.go seed-chain --env .env --names user`,
	Run: func(cmd *cobra.Command, args []string) {
		app := fx.New(
			infraLogger.Module,
			fx.Provide(
				NewConfigFromCmd(cmd),
				gormInfra.NewGorm,
				user.NewGormUserRepository,
				chain.NewClient,
				chain.NewAuthorityService,
			),
			fx.Invoke(func(shutdowner fx.Shutdowner, cfg *config.Config, userRepo domain.UserRepository, authorityService chain.AuthorityService, logger *zap.Logger) {
				go func() {
					if err := seedChainRun(cfg, userRepo, authorityService, seedChainNames, logger); err != nil {
						logger.Error("seed-chain failed", zap.Error(err))
					}
					shutdowner.Shutdown()
				}()
			}),
		)

		if err := app.Start(context.Background()); err != nil {
			log.Fatal(err)
		}

		<-app.Done()
	},
}

// seedChainRun reads all users from PostgreSQL via a single Get query,
// derives the SuperAdmin wallet (mnemonic index 1), and registers every
// non-None-role user on-chain in one batch transaction.
func seedChainRun(cfg *config.Config, userRepo domain.UserRepository, authorityService chain.AuthorityService, names []string, logger *zap.Logger) error {
	ctx := context.Background()
	mnemonic := seedGetHardhatMnemonic(cfg)

	logger.Info("reading seeded users from database")

	// Single query: pass nil for no search/filter/pagination — helpers handle nil safely.
	allUsers, total, err := userRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed-chain: read users: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("seed-chain: no users in database — run 'seed' first")
	}

	// Build the on-chain update list. SuperAdmin is skipped (set during deploy,
	// contract blocks SuperAdmin updates via batchUpdateUserRoleWithSignature).
	// Deleted users get RoleNone on-chain while preserving their DB role.
	// Active users get their DB role as-is.
	var usersToRegister []domain.User
	for _, u := range allUsers {
		if u.Role == domain.RoleSuperAdmin {
			continue
		}
		update := u
		if u.DeletedAt != nil {
			update.Role = domain.RoleNone
		}
		usersToRegister = append(usersToRegister, update)
	}

	logger.Info("users loaded for chain registration",
		zap.Int("total_in_db", total),
		zap.Int("to_register", len(usersToRegister)),
	)

	// Derive SuperAdmin wallet from mnemonic index 1 (Muhammad Arfan).
	privKey, _, err := cryptoInfra.DeriveKeyFromMnemonic(mnemonic, 1)
	if err != nil {
		return fmt.Errorf("seed-chain: derive super admin key: %w", err)
	}

	encryptedKey, err := cryptoInfra.Encrypt([]byte(privKey), []byte(*cfg.WalletEncryptionKey))
	if err != nil {
		return fmt.Errorf("seed-chain: encrypt super admin key: %w", err)
	}

	addr, err := cryptoInfra.DeriveAddressFromPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("seed-chain: derive super admin address: %w", err)
	}

	superAdminWallet := domain.Wallet{
		Address:             addr,
		EncryptedPrivateKey: encryptedKey,
	}

	logger.Info("registering users on-chain",
		zap.Int("count", len(usersToRegister)),
		zap.String("signer", superAdminWallet.Address),
	)

	if err := authorityService.UpdateUserRole(ctx, superAdminWallet, usersToRegister...); err != nil {
		return fmt.Errorf("seed-chain: on-chain registration: %w", err)
	}

	logger.Info("seed-chain completed successfully",
		zap.Int("users_registered", len(usersToRegister)),
	)

	return nil
}

package cmd

import (
	"context"
	"fmt"
	"log"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/feature/credential"
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
				credential.NewGormCredentialRepository,
				chain.NewClient,
				chain.NewAuthorityService,
				chain.NewRegistryService,
			),
			fx.Invoke(func(shutdowner fx.Shutdowner, cfg *config.Config, userRepo domain.UserRepository, credentialRepo domain.CredentialRepository, authorityService chain.AuthorityService, registryService chain.RegistryService, logger *zap.Logger) {
				go func() {
					if err := seedChainRun(cfg, userRepo, credentialRepo, authorityService, registryService, seedChainNames, logger); err != nil {
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
//
// After role registration, it also mints credentials on-chain for any
// credentials whose TokenID is nil (not yet minted).
func seedChainRun(cfg *config.Config, userRepo domain.UserRepository, credentialRepo domain.CredentialRepository, authorityService chain.AuthorityService, registryService chain.RegistryService, names []string, logger *zap.Logger) error {
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
	// SameRoleUpdateError is avoided by also skipping users whose target role
	// matches their current on-chain role (RoleNone for new deployments).
	var usersToRegister []domain.User
	for _, u := range allUsers {
		if u.Role == domain.RoleSuperAdmin {
			continue
		}
		update := u
		if u.DeletedAt != nil {
			update.Role = domain.RoleNone
		}
		if update.Role == domain.RoleNone {
			continue // already None on-chain on fresh deploy
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

	const maxBatchRole = 100
	registeredCount := 0
	for start := 0; start < len(usersToRegister); start += maxBatchRole {
		end := start + maxBatchRole
		if end > len(usersToRegister) {
			end = len(usersToRegister)
		}
		chunk := usersToRegister[start:end]

		logger.Info("registering users on-chain",
			zap.Int("chunk_size", len(chunk)),
			zap.Int("chunk_start", start),
			zap.Int("total", len(usersToRegister)),
			zap.String("signer", superAdminWallet.Address),
		)

		if err := authorityService.UpdateUserRole(ctx, superAdminWallet, chunk...); err != nil {
			return fmt.Errorf("seed-chain: on-chain registration chunk [%d:%d]: %w", start, end, err)
		}
		registeredCount += len(chunk)
	}

	// ── Credential minting ──────────────────────────────────────────────

	logger.Info("reading credentials for on-chain minting")

	credQuery := &domainQuery.Query{
		Includes: []string{"holder"},
	}
	allCredentials, credTotal, err := credentialRepo.Get(ctx, credQuery)
	if err != nil {
		return fmt.Errorf("seed-chain: read credentials: %w", err)
	}

	var credentialsToMint []domain.Credential
	for _, c := range allCredentials {
		if c.TokenID != nil {
			continue
		}
		credentialsToMint = append(credentialsToMint, c)
	}

	logger.Info("credentials loaded for minting",
		zap.Int("total_in_db", credTotal),
		zap.Int("to_mint", len(credentialsToMint)),
	)

	const maxBatchCredential = 100
	mintedCount := 0
	for start := 0; start < len(credentialsToMint); start += maxBatchCredential {
		end := start + maxBatchCredential
		if end > len(credentialsToMint) {
			end = len(credentialsToMint)
		}
		chunk := credentialsToMint[start:end]

		issuances := make([]chain.CredentialIssuance, len(chunk))
		for i, c := range chunk {
			holderAddr := ""
			if c.Holder != nil {
				holderAddr = c.Holder.WalletAddress
			}
			uri := c.ID
			issuances[i] = chain.CredentialIssuance{
				HolderAddress: holderAddr,
				Hash:          c.FileHash,
				URI:           uri,
			}
		}

		logger.Info("minting credentials on-chain",
			zap.Int("chunk_size", len(chunk)),
			zap.Int("chunk_start", start),
			zap.Int("total", len(credentialsToMint)),
			zap.String("signer", superAdminWallet.Address),
		)

		tokenIds, err := registryService.IssueCredentials(ctx, superAdminWallet, issuances...)
		if err != nil {
			return fmt.Errorf("seed-chain: on-chain minting chunk [%d:%d]: %w", start, end, err)
		}

		for i := range chunk {
			tid := tokenIds[i].String()
			chunk[i].TokenID = &tid
		}
		if _, err := credentialRepo.Update(ctx, chunk...); err != nil {
			return fmt.Errorf("seed-chain: update token IDs chunk [%d:%d]: %w", start, end, err)
		}

		mintedCount += len(chunk)
	}

	logger.Info("seed-chain completed successfully",
		zap.Int("users_registered", registeredCount),
		zap.Int("credentials_minted", mintedCount),
	)

	return nil
}

package user

import (
	"context"
	"encoding/hex"
	"strings"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	httpContext "CredChain_Golang/infrastructure/http/context"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type UserService interface {
	// Query-based retrieval
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error)

	// Single item lookups
	Find(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// Multiple item lookups
	FindByIds(ctx context.Context, ids ...string) ([]domain.User, error)

	// Update operations
	Update(ctx context.Context, user domain.User) (*domain.User, error)
	UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta *domain.JSONB) (*domain.User, error)
	UpdateEmail(ctx context.Context, id string, email string) (string, error)
	UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error)

	// CRUD operations
	Store(ctx context.Context, users ...domain.User) ([]domain.User, error)
	Destroy(ctx context.Context, ids ...string) (int64, error)
}

type Service struct {
	userRepo            domain.UserRepository
	uow                 domain.UnitOfWork
	walletEncryptionKey string
	chainClient         *chain.Client
	logger              *zap.Logger
	policy              *UserPolicy
}

type UserServiceParams struct {
	fx.In
	UserRepo    domain.UserRepository
	UoW         domain.UnitOfWork
	Config      *config.Config
	ChainClient *chain.Client
	Logger      *zap.Logger
	Policy      *UserPolicy
}

func NewUserService(p UserServiceParams) *Service {
	return &Service{
		userRepo:            p.UserRepo,
		uow:                 p.UoW,
		walletEncryptionKey: p.Config.WalletEncryptionKey,
		chainClient:         p.ChainClient,
		logger:              p.Logger,
		policy:              p.Policy,
	}
}

// ============================================================================
// STORE - Create users with wallet generation and blockchain sync
// ============================================================================

// Store creates multiple users in a single transaction.
// It performs policy validation, email uniqueness checks, wallet generation,
// and blockchain role synchronization.
// Returns all created users or an error if any step fails.
func (s *Service) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	// 1. Policy validation
	if err := s.policy.Store(ctx, users...); err != nil {
		return nil, err
	}

	// 2. Email validation
	if err := s.storeValidateEmails(ctx, users); err != nil {
		return nil, err
	}

	// 3. Generate wallets
	if err := s.storeGenerateWallets(users); err != nil {
		return nil, err
	}

	// 4. Persist with blockchain
	return s.storeUsersAndSyncBlockchainRoles(ctx, users)
}

// storeValidateEmails checks for duplicate emails in the batch and database.
// Collects ALL duplicates before returning (not just the first one).
// Returns batch duplicates with priority over database duplicates.
func (s *Service) storeValidateEmails(ctx context.Context, users []domain.User) error {
	batchDuplicates := []string{}
	dbDuplicates := []string{}

	// 1. Check batch-internal duplicates - collect ALL
	emailIndex := make(map[string][]int)
	for i, u := range users {
		emailIndex[u.Email] = append(emailIndex[u.Email], i)
	}
	for email, indices := range emailIndex {
		if len(indices) > 1 {
			batchDuplicates = append(batchDuplicates, email)
		}
	}

	// 2. Check database duplicates only if batch is clean
	if len(batchDuplicates) == 0 {
		emails := make([]string, len(users))
		for i, u := range users {
			emails[i] = u.Email
		}

		existing, _ := s.userRepo.FindByEmails(ctx, emails...)
		if len(existing) > 0 {
			existingEmails := make(map[string]bool)
			for _, u := range existing {
				existingEmails[u.Email] = true
			}

			for _, u := range users {
				if existingEmails[u.Email] {
					dbDuplicates = append(dbDuplicates, u.Email)
				}
			}
		}
	}

	// 3. Return appropriate error
	if len(batchDuplicates) > 0 {
		return domain.NewError(
			domain.CodeUserStoreEmailDuplicateInBatch,
			domain.WithMetadata("emails", batchDuplicates),
		)
	}
	if len(dbDuplicates) > 0 {
		return domain.NewError(
			domain.CodeUserStoreEmailDuplicateInDatabase,
			domain.WithMetadata("emails", dbDuplicates),
		)
	}

	return nil
}

// storeGenerateWallets creates Ethereum wallets for all users.
// Generates private key, derives address, and encrypts private key.
// Returns error with WithError() for audit trail on failure.
func (s *Service) storeGenerateWallets(users []domain.User) error {
	for i := range users {
		key, err := ethCrypto.GenerateKey()
		if err != nil {
			return domain.NewError(
				domain.CodeUserStoreWalletGenerationFailed,
				domain.WithError(err),
			)
		}
		privateKeyHex := hex.EncodeToString(ethCrypto.FromECDSA(key))
		address := ethCrypto.PubkeyToAddress(key.PublicKey).Hex()

		encrypted, err := cryptoInfra.Encrypt([]byte(privateKeyHex), []byte(s.walletEncryptionKey))
		if err != nil {
			return domain.NewError(
				domain.CodeUserStoreWalletGenerationFailed,
				domain.WithError(err),
			)
		}

		users[i].WalletAddress = address
		users[i].WalletPrivateKey = encrypted
	}
	return nil
}

// storeUsersAndSyncBlockchainRoles persists users to database and updates roles on-chain.
// Uses UnitOfWork for atomic transaction (DB + blockchain all-or-nothing).
// Signs the operation with auth user's private key for blockchain authorization.
func (s *Service) storeUsersAndSyncBlockchainRoles(ctx context.Context, users []domain.User) ([]domain.User, error) {
	authUser := httpContext.MustGetUser(ctx)

	// 1. Prepare blockchain data
	targetUsers := make([]common.Address, len(users))
	newRoles := make([]uint8, len(users))
	for i, user := range users {
		targetUsers[i] = common.HexToAddress(user.WalletAddress)
		newRoles[i] = user.Role.ToUint8()
	}

	// 2. Fetch nonce from Registry
	authUserAddr := common.HexToAddress(authUser.WalletAddress)
	nonce, err := s.chainClient.FetchNonce(ctx, authUserAddr)
	if err != nil {
		return nil, domain.NewError(
			domain.CodeUserStoreBlockchainSyncFailed,
			domain.WithError(err),
		)
	}

	// 3. Decrypt auth user's private key for signing
	decryptedKey, err := cryptoInfra.Decrypt(authUser.WalletPrivateKey, []byte(s.walletEncryptionKey))
	if err != nil {
		return nil, domain.NewError(
			domain.CodeUserStoreBlockchainSyncFailed,
			domain.WithError(err),
		)
	}

	privateKey, err := ethCrypto.HexToECDSA(strings.TrimPrefix(string(decryptedKey), "0x"))
	if err != nil {
		return nil, domain.NewError(
			domain.CodeUserStoreBlockchainSyncFailed,
			domain.WithError(err),
		)
	}

	// 4. Sign the operation (match Solidity packing)
	var packed []byte
	packed = append(packed, authUserAddr.Bytes()...)
	for _, addr := range targetUsers {
		packed = append(packed, addr.Bytes()...)
	}
	for _, role := range newRoles {
		packed = append(packed, role)
	}
	packed = append(packed, common.LeftPadBytes(nonce.Bytes(), 32)...)

	digest := ethCrypto.Keccak256(packed)
	signature, err := ethCrypto.Sign(accounts.TextHash(digest), privateKey)
	if err != nil {
		return nil, domain.NewError(
			domain.CodeUserStoreBlockchainSyncFailed,
			domain.WithError(err),
		)
	}
	signature[64] += 27 // Ethereum V value

	// 5. Execute DB + Chain in transaction (all-or-nothing)
	var created []domain.User
	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		// 5a. Store to database
		created, err = uow.User().Store(ctx, users...)
		if err != nil {
			return err
		}

		// 5b. Call blockchain (if this fails, DB rolls back)
		tx, err := s.chainClient.Authority.BatchUpdateUserRoleWithSignature(
			s.chainClient.Relayer,
			authUserAddr,
			targetUsers,
			newRoles,
			nonce,
			signature,
		)
		if err != nil {
			return domain.NewError(
				domain.CodeUserStoreBlockchainSyncFailed,
				domain.WithError(err),
			)
		}

		// 5c. Log transaction
		s.logger.Info(
			"user roles updated on chain",
			zap.String("tx_hash", tx.Hash().Hex()),
			zap.Int("user_count", len(users)),
		)

		return nil
	})

	return created, err
}

// ============================================================================
// QUERY - Retrieve users
// ============================================================================

// Paginate retrieves users with pagination support.
// Returns users slice, total count, and any error.
func (s *Service) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	return s.userRepo.Get(ctx, query)
}

// Find retrieves a single user by ID.
// Returns error if user not found.
func (s *Service) Find(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.Find(ctx, id)
}

// FindByEmails retrieves multiple users by their email addresses.
// Returns empty slice if no users found (not an error).
func (s *Service) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) {
	return s.userRepo.FindByEmails(ctx, emails...)
}

// FindByIds retrieves multiple users by their IDs.
// Returns empty slice if no users found (not an error).
func (s *Service) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	return s.userRepo.FindByIds(ctx, ids...)
}

// ============================================================================
// UPDATE - Modify user data
// ============================================================================

// Update persists changes to a user entity.
// Caller must ensure the user exists.
func (s *Service) Update(ctx context.Context, user domain.User) (*domain.User, error) {
	return s.userRepo.Update(ctx, user)
}

// UpdateProfile updates user profile fields (name, number, phone, meta).
// Only provided fields are updated (nil fields are ignored).
func (s *Service) UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta *domain.JSONB) (*domain.User, error) {
	return s.userRepo.Update(ctx, domain.User{
		Id:          id,
		Name:        name,
		Number:      number,
		PhoneNumber: phoneNumber,
		Meta:        meta,
	})
}

// UpdateEmail changes a user's email address.
// Returns the new email address or an error if update fails.
func (s *Service) UpdateEmail(ctx context.Context, id string, email string) (string, error) {
	updated, err := s.userRepo.Update(ctx, domain.User{
		Id:    id,
		Email: email,
	})
	if err != nil {
		return "", err
	}
	return updated.Email, nil
}

// UpdateRole updates roles for multiple users in a single transaction.
// Performs authorization checks:
// - Admins cannot update other Admins or SuperAdmins
// - Admins cannot assign Admin or SuperAdmin roles
// - SuperAdmin role assignment is forbidden in batch operations
// Returns updated users, rows affected, and any error.
func (s *Service) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error) {
	authUser := httpContext.MustGetUser(ctx)

	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return nil, 0, domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden)
	}

	// Use UoW for transaction
	var updatedUsers []domain.User
	var rowsAffected int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		// Fetch all target users (1 query)
		userIDs := make([]string, len(updates))
		for i, u := range updates {
			userIDs[i] = u.UserID
		}

		targetUsers, err := uow.User().FindByIds(ctx, userIDs...)
		if err != nil {
			return err
		}

		// Build map for validation
		targetUserMap := make(map[string]domain.User)
		for _, tu := range targetUsers {
			targetUserMap[tu.Id] = tu
		}

		usersToUpdate := make([]domain.User, 0, len(updates))
		for _, update := range updates {
			targetUser, ok := targetUserMap[update.UserID]
			if !ok {
				return domain.NewError(domain.CodeUserFetchNotFound, domain.WithMetadata("user_id", update.UserID))
			}

			if targetUser.Role == update.Role {
				return domain.NewError(
					domain.CodeUserRoleSameRoleUpdateForbidden,
					domain.WithMetadata("user_id", update.UserID),
					domain.WithMetadata("current_role", targetUser.Role.String()),
				)
			}

			if authUser.Role == domain.RoleAdmin {
				if targetUser.Role.Rank() >= domain.RoleAdmin.Rank() {
					return domain.NewError(
						domain.CodeUserRoleAdminUpdatePeerForbidden,
						domain.WithMetadata("auth_user_id", authUser.Id),
						domain.WithMetadata("target_user_id", update.UserID),
					)
				}
				if update.Role.Rank() >= domain.RoleAdmin.Rank() {
					return domain.NewError(
						domain.CodeUserRoleSignerAdminRequiredForbidden,
						domain.WithMetadata("auth_user_id", authUser.Id),
						domain.WithMetadata("attempted_role", update.Role.String()),
					)
				}
			}

			if update.Role == domain.RoleSuperAdmin {
				return domain.NewError(
					domain.CodeUserRoleSuperAdminBatchForbidden,
					domain.WithMetadata("user_id", update.UserID),
					domain.WithMetadata("attempted_role", "super_admin"),
				)
			}

			// Prepare updated user
			targetUser.Role = update.Role
			usersToUpdate = append(usersToUpdate, targetUser)
		}

		// Batch update (1 efficient query)
		updatedUsers, rowsAffected, err = uow.User().UpdateRole(ctx, usersToUpdate...)
		return err
	})

	if err != nil {
		return nil, 0, err
	}

	return updatedUsers, rowsAffected, nil
}

// ============================================================================
// DELETE - Remove users
// ============================================================================

// Destroy deletes multiple users by their IDs.
// Performs authorization checks:
// - Users cannot delete themselves
// - Admins cannot delete other Admins or SuperAdmins
// Returns number of rows affected and any error.
func (s *Service) Destroy(ctx context.Context, ids ...string) (int64, error) {
	authUser := httpContext.MustGetUser(ctx)

	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return 0, domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden)
	}

	for _, id := range ids {
		if id == authUser.Id {
			return 0, domain.NewError(domain.CodeAuthLoginForbidden, domain.WithMetadata("user_id", authUser.Id))
		}
	}

	var rowsAffected int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targetUsers, err := uow.User().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}

		if len(targetUsers) == 0 {
			return nil
		}

		if authUser.Role.Rank() == domain.RoleAdmin.Rank() {
			for _, target := range targetUsers {
				if target.Role.Rank() >= domain.RoleAdmin.Rank() {
					return domain.NewError(
						domain.CodeUserDeleteAdminForbidden,
						domain.WithMetadata("auth_user_id", authUser.Id),
						domain.WithMetadata("target_user_id", target.Id),
						domain.WithMetadata("target_role", target.Role.String()),
					)
				}
			}
		}

		// Batch delete (1 query)
		rowsAffected, err = uow.User().Destroy(ctx, ids...)
		return err
	})

	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

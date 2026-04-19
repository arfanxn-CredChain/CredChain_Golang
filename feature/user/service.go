package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	httpContext "CredChain_Golang/infrastructure/http/context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/fx"
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
	UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) error

	// CRUD operations
	Store(ctx context.Context, users ...domain.User) ([]domain.User, error)
	Destroy(ctx context.Context, ids ...string) error
}

type Service struct {
	userRepo            domain.UserRepository
	walletEncryptionKey string
	chainClient         *chain.Client
}

type UserServiceParams struct {
	fx.In
	UserRepo    domain.UserRepository
	Config      *config.Config
	ChainClient *chain.Client
}

func NewUserService(p UserServiceParams) *Service {
	return &Service{
		userRepo:            p.UserRepo,
		walletEncryptionKey: p.Config.WalletEncryptionKey,
		chainClient:         p.ChainClient,
	}
}

// ... Implementation ...

func (s *Service) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	return s.userRepo.Store(ctx, users...)
}

func (s *Service) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	return s.userRepo.Get(ctx, query)
}

func (s *Service) Find(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.Find(ctx, id)
}

func (s *Service) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.userRepo.FindByEmail(ctx, email)
}

func (s *Service) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	return s.userRepo.FindByIds(ctx, ids...)
}

func (s *Service) Update(ctx context.Context, user domain.User) (*domain.User, error) {
	return s.userRepo.Update(ctx, user)
}

func (s *Service) UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta *domain.JSONB) (*domain.User, error) {
	return s.userRepo.Update(ctx, domain.User{
		Id:          id,
		Name:        name,
		Number:      number,
		PhoneNumber: phoneNumber,
		Meta:        meta,
	})
}

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

func (s *Service) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) error {
	// Extract userId from context (auth middleware injected it)
	userId, err := httpContext.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("missing user context: %w", err)
	}

	callerUser, err := s.userRepo.Find(ctx, userId)
	if err != nil {
		return fmt.Errorf("failed to fetch caller: %w", err)
	}

	if callerUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return fmt.Errorf("%d", domain.CodeUserRoleSignerAdminRequiredForbidden)
	}

	var userIDs []string
	for _, u := range updates {
		userIDs = append(userIDs, u.UserID)
	}

	targetUsers, err := s.userRepo.FindByIds(ctx, userIDs...)
	if err != nil {
		return err
	}

	targetUserMap := make(map[string]domain.User)
	for _, tu := range targetUsers {
		targetUserMap[tu.Id] = tu
	}

	for _, update := range updates {
		targetUser, ok := targetUserMap[update.UserID]
		if !ok {
			return errors.New("target user not found")
		}

		if callerUser.Role.Rank() == domain.RoleAdmin.Rank() {
			if targetUser.Role.Rank() >= domain.RoleAdmin.Rank() {
				return fmt.Errorf("%d", domain.CodeUserRoleAdminUpdatePeerForbidden)
			}
			if update.Role.Rank() >= domain.RoleAdmin.Rank() {
				return fmt.Errorf("%d", domain.CodeUserRoleSignerAdminRequiredForbidden)
			}
		}
	}

	encKey := s.walletEncryptionKey
	if encKey == "" {
		return fmt.Errorf("missing WALLET_ENCRYPTION_KEY env var")
	}
	encryptionKey := make([]byte, 32)
	copy(encryptionKey, []byte(encKey))

	decryptedKey, err := cryptoInfra.Decrypt(callerUser.WalletPrivateKey, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt caller private key: %w", err)
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(string(decryptedKey), "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse caller private key: %w", err)
	}

	callerWallet := common.HexToAddress(callerUser.WalletAddress)
	nonce, err := s.chainClient.FetchNonce(ctx, callerWallet)
	if err != nil {
		return fmt.Errorf("failed to fetch nonce: %w", err)
	}

	var packed []byte
	packed = append(packed, chain.EncodeAddress(callerUser.WalletAddress)...)
	for _, targetUser := range targetUsers {
		packed = append(packed, chain.EncodeAddress(targetUser.WalletAddress)...)
	}
	for _, update := range updates {
		packed = append(packed, byte(update.Role.Rank()))
	}
	nonceBytes, err := chain.EncodeUint256(nonce.String())
	if err != nil {
		return fmt.Errorf("failed to encode nonce: %w", err)
	}
	packed = append(packed, nonceBytes...)

	signature, err := chain.PackAndSign(privateKey, packed)
	if err != nil {
		return fmt.Errorf("failed to sign payload: %w", err)
	}

	targetAddrs := make([]common.Address, len(targetUsers))
	for i, tu := range targetUsers {
		targetAddrs[i] = common.HexToAddress(tu.WalletAddress)
	}

	newRoles := make([]uint8, len(updates))
	for i, u := range updates {
		newRoles[i] = uint8(u.Role.Rank())
	}

	tx, err := s.chainClient.DispatchRelayerTx(ctx, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return s.chainClient.Authority.BatchUpdateUserRoleWithSignature(
			opts,
			callerWallet,
			targetAddrs,
			newRoles,
			nonce,
			signature,
		)
	})
	if err != nil {
		return fmt.Errorf("failed to dispatch blockchain transaction: %w", err)
	}

	_ = tx

	return s.userRepo.UpdateRole(ctx, updates)
}

func (s *Service) Destroy(ctx context.Context, ids ...string) error {
	// Extract userId from context (auth middleware injected it)
	userId, err := httpContext.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("missing user context: %w", err)
	}

	callerUser, err := s.userRepo.Find(ctx, userId)
	if err != nil {
		return fmt.Errorf("failed to fetch caller: %w", err)
	}

	if callerUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return fmt.Errorf("%d", domain.CodeUserRoleSignerAdminRequiredForbidden)
	}

	// Users cannot delete themselves (any role)
	for _, id := range ids {
		if id == userId {
			return fmt.Errorf("users cannot delete their own account")
		}
	}

	targetUsers, err := s.userRepo.FindByIds(ctx, ids...)
	if err != nil {
		return err
	}

	if len(targetUsers) == 0 {
		return nil
	}

	// Admins cannot delete other admins
	if callerUser.Role.Rank() == domain.RoleAdmin.Rank() {
		for _, target := range targetUsers {
			if target.Role.Rank() >= domain.RoleAdmin.Rank() {
				return fmt.Errorf("admins cannot delete admin or super admin users")
			}
		}
	}

	encKey := s.walletEncryptionKey
	if encKey == "" {
		return fmt.Errorf("missing WALLET_ENCRYPTION_KEY env var")
	}
	encryptionKey := make([]byte, 32)
	copy(encryptionKey, []byte(encKey))

	decryptedKey, err := cryptoInfra.Decrypt(callerUser.WalletPrivateKey, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt caller private key: %w", err)
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(string(decryptedKey), "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse caller private key: %w", err)
	}

	callerWallet := common.HexToAddress(callerUser.WalletAddress)
	nonce, err := s.chainClient.FetchNonce(ctx, callerWallet)
	if err != nil {
		return fmt.Errorf("failed to fetch nonce: %w", err)
	}

	var packed []byte
	packed = append(packed, chain.EncodeAddress(callerUser.WalletAddress)...)
	for _, targetUser := range targetUsers {
		packed = append(packed, chain.EncodeAddress(targetUser.WalletAddress)...)
	}
	for i := 0; i < len(targetUsers); i++ {
		packed = append(packed, byte(0))
	}
	nonceBytes, err := chain.EncodeUint256(nonce.String())
	if err != nil {
		return fmt.Errorf("failed to encode nonce: %w", err)
	}
	packed = append(packed, nonceBytes...)

	signature, err := chain.PackAndSign(privateKey, packed)
	if err != nil {
		return fmt.Errorf("failed to sign payload: %w", err)
	}

	targetAddrs := make([]common.Address, len(targetUsers))
	for i, tu := range targetUsers {
		targetAddrs[i] = common.HexToAddress(tu.WalletAddress)
	}

	newRoles := make([]uint8, len(targetUsers))
	for i := range targetUsers {
		newRoles[i] = 0
	}

	tx, err := s.chainClient.DispatchRelayerTx(ctx, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return s.chainClient.Authority.BatchUpdateUserRoleWithSignature(
			opts,
			callerWallet,
			targetAddrs,
			newRoles,
			nonce,
			signature,
		)
	})
	if err != nil {
		return fmt.Errorf("failed to dispatch blockchain transaction: %w", err)
	}

	_ = tx

	return s.userRepo.Destroy(ctx, ids...)
}

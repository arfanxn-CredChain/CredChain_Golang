package user

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"strings"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oklog/ulid/v2"
	"go.uber.org/fx"
)

type Service struct {
	userRepo domain.UserRepository
}

type UserServiceParams struct {
	fx.In
	UserRepo domain.UserRepository
}

func NewService(p UserServiceParams) *Service {
	return &Service{userRepo: p.UserRepo}
}

// ... Implementation ...

func (s *Service) CreateUsers(ctx context.Context, newUsers []CreateUserRequest) ([]domain.User, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("missing JWT_SECRET env var required for encryption")
	}

	encryptionKey := make([]byte, 32)
	copy(encryptionKey, []byte(jwtSecret))

	var domainUsers []domain.User

	for _, nu := range newUsers {
		email := strings.ToLower(nu.Email)
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate ethereum wallet: %v", err)
		}
		privateKeyBytes := crypto.FromECDSA(privateKey)
		privateKeyHex := hexutil.Encode(privateKeyBytes)

		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("failed to cast public key")
		}
		walletAddress := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

		encryptedKey, err := database.Encrypt([]byte(privateKeyHex), encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt wallet key: %v", err)
		}

		domainUsers = append(domainUsers, domain.User{
			ID:               ulid.Make().String(),
			Name:             &nu.Name,
			Email:            email,
			Role:             nu.Role,
			WalletAddress:    strings.ToLower(walletAddress),
			WalletPrivateKey: encryptedKey,
		})
	}

	return s.userRepo.BatchCreate(ctx, domainUsers)
}

func (s *Service) GetUsers(ctx context.Context) ([]domain.User, error) {
	return s.userRepo.GetUsers(ctx)
}

func (s *Service) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

func (s *Service) UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta *domain.JSONB) (*domain.User, error) {
	return s.userRepo.UpdateProfile(ctx, id, name, number, phoneNumber, meta)
}

func (s *Service) UpdateEmail(ctx context.Context, id string, email string) (string, error) {
	return s.userRepo.UpdateEmail(ctx, id, email)
}

func (s *Service) BatchUpdateRole(ctx context.Context, callerRole domain.Role, updates []domain.UserRoleUpdate) error {
	if callerRole.Rank() < domain.RoleAdmin.Rank() {
		return fmt.Errorf("%d", domain.CodeUserRoleSignerAdminRequiredForbidden)
	}

	var userIDs []string
	for _, u := range updates {
		userIDs = append(userIDs, u.UserID)
	}

	targetUsers, err := s.userRepo.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		return err
	}
	
	targetUserMap := make(map[string]domain.User)
	for _, tu := range targetUsers {
		targetUserMap[tu.ID] = tu
	}

	for _, update := range updates {
		targetUser, ok := targetUserMap[update.UserID]
		if !ok {
			return errors.New("target user not found")
		}

		if callerRole.Rank() == domain.RoleAdmin.Rank() {
			if targetUser.Role.Rank() >= domain.RoleAdmin.Rank() {
				return fmt.Errorf("%d", domain.CodeUserRoleAdminUpdatePeerForbidden)
			}
			if update.Role.Rank() >= domain.RoleAdmin.Rank() {
				return fmt.Errorf("%d", domain.CodeUserRoleSignerAdminRequiredForbidden)
			}
		}
	}

	return s.userRepo.BatchUpdateRole(ctx, updates)
}

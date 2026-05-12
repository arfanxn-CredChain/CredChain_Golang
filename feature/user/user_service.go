package user

import (
	"context"
	"encoding/hex"
	"slices"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	httpContext "CredChain_Golang/infrastructure/http/context"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type UserService interface {
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error)
	Find(ctx context.Context, id string) (*domain.User, error)
	FindByIds(ctx context.Context, ids ...string) ([]domain.User, error)
	Update(ctx context.Context, user domain.User) (*domain.User, error)
	UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta map[string]any) (*domain.User, error)
	UpdateEmail(ctx context.Context, id string, email string) (string, error)
	UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error)
	Store(ctx context.Context, users ...domain.User) ([]domain.User, error)
	Destroy(ctx context.Context, ids ...string) (int64, error)
}

type userService struct {
	userRepo         domain.UserRepository
	uow              domain.UnitOfWork
	walletEncryptionKey string
	authorityService chain.AuthorityService
	logger           *zap.Logger
	policy           UserPolicy
}

type UserServiceParams struct {
	fx.In
	UserRepo         domain.UserRepository
	UoW              domain.UnitOfWork
	Config           *config.Config
	AuthorityService chain.AuthorityService
	Logger           *zap.Logger
	Policy           UserPolicy
}

func NewUserService(p UserServiceParams) UserService {
	return &userService{
		userRepo:            p.UserRepo,
		uow:                 p.UoW,
		walletEncryptionKey: *p.Config.WalletEncryptionKey,
		authorityService:    p.AuthorityService,
		logger:              p.Logger,
		policy:              p.Policy,
	}
}

func (s *userService) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	if err := s.policy.Store(ctx, users...); err != nil {
		return nil, err
	}
	if err := s.storeValidateEmails(ctx, users); err != nil {
		return nil, err
	}
	if err := s.storeGenerateWallets(users); err != nil {
		return nil, err
	}
	return s.storeUsersAndSyncBlockchainRoles(ctx, users)
}

func (s *userService) storeValidateEmails(ctx context.Context, users []domain.User) error {
	batchDuplicates := []string{}
	dbDuplicates := []string{}
	emailIndex := make(map[string][]int)
	for i, u := range users {
		emailIndex[u.Email] = append(emailIndex[u.Email], i)
	}
	for email, indices := range emailIndex {
		if len(indices) > 1 {
			batchDuplicates = append(batchDuplicates, email)
		}
	}
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
	if len(batchDuplicates) > 0 {
		return domain.NewError(domain.CodeUserStoreEmailDuplicateInBatch, domain.WithMetadata("emails", batchDuplicates))
	}
	if len(dbDuplicates) > 0 {
		return domain.NewError(domain.CodeUserStoreEmailDuplicateInDatabase, domain.WithMetadata("emails", dbDuplicates))
	}
	return nil
}

func (s *userService) storeGenerateWallets(users []domain.User) error {
	for i := range users {
		key, err := ethCrypto.GenerateKey()
		if err != nil {
			return domain.NewError(domain.CodeUserStoreWalletGenerationFailed, domain.WithError(err))
		}
		privateKeyHex := hex.EncodeToString(ethCrypto.FromECDSA(key))
		address := ethCrypto.PubkeyToAddress(key.PublicKey).Hex()
		encrypted, err := cryptoInfra.Encrypt([]byte(privateKeyHex), []byte(s.walletEncryptionKey))
		if err != nil {
			return domain.NewError(domain.CodeUserStoreWalletGenerationFailed, domain.WithError(err))
		}
		users[i].WalletAddress = address
		users[i].EncryptedWalletPrivateKey = encrypted
	}
	return nil
}

func (s *userService) storeUsersAndSyncBlockchainRoles(ctx context.Context, users []domain.User) ([]domain.User, error) {
	var created []domain.User
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		var err error
		created, err = uow.User().Store(ctx, users...)
		if err != nil {
			return err
		}

		err = s.authorityService.UpdateUserRole(ctx, users...)
		if err != nil {
			return domain.NewError(domain.CodeUserStoreBlockchainSyncFailed, domain.WithError(err))
		}

		return nil
	})
	return created, err
}

func (s *userService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	return s.userRepo.Get(ctx, query)
}

func (s *userService) Find(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.Find(ctx, id)
}

func (s *userService) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) {
	return s.userRepo.FindByEmails(ctx, emails...)
}

func (s *userService) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	return s.userRepo.FindByIds(ctx, ids...)
}

func (s *userService) Update(ctx context.Context, user domain.User) (*domain.User, error) {
	return s.userRepo.Update(ctx, user)
}

func (s *userService) UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta map[string]any) (*domain.User, error) {
	return s.userRepo.Update(ctx, domain.User{Id: id, Name: name, Number: number, PhoneNumber: phoneNumber, Meta: meta})
}

func (s *userService) UpdateEmail(ctx context.Context, id string, email string) (string, error) {
	updated, err := s.userRepo.Update(ctx, domain.User{Id: id, Email: email})
	if err != nil {
		return "", err
	}
	return updated.Email, nil
}

func (s *userService) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error) {
	authUser := httpContext.MustGetUser(ctx)
	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return nil, 0, domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden)
	}
	var updatedUsers []domain.User
	var rowsAffected int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		userIDs := make([]string, len(updates))
		for i, u := range updates {
			userIDs[i] = u.UserID
		}
		targetUsers, err := uow.User().FindByIds(ctx, userIDs...)
		if err != nil {
			return err
		}
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
				return domain.NewError(domain.CodeUserRoleSameRoleUpdateForbidden, domain.WithMetadata("user_id", update.UserID), domain.WithMetadata("current_role", targetUser.Role.String()))
			}
			if authUser.Role == domain.RoleAdmin {
				if targetUser.Role.Rank() >= domain.RoleAdmin.Rank() {
					return domain.NewError(domain.CodeUserRoleAdminUpdatePeerForbidden, domain.WithMetadata("auth_user_id", authUser.Id), domain.WithMetadata("target_user_id", update.UserID))
				}
				if update.Role.Rank() >= domain.RoleAdmin.Rank() {
					return domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden, domain.WithMetadata("auth_user_id", authUser.Id), domain.WithMetadata("attempted_role", update.Role.String()))
				}
			}
			if update.Role == domain.RoleSuperAdmin {
				return domain.NewError(domain.CodeUserRoleSuperAdminBatchForbidden, domain.WithMetadata("user_id", update.UserID), domain.WithMetadata("attempted_role", "super_admin"))
			}
			targetUser.Role = update.Role
			usersToUpdate = append(usersToUpdate, targetUser)
		}
		updatedUsers, rowsAffected, err = uow.User().UpdateRole(ctx, usersToUpdate...)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	return updatedUsers, rowsAffected, nil
}

func (s *userService) Destroy(ctx context.Context, ids ...string) (int64, error) {
	authUser := httpContext.MustGetUser(ctx)
	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return 0, domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden)
	}
	if slices.Contains(ids, authUser.Id) {
		return 0, domain.NewError(domain.CodeAuthForbidden, domain.WithMetadata("user_id", authUser.Id))
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
					return domain.NewError(domain.CodeUserDeleteAdminForbidden, domain.WithMetadata("auth_user_id", authUser.Id), domain.WithMetadata("target_user_id", target.Id), domain.WithMetadata("target_role", target.Role.String()))
				}
			}
		}
		rowsAffected, err = uow.User().Destroy(ctx, ids...)
		return err
	})
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

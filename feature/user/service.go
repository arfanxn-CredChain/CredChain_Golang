package user

import (
	"context"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/chain"
	httpContext "CredChain_Golang/infrastructure/http/context"

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
	uow                 domain.UnitOfWork
	walletEncryptionKey string
	chainClient         *chain.Client
}

type UserServiceParams struct {
	fx.In
	UserRepo    domain.UserRepository
	UoW         domain.UnitOfWork
	Config      *config.Config
	ChainClient *chain.Client
}

func NewUserService(p UserServiceParams) *Service {
	return &Service{
		userRepo:            p.UserRepo,
		uow:                 p.UoW,
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
	authUserID, err := httpContext.GetUserId(ctx)
	if err != nil {
		return domain.NewError(domain.CodeAuthLoginUnauthorized, domain.WithError(err))
	}

	authUser, err := s.userRepo.Find(ctx, authUserID)
	if err != nil {
		return domain.NewError(domain.CodeUserFetchNotFound, domain.WithError(err))
	}

	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden)
	}

	// Use UoW for transaction
	return s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
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
						domain.WithMetadata("auth_user_id", authUserID),
						domain.WithMetadata("target_user_id", update.UserID),
					)
				}
				if update.Role.Rank() >= domain.RoleAdmin.Rank() {
					return domain.NewError(
						domain.CodeUserRoleSignerAdminRequiredForbidden,
						domain.WithMetadata("auth_user_id", authUserID),
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
		return uow.User().UpdateRole(ctx, usersToUpdate...)
	})
}
func (s *Service) Destroy(ctx context.Context, ids ...string) error {
	authUserID, err := httpContext.GetUserId(ctx)
	if err != nil {
		return domain.NewError(domain.CodeAuthLoginUnauthorized, domain.WithError(err))
	}

	authUser, err := s.userRepo.Find(ctx, authUserID)
	if err != nil {
		return domain.NewError(domain.CodeUserFetchNotFound, domain.WithError(err))
	}

	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden)
	}

	for _, id := range ids {
		if id == authUserID {
			return domain.NewError(domain.CodeAuthLoginForbidden, domain.WithMetadata("user_id", authUserID))
		}
	}

	return s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
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
						domain.WithMetadata("auth_user_id", authUserID),
						domain.WithMetadata("target_user_id", target.Id),
						domain.WithMetadata("target_role", target.Role.String()),
					)
				}
			}
		}

		// Blockchain logic (outside DB transaction - it's external)
		// ... (keep existing blockchain logic here if needed)

		// Batch delete (1 query)
		return uow.User().Destroy(ctx, ids...)
	})
}

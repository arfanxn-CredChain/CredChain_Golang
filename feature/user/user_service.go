package user

import (
	"context"
	"encoding/hex"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/oauth"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type UserService interface {
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error)
	Find(ctx context.Context, id string) (*domain.User, error)
	FindByIds(ctx context.Context, ids ...string) ([]domain.User, error)
	Update(ctx context.Context, users ...domain.User) ([]domain.User, error)
	UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, birthDate *time.Time, meta map[string]any) (*domain.User, error)
	UpdateEmail(ctx context.Context, id string, email string, idToken string) (string, error)
	UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error)
	Store(ctx context.Context, users ...domain.User) ([]domain.User, error)
	Delete(ctx context.Context, ids ...string) (int64, error)
}

type userService struct {
	userRepo         domain.UserRepository
	uow              domain.UnitOfWork
	cfg              *config.Config
	authorityService chain.AuthorityService
	logger           *zap.Logger
	policy           UserPolicy
	oauthClient      oauth.GoogleOAuthClient
}

type UserServiceParams struct {
	fx.In
	UserRepo         domain.UserRepository
	UoW              domain.UnitOfWork
	Config           *config.Config
	AuthorityService chain.AuthorityService
	Logger           *zap.Logger
	Policy           UserPolicy
	OAuthClient      oauth.GoogleOAuthClient
}

func NewUserService(p UserServiceParams) UserService {
	return &userService{
		userRepo:         p.UserRepo,
		uow:              p.UoW,
		cfg:              p.Config,
		authorityService: p.AuthorityService,
		logger:           p.Logger,
		policy:           p.Policy,
		oauthClient:      p.OAuthClient,
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
		encrypted, err := cryptoInfra.Encrypt([]byte(privateKeyHex), []byte(*s.cfg.WalletEncryptionKey))
		if err != nil {
			return domain.NewError(domain.CodeUserStoreWalletGenerationFailed, domain.WithError(err))
		}
		users[i].WalletAddress = address
		users[i].EncryptedWalletPrivateKey = encrypted
	}
	return nil
}

func (s *userService) storeUsersAndSyncBlockchainRoles(ctx context.Context, users []domain.User) ([]domain.User, error) {
	authUser := httpContext.MustGetUser(ctx)
	wallet := domain.WalletFromUser(*authUser)

	var created []domain.User
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		var err error
		created, err = uow.User().Store(ctx, users...)
		if err != nil {
			return err
		}

		err = s.authorityService.UpdateUserRole(ctx, wallet, users...)
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

func (s *userService) Update(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	if err := s.policy.UpdatePreFetch(ctx, users...); err != nil {
		return nil, err
	}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}
	var updated []domain.User
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.User().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}
		if len(targets) != len(users) {
			return domain.NewError(domain.CodeUserUpdateNotFound)
		}
		if err := s.policy.UpdatePostFetch(ctx, targets); err != nil {
			return err
		}
		updated, err = uow.User().Update(ctx, users...)
		return err
	})
	return updated, err
}

func (s *userService) UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, birthDate *time.Time, meta map[string]any) (*domain.User, error) {
	updated, err := s.userRepo.Update(ctx, domain.User{Id: id, Name: name, Number: number, PhoneNumber: phoneNumber, BirthDate: birthDate, Meta: meta})
	if err != nil {
		return nil, err
	}
	if len(updated) == 0 {
		return nil, domain.NewError(domain.CodeUserFetchNotFound, domain.WithMetadata("user_id", id))
	}
	return &updated[0], nil
}

func (s *userService) UpdateEmail(ctx context.Context, id string, email string, idToken string) (string, error) {
	payload, err := s.oauthClient.Validate(ctx, idToken, *s.cfg.GoogleClientID)
	if err != nil {
		return "", domain.NewError(domain.CodeUserEmailInvalidIdToken, domain.WithError(err))
	}
	tokenEmail, _ := payload.Claims["email"].(string)
	if tokenEmail != email {
		return "", domain.NewError(domain.CodeUserEmailMismatchedIdToken,
			domain.WithMetadata("token_email", tokenEmail),
			domain.WithMetadata("requested_email", email))
	}
	existing, err := s.userRepo.FindByEmails(ctx, email)
	if err != nil {
		return "", err
	}
	for _, u := range existing {
		if u.Id != id {
			return "", domain.NewError(domain.CodeUserEmailConflict, domain.WithMetadata("email", email))
		}
	}
	var updatedEmail string
	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		updated, err := uow.User().Update(ctx, domain.User{Id: id, Email: email})
		if err != nil {
			return err
		}
		if len(updated) == 0 {
			return domain.NewError(domain.CodeUserFetchNotFound, domain.WithMetadata("user_id", id))
		}
		updatedEmail = updated[0].Email
		_, err = uow.UserToken().RevokeByUserIdAndType(ctx, id, domain.UserTokenTypeRefresh)
		return err
	})
	if err != nil {
		return "", err
	}
	return updatedEmail, nil
}

func (s *userService) updateRoleValidateAndPrepare(ctx context.Context, updates []domain.UserRoleUpdate, uow domain.UnitOfWork) ([]domain.User, error) {
	userIDs := make([]string, len(updates))
	for i, u := range updates {
		userIDs[i] = u.UserID
	}
	targetUsers, err := uow.User().FindByIds(ctx, userIDs...)
	if err != nil {
		return nil, err
	}
	if err := s.policy.UpdateRolePostFetch(ctx, targetUsers, updates...); err != nil {
		return nil, err
	}
	targetMap := make(map[string]domain.User, len(targetUsers))
	for _, t := range targetUsers {
		targetMap[t.Id] = t
	}
	usersToUpdate := make([]domain.User, 0, len(updates))
	for _, u := range updates {
		target := targetMap[u.UserID]
		target.Role = u.Role
		usersToUpdate = append(usersToUpdate, target)
	}
	return usersToUpdate, nil
}

func (s *userService) updateRoleAndSyncBlockchainRoles(ctx context.Context, usersToUpdate []domain.User) ([]domain.User, int64, error) {
	authUser := httpContext.MustGetUser(ctx)
	wallet := domain.WalletFromUser(*authUser)
	var updatedUsers []domain.User
	var rowsAffected int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		var err error
		updatedUsers, rowsAffected, err = uow.User().UpdateRole(ctx, usersToUpdate...)
		if err != nil {
			return err
		}
		if err := s.authorityService.UpdateUserRole(ctx, wallet, usersToUpdate...); err != nil {
			return domain.NewError(domain.CodeUserRoleBlockchainSyncFailed, domain.WithError(err))
		}
		return nil
	})
	return updatedUsers, rowsAffected, err
}

func (s *userService) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error) {
	if err := s.policy.UpdateRolePreFetch(ctx, updates...); err != nil {
		return nil, 0, err
	}
	var usersToUpdate []domain.User
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		var err error
		usersToUpdate, err = s.updateRoleValidateAndPrepare(ctx, updates, uow)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	return s.updateRoleAndSyncBlockchainRoles(ctx, usersToUpdate)
}

func (s *userService) deleteValidateAndPrepare(ctx context.Context, ids []string, uow domain.UnitOfWork) ([]domain.User, error) {
	targetUsers, err := uow.User().FindByIds(ctx, ids...)
	if err != nil {
		return nil, err
	}
	if err := s.policy.DeletePostFetch(ctx, targetUsers); err != nil {
		return nil, err
	}
	return targetUsers, nil
}

func (s *userService) deleteUserAndSyncBlockchain(ctx context.Context, ids []string, targetUsers []domain.User) (int64, error) {
	authUser := httpContext.MustGetUser(ctx)
	wallet := domain.WalletFromUser(*authUser)
	var rowsAffected int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		var err error
		rowsAffected, err = uow.User().Delete(ctx, ids...)
		if err != nil {
			return err
		}
		revocationUsers := make([]domain.User, len(targetUsers))
		for i, t := range targetUsers {
			revocationUsers[i] = domain.User{
				WalletAddress:             t.WalletAddress,
				EncryptedWalletPrivateKey: t.EncryptedWalletPrivateKey,
				Role:                      domain.RoleNone,
			}
		}
		// TODO: revoke user's credentials in DB (mark revoked_at, revoker_user_id)
		// and on-chain via CredentialRegistry.batchRevokeCredentialsWithSignature
		// once the credential feature is implemented. Without this, deleted users'
		// credentials remain active in the database and on-chain indefinitely.
		if err := s.authorityService.UpdateUserRole(ctx, wallet, revocationUsers...); err != nil {
			return domain.NewError(domain.CodeUserDeleteBlockchainSyncFailed, domain.WithError(err))
		}
		return nil
	})
	return rowsAffected, err
}

func (s *userService) Delete(ctx context.Context, ids ...string) (int64, error) {
	if err := s.policy.DeletePreFetch(ctx, ids...); err != nil {
		return 0, err
	}
	var targetUsers []domain.User
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		var err error
		targetUsers, err = s.deleteValidateAndPrepare(ctx, ids, uow)
		return err
	})
	if err != nil {
		return 0, err
	}
	if len(targetUsers) == 0 {
		return 0, nil
	}
	return s.deleteUserAndSyncBlockchain(ctx, ids, targetUsers)
}

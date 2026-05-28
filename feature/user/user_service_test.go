package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/oauth"
	"CredChain_Golang/infrastructure/testutil/fixtures"
	"CredChain_Golang/infrastructure/testutil/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"google.golang.org/api/idtoken"
)

func mkSvcCfg() *config.Config {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = 0xAA
	}
	s := string(raw)
	return &config.Config{JWTSecret: &s, WalletEncryptionKey: &s}
}

func ctxWithAuth(u *domain.User) context.Context {
	return context.WithValue(context.Background(), httpContext.UserKey, u)
}

func TestUserService_Store_PolicyFails(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	policy.On("Store", mock.Anything, mock.Anything).Return(errors.New("policy denied"))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Store(ctxWithAuth(&authUser), fixtures.NewDomainUser())
	assert.Error(t, err)
}

func TestUserService_Store_BatchDuplicateEmails(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	policy.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	u1 := fixtures.NewDomainUser(fixtures.WithEmail("dup@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithEmail("dup@x.com"))
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Store(ctxWithAuth(&authUser), u1, u2)
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserStoreEmailDuplicateInBatch, de.Code)
}

func TestUserService_Store_DBDuplicateEmails(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	policy.On("Store", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return(
		[]domain.User{fixtures.NewDomainUser(fixtures.WithEmail("dup@x.com"))}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	u1 := fixtures.NewDomainUser(fixtures.WithEmail("new@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithEmail("dup@x.com"))
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Store(ctxWithAuth(&authUser), u1, u2)
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserStoreEmailDuplicateInDatabase, de.Code)
}

func TestUserService_Paginate(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	repo.On("Get", mock.Anything, mock.Anything).Return([]domain.User{}, 0, nil)
	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})

	_, _, err := svc.Paginate(context.Background(), &domainQuery.Query{})
	assert.NoError(t, err)
}

func TestUserService_Find(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	u := fixtures.NewDomainUser()
	repo.On("Find", mock.Anything, "u1").Return(&u, nil)
	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})

	_, err := svc.Find(context.Background(), "u1")
	assert.NoError(t, err)
}

func TestUserService_Find_PropagatesError(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	repo.On("Find", mock.Anything, "missing").Return(nil, errors.New("not found"))
	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})

	_, err := svc.Find(context.Background(), "missing")
	assert.Error(t, err)
}

func TestUserService_UpdateProfile(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	updated := fixtures.NewDomainUser(fixtures.WithID("u1"))
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{updated}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})
	name := "Alice"
	got, err := svc.UpdateProfile(context.Background(), "u1", &name, nil, nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "u1", got.Id)
}

func TestUserService_UpdateProfile_PassesBirthDate(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	bd := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := fixtures.NewDomainUser(fixtures.WithID("u1"))
	updated.BirthDate = &bd
	repo.On("Update", mock.Anything, mock.MatchedBy(func(users []domain.User) bool {
		return len(users) == 1 && users[0].Id == "u1" && users[0].BirthDate != nil && users[0].BirthDate.Year() == 1990
	})).Return([]domain.User{updated}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})
	name := "Alice"
	got, err := svc.UpdateProfile(context.Background(), "u1", &name, nil, nil, &bd, nil)
	assert.NoError(t, err)
	assert.Equal(t, "u1", got.Id)
}

func TestUserService_UpdateEmail(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", mock.Anything).Return(&idtoken.Payload{
		Claims: map[string]any{"email": "new@x.com"},
	}, nil)
	repo := &mocks.MockUserRepository{}
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	u := fixtures.NewDomainUser(fixtures.WithEmail("new@x.com"))
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{u}, nil)

	tokenRepo := &mocks.MockUserTokenRepository{}
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)

	svc := newEmailSvc(oauthClient, repo, uow)
	got, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "ok-token")
	assert.NoError(t, err)
	assert.Equal(t, "new@x.com", got)
}

func TestUserService_UpdateRole_SignerBelowAdmin(t *testing.T) {
	svc := NewUserService(UserServiceParams{
		UserRepo: &mocks.MockUserRepository{}, UoW: &mocks.MockUnitOfWork{}, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: NewUserPolicy(),
	})

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	_, _, err := svc.UpdateRole(ctxWithAuth(&authUser),
		domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSignerAdminRequiredForbidden, de.Code)
}

func TestUserService_Delete_SignerBelowAdmin(t *testing.T) {
	svc := NewUserService(UserServiceParams{
		UserRepo: &mocks.MockUserRepository{}, UoW: &mocks.MockUnitOfWork{}, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: NewUserPolicy(),
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	_, err := svc.Delete(ctxWithAuth(&authUser), "u1")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSignerAdminRequiredForbidden, de.Code)
}

func TestUserService_Delete_SelfDeleteForbidden(t *testing.T) {
	svc := NewUserService(UserServiceParams{
		UserRepo: &mocks.MockUserRepository{}, UoW: &mocks.MockUnitOfWork{}, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: NewUserPolicy(),
	})

	authUser := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Delete(ctxWithAuth(&authUser), "self")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeAuthForbidden, de.Code)
}

func TestUserService_UpdateRole_BlockchainSyncFailure_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))

	policy.On("UpdateRolePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdateRolePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("UpdateRole", mock.Anything, mock.Anything).Return([]domain.User{target}, 1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("chain error"))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	_, _, err := svc.UpdateRole(ctxWithAuth(&authUser), domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleBlockchainSyncFailed, de.Code)
}

func TestUserService_UpdateRole_Success_CallsAuthorityService(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))

	policy.On("UpdateRolePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdateRolePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("UpdateRole", mock.Anything, mock.Anything).Return([]domain.User{target}, 1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	users, _, err := svc.UpdateRole(ctxWithAuth(&authUser), domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	auth.AssertCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Delete_BlockchainRevokeFailure_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))

	policy.On("DeletePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("DeletePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Delete", mock.Anything, mock.Anything).Return(1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("chain error"))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	_, err := svc.Delete(ctxWithAuth(&authUser), "u1")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserDeleteBlockchainSyncFailed, de.Code)
}

func TestUserService_Delete_Success_CallsAuthorityServiceWithRoleNone(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))

	policy.On("DeletePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("DeletePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Delete", mock.Anything, mock.Anything).Return(1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.MatchedBy(func(users []domain.User) bool {
		return len(users) == 1 && users[0].Role == domain.RoleNone
	})).Return(nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	count, err := svc.Delete(ctxWithAuth(&authUser), "u1")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, count)
	auth.AssertCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func newEmailSvc(oauthClient oauth.GoogleOAuthClient, repo domain.UserRepository, uow domain.UnitOfWork) UserService {
	if repo == nil {
		repo = &mocks.MockUserRepository{}
	}
	if uow == nil {
		uow = &mocks.MockUnitOfWork{}
	}
	clientID := ""
	walletKey := string(make([]byte, 32))
	jwt := "s"
	return NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: &config.Config{
			JWTSecret:           &jwt,
			WalletEncryptionKey: &walletKey,
			GoogleClientID:      &clientID,
		},
		AuthorityService: &mocks.MockAuthorityService{},
		Logger:           zap.NewNop(),
		Policy:           &mocks.MockUserPolicy{},
		OAuthClient:      oauthClient,
	})
}

func TestUserService_UpdateEmail_InvalidIdToken_Rejects(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "bad-token", mock.Anything).Return(nil, errors.New("invalid"))

	svc := newEmailSvc(oauthClient, nil, nil)
	_, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "bad-token")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserEmailInvalidIdToken, de.Code)
}

func TestUserService_UpdateEmail_MismatchedEmail_Rejects(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", mock.Anything).Return(&idtoken.Payload{
		Claims: map[string]any{"email": "other@x.com"},
	}, nil)

	svc := newEmailSvc(oauthClient, nil, nil)
	_, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "ok-token")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserEmailMismatchedIdToken, de.Code)
}

func TestUserService_UpdateEmail_ConflictWithExistingUser_Rejects(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", mock.Anything).Return(&idtoken.Payload{
		Claims: map[string]any{"email": "new@x.com"},
	}, nil)
	repo := &mocks.MockUserRepository{}
	existing := fixtures.NewDomainUser(fixtures.WithID("other"), fixtures.WithEmail("new@x.com"))
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{existing}, nil)

	svc := newEmailSvc(oauthClient, repo, nil)
	_, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "ok-token")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserEmailConflict, de.Code)
}

func TestUserService_UpdateEmail_Success_RevokesRefreshTokens(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", mock.Anything).Return(&idtoken.Payload{
		Claims: map[string]any{"email": "new@x.com"},
	}, nil)
	repo := &mocks.MockUserRepository{}
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	updated := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithEmail("new@x.com"))
	repo.On("Update", mock.Anything, mock.MatchedBy(func(users []domain.User) bool {
		return len(users) == 1 && users[0].Id == "u1" && users[0].Email == "new@x.com"
	})).Return([]domain.User{updated}, nil)

	tokenRepo := &mocks.MockUserTokenRepository{}
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)

	svc := newEmailSvc(oauthClient, repo, uow)
	email, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "ok-token")
	assert.NoError(t, err)
	assert.Equal(t, "new@x.com", email)
	tokenRepo.AssertCalled(t, "RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh)
}

func TestUserService_UpdateBatch_PolicyPreFetchFails(t *testing.T) {
	policy := &mocks.MockUserPolicy{}
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(errors.New("denied"))
	svc := NewUserService(UserServiceParams{
		UserRepo: &mocks.MockUserRepository{}, UoW: &mocks.MockUnitOfWork{}, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: policy,
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Update(ctxWithAuth(&auth), domain.User{Id: "u1"})
	assert.Error(t, err)
}

func TestUserService_UpdateBatch_NotFound(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	policy := &mocks.MockUserPolicy{}
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: policy,
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Update(ctxWithAuth(&auth), domain.User{Id: "missing"})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdateNotFound, de.Code)
}

func TestUserService_UpdateBatch_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	policy := &mocks.MockUserPolicy{}
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	n := "Renamed"

	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	updated := target
	updated.Name = &n
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{updated}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: policy,
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	out, err := svc.Update(ctxWithAuth(&auth), domain.User{Id: "u1", Name: &n})
	assert.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "Renamed", *out[0].Name)
}

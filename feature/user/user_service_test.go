package user

import (
	"context"
	"errors"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/testutil/fixtures"
	"CredChain_Golang/infrastructure/testutil/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
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
	repo.On("Update", mock.Anything, mock.Anything).Return(&updated, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})
	name := "Alice"
	got, err := svc.UpdateProfile(context.Background(), "u1", &name, nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "u1", got.Id)
}

func TestUserService_UpdateEmail(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	u := fixtures.NewDomainUser(fixtures.WithEmail("new@x.com"))
	repo.On("Update", mock.Anything, mock.Anything).Return(&u, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})
	got, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com")
	assert.NoError(t, err)
	assert.Equal(t, "new@x.com", got)
}

func TestUserService_UpdateRole_SignerBelowAdmin(t *testing.T) {
	svc := NewUserService(UserServiceParams{
		UserRepo: &mocks.MockUserRepository{}, UoW: &mocks.MockUnitOfWork{}, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: &mocks.MockUserPolicy{},
	})

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	_, _, err := svc.UpdateRole(ctxWithAuth(&authUser),
		domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSignerAdminRequiredForbidden, de.Code)
}

func TestUserService_Destroy_SignerBelowAdmin(t *testing.T) {
	svc := NewUserService(UserServiceParams{
		UserRepo: &mocks.MockUserRepository{}, UoW: &mocks.MockUnitOfWork{}, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: &mocks.MockUserPolicy{},
	})

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	_, err := svc.Destroy(ctxWithAuth(&authUser), "u1")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSignerAdminRequiredForbidden, de.Code)
}

func TestUserService_Destroy_SelfDeleteForbidden(t *testing.T) {
	svc := NewUserService(UserServiceParams{
		UserRepo: &mocks.MockUserRepository{}, UoW: &mocks.MockUnitOfWork{}, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: &mocks.MockUserPolicy{},
	})

	authUser := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Destroy(ctxWithAuth(&authUser), "self")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeAuthForbidden, de.Code)
}

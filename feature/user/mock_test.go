package user

import (
	"context"

	"CredChain_Golang/domain"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock type for domain.UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetUsers(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetUsersByIDs(ctx context.Context, ids []string) ([]domain.User, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockUserRepository) UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta *domain.JSONB) (*domain.User, error) {
	args := m.Called(ctx, id, name, number, phoneNumber, meta)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) UpdateEmail(ctx context.Context, id string, email string) (string, error) {
	args := m.Called(ctx, id, email)
	return args.String(0), args.Error(1)
}

func (m *MockUserRepository) BatchUpdateRole(ctx context.Context, updates []domain.UserRoleUpdate) error {
	args := m.Called(ctx, updates)
	return args.Error(0)
}

func (m *MockUserRepository) BatchCreate(ctx context.Context, users []domain.User) ([]domain.User, error) {
	args := m.Called(ctx, users)
	return args.Get(0).([]domain.User), args.Error(1)
}

// MockUserService is a mock type for UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUsers(ctx context.Context, newUsers []CreateUserRequest) ([]domain.User, error) {
	args := m.Called(ctx, newUsers)
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockUserService) GetUsers(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta *domain.JSONB) (*domain.User, error) {
	args := m.Called(ctx, id, name, number, phoneNumber, meta)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) UpdateEmail(ctx context.Context, id string, email string) (string, error) {
	args := m.Called(ctx, id, email)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) BatchUpdateRole(ctx context.Context, callerRole domain.Role, updates []domain.UserRoleUpdate) error {
	args := m.Called(ctx, callerRole, updates)
	return args.Error(0)
}

// MockCredentialRepository is a mock type for domain.CredentialRepository
type MockCredentialRepository struct {
	mock.Mock
}

func (m *MockCredentialRepository) GetCredentials(ctx context.Context) ([]domain.Credential, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Credential), args.Error(1)
}

func (m *MockCredentialRepository) GetCredentialByID(ctx context.Context, id string) (*domain.Credential, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.Credential), args.Error(1)
}

func (m *MockCredentialRepository) GetCredentialsByHolder(ctx context.Context, holderID string) ([]domain.Credential, error) {
	args := m.Called(ctx, holderID)
	return args.Get(0).([]domain.Credential), args.Error(1)
}

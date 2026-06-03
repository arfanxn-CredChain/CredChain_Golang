package mocks

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockCredentialExtractJobRepository struct {
	mock.Mock
}

func (m *MockCredentialExtractJobRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialExtractJob, int, error) {
	args := m.Called(ctx, query)
	if v := args.Get(0); v != nil {
		return v.([]domain.CredentialExtractJob), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

func (m *MockCredentialExtractJobRepository) FindPending(ctx context.Context, query *domainQuery.Query) (*domain.CredentialExtractJob, error) {
	args := m.Called(ctx, query)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialExtractJob), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialExtractJobRepository) FindPendingTx(ctx context.Context, tx *gorm.DB, query *domainQuery.Query) (*domain.CredentialExtractJob, error) {
	args := m.Called(ctx, tx, query)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialExtractJob), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialExtractJobRepository) Store(ctx context.Context, job *domain.CredentialExtractJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockCredentialExtractJobRepository) MarkRunning(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCredentialExtractJobRepository) MarkSucceeded(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCredentialExtractJobRepository) MarkFailed(ctx context.Context, id string, errMsg string, maxAttempts int) error {
	args := m.Called(ctx, id, errMsg, maxAttempts)
	return args.Error(0)
}

var _ domain.CredentialExtractJobRepository = (*MockCredentialExtractJobRepository)(nil)

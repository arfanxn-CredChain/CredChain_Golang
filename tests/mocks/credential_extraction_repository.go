package mocks

import (
	"context"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/mock"
)

type MockCredentialExtractionRepository struct {
	mock.Mock
}

func (m *MockCredentialExtractionRepository) Store(ctx context.Context, extraction domain.CredentialExtraction) error {
	return m.Called(ctx, extraction).Error(0)
}

func (m *MockCredentialExtractionRepository) FindByCredentialId(ctx context.Context, credentialID string) (*domain.CredentialExtraction, error) {
	args := m.Called(ctx, credentialID)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialExtraction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialExtractionRepository) FindRankedByIds(ctx context.Context, values []string, limit int) ([]domain.CredentialExtraction, error) {
	args := m.Called(ctx, values, limit)
	if v := args.Get(0); v != nil {
		return v.([]domain.CredentialExtraction), args.Error(1)
	}
	return nil, args.Error(1)
}

var _ domain.CredentialExtractionRepository = (*MockCredentialExtractionRepository)(nil)

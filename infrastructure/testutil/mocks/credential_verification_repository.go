package mocks

import (
	"context"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/mock"
)

type MockCredentialVerificationRepository struct {
	mock.Mock
}

func (m *MockCredentialVerificationRepository) FindByUploadedFileHash(ctx context.Context, hash string) (*domain.CredentialVerification, error) {
	args := m.Called(ctx, hash)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialVerification), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialVerificationRepository) Store(ctx context.Context, verification domain.CredentialVerification) error {
	return m.Called(ctx, verification).Error(0)
}

var _ domain.CredentialVerificationRepository = (*MockCredentialVerificationRepository)(nil)

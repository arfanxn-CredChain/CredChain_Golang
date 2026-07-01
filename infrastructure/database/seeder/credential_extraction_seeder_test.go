package seeder_test

import (
	"context"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/infrastructure/testutil/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCredentialExtractionSeeder_Name(t *testing.T) {
	s := seeder.NewCredentialExtractionSeeder(nil, nil)
	assert.Equal(t, "credential_extraction", s.Name())
}

func TestCredentialExtractionSeeder_Seed(t *testing.T) {
	userName := "Budi Santoso"
	issuerName := "Edy Susilo"

	user := &domain.User{
		Id:    "user_1",
		Name:  &userName,
		Email: "budi@example.com",
	}
	issuer := &domain.User{
		Id:    "issuer_1",
		Name:  &issuerName,
		Email: "eddy@example.com",
	}

	cred := domain.Credential{
		ID:           "cred_1",
		Name:         "Sarjana Komputer",
		Holder:       user,
		Issuer:       issuer,
		IssuerUserID: issuer.Id,
		HolderUserID: user.Id,
		IssuedAt:     time.Now(),
	}
	credentials := []domain.Credential{cred}

	mockCredRepo := new(mocks.MockCredentialRepository)
	mockExtractionRepo := new(mocks.MockCredentialExtractionRepository)

	mockCredRepo.On("Get", mock.Anything, mock.MatchedBy(func(q *domainQuery.Query) bool {
		return len(q.Includes) == 2 && q.Includes[0] == "holder" && q.Includes[1] == "issuer"
	})).Return(credentials, len(credentials), nil)

	mockExtractionRepo.On("Store", mock.Anything, mock.MatchedBy(func(e domain.CredentialExtraction) bool {
		return e.CredentialID == "cred_1" &&
			len(e.FileHash) == 64 &&
			e.Text != "" &&
			len(e.IDs) > 0 &&
			!e.CreatedAt.IsZero() &&
			!e.UpdatedAt.IsZero()
	})).Return(nil)

	s := seeder.NewCredentialExtractionSeeder(mockExtractionRepo, mockCredRepo)
	ctx := context.Background()
	err := s.Seed(ctx)

	require.NoError(t, err)
	mockExtractionRepo.AssertExpectations(t)
	mockCredRepo.AssertExpectations(t)
	assert.Equal(t, domain.ExtractStatusSucceeded, credentials[0].ExtractStatus)
	assert.NotNil(t, credentials[0].ExtractedAt)
}

func TestCredentialExtractionSeeder_SkipWhenAlreadyExtracted(t *testing.T) {
	cred := domain.Credential{
		ID:            "cred_2",
		Name:          "Sarjana Komputer",
		ExtractStatus: domain.ExtractStatusSucceeded,
	}
	credentials := []domain.Credential{cred}

	mockCredRepo := new(mocks.MockCredentialRepository)
	mockExtractionRepo := new(mocks.MockCredentialExtractionRepository)

	mockCredRepo.On("Get", mock.Anything, mock.Anything).Return(credentials, len(credentials), nil)

	s := seeder.NewCredentialExtractionSeeder(mockExtractionRepo, mockCredRepo)
	ctx := context.Background()
	err := s.Seed(ctx)

	require.NoError(t, err)
	mockExtractionRepo.AssertNotCalled(t, "Store", mock.Anything, mock.Anything)
}

func TestCredentialExtractionSeeder_ErrorWhenHolderNil(t *testing.T) {
	issuerName := "Edy Susilo"
	cred := domain.Credential{
		ID:           "cred_3",
		Name:         "Sarjana Komputer",
		Holder:       nil,
		Issuer:       &domain.User{Id: "issuer_1", Name: &issuerName},
		IssuerUserID: "issuer_1",
		HolderUserID: "user_1",
		IssuedAt:     time.Now(),
	}
	credentials := []domain.Credential{cred}

	mockCredRepo := new(mocks.MockCredentialRepository)
	mockExtractionRepo := new(mocks.MockCredentialExtractionRepository)

	mockCredRepo.On("Get", mock.Anything, mock.Anything).Return(credentials, len(credentials), nil)

	s := seeder.NewCredentialExtractionSeeder(mockExtractionRepo, mockCredRepo)
	ctx := context.Background()
	err := s.Seed(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "holder or issuer not preloaded")
}

func TestCredentialExtractionSeeder_ErrorWhenIssuerNil(t *testing.T) {
	userName := "Budi Santoso"
	cred := domain.Credential{
		ID:           "cred_4",
		Name:         "Sarjana Komputer",
		Holder:       &domain.User{Id: "user_1", Name: &userName},
		Issuer:       nil,
		IssuerUserID: "issuer_1",
		HolderUserID: "user_1",
		IssuedAt:     time.Now(),
	}
	credentials := []domain.Credential{cred}

	mockCredRepo := new(mocks.MockCredentialRepository)
	mockExtractionRepo := new(mocks.MockCredentialExtractionRepository)

	mockCredRepo.On("Get", mock.Anything, mock.Anything).Return(credentials, len(credentials), nil)

	s := seeder.NewCredentialExtractionSeeder(mockExtractionRepo, mockCredRepo)
	ctx := context.Background()
	err := s.Seed(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "holder or issuer not preloaded")
}

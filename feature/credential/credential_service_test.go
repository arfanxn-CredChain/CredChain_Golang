package credential

import (
	"context"
	"testing"
	"time"

	"CredChain_Golang/domain"
	pyai "CredChain_Golang/infrastructure/ai/pyai"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/testutil/fixtures"
	"CredChain_Golang/infrastructure/testutil/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type testCredentialMocks struct {
	credRepo *mocks.MockCredentialRepository
	verRepo  *mocks.MockCredentialVerificationRepository
	extRepo  *mocks.MockCredentialExtractionRepository
	aiClient *mocks.MockPythonAIClient
	regSvc   *mocks.MockRegistryService
}

func newTestCredentialService(m *testCredentialMocks) *credentialService {
	return &credentialService{
		repo:             m.credRepo,
		verificationRepo: m.verRepo,
		extractionRepo:   m.extRepo,
		aiClient:         m.aiClient,
		registryService:  m.regSvc,
		policy:           &credentialPolicy{},
		logger:           zap.NewNop(),
	}
}

func ctxWithAuth(u *domain.User) context.Context {
	return context.WithValue(context.Background(), httpContext.UserKey, u)
}

func TestVerify_CacheHit(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	credID := "01J0000000000000000000000A"
	cached := &domain.CredentialVerification{
		VerdictCode:         domain.CodeCredentialVerifyAuthentic,
		MatchedCredentialID: &credID,
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
	m.credRepo.On("Find", mock.Anything, credID, mock.Anything).Return(&domain.Credential{ID: credID}, nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyAuthentic, code)
	assert.NotNil(t, cred)
	assert.Equal(t, credID, cred.ID)
	assert.Nil(t, score)
	assert.Nil(t, percent)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
	m.aiClient.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
	m.extRepo.AssertNotCalled(t, "FindRankedByIds", mock.Anything, mock.Anything, mock.Anything)
	m.regSvc.AssertNotCalled(t, "FindCredentialByHash", mock.Anything, mock.Anything)
}

func TestVerify_ExactAuthentic(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-1", FileHash: "0xabc", RevokedAt: nil},
	}, nil)
	m.regSvc.On("FindCredentialByHash", mock.Anything, mock.Anything).Return("0xtokenid", true, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyAuthentic, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.Nil(t, score)
	assert.Nil(t, percent)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
}

func TestVerify_ExactRevoked(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	now := time.Now()
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-1", FileHash: "0xabc", RevokedAt: &now},
	}, nil)
	m.regSvc.On("FindCredentialByHash", mock.Anything, mock.Anything).Return("0xtokenid", true, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyRevoked, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.Nil(t, score)
	assert.Nil(t, percent)
}

func TestVerify_ExactIntegrityWarning(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-1", FileHash: "0xabc", RevokedAt: nil},
	}, nil)
	m.regSvc.On("FindCredentialByHash", mock.Anything, mock.Anything).Return("", false, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyIntegrityWarning, code)
	assert.NotNil(t, cred)
	assert.Nil(t, score)
	assert.Nil(t, percent)
}

func TestVerify_FuzzyNoIdentifiers(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyNoIdentifiers, code)
	assert.Nil(t, cred)
	assert.Nil(t, score)
	assert.Nil(t, percent)
}

func TestVerify_FuzzyNoMatch(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyNoMatch, code)
	assert.Nil(t, cred)
	assert.Nil(t, score)
	assert.Nil(t, percent)
}

func TestVerify_FuzzyTampered(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-1", IDs: []domain.ExtractedID{{Value: "12345"}}, Embedding: []float64{0.1, 0.2}},
	}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{
		Verdict: "tampered", SimilarityScore: 0.3, SimilarityPercent: "30%",
	}, nil)
	m.credRepo.On("Find", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.NotNil(t, score)
	assert.Equal(t, 0.3, *score)
	assert.NotNil(t, percent)
	assert.Equal(t, "30%", *percent)
}

func TestVerify_FuzzySuspicious(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-1", IDs: []domain.ExtractedID{{Value: "12345"}}, Embedding: []float64{0.1, 0.2}},
	}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{
		Verdict: "suspicious", SimilarityScore: 0.5, SimilarityPercent: "50%",
	}, nil)
	m.credRepo.On("Find", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifySuspicious, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.Equal(t, 0.5, *score)
	assert.Equal(t, "50%", *percent)
}

func TestVerify_FuzzyLowSimilarity(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-1", IDs: []domain.ExtractedID{{Value: "12345"}}, Embedding: []float64{0.1, 0.2}},
	}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{
		Verdict: "low_similarity", SimilarityScore: 0.4, SimilarityPercent: "40%",
	}, nil)
	m.credRepo.On("Find", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyLowSimilarity, code)
	assert.NotNil(t, cred)
	assert.Equal(t, 0.4, *score)
	assert.Equal(t, "40%", *percent)
}

func TestVerify_FuzzyNotSimilar(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-1", IDs: []domain.ExtractedID{{Value: "12345"}}, Embedding: []float64{0.1, 0.2}},
	}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{
		Verdict: "not_similar", SimilarityScore: 0.2, SimilarityPercent: "20%",
	}, nil)
	m.credRepo.On("Find", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyNotSimilar, code)
	assert.NotNil(t, cred)
	assert.Equal(t, 0.2, *score)
	assert.Equal(t, "20%", *percent)
}

func TestVerify_TieBreakNonRevokedPreferred(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "ID1"},
		{Type: "student_id", Value: "ID2"},
	}, nil)

	now := time.Now()
	earlier := now.Add(-24 * time.Hour)

	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-revoked", IDs: []domain.ExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{1.0, 0.0}},
		{CredentialID: "cred-live", IDs: []domain.ExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{2.0, 0.0}},
	}, nil)

	m.credRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-revoked", RevokedAt: &now, IssuedAt: earlier},
		{ID: "cred-live", RevokedAt: nil, IssuedAt: earlier},
	}, nil)

	var actualEmbedding []float64
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).
		Return(&pyai.VerifyResult{Verdict: "tampered", SimilarityScore: 0.3, SimilarityPercent: "30%"}, nil).
		Run(func(args mock.Arguments) {
			actualEmbedding = args.Get(2).([]float64)
		})

	m.credRepo.On("Find", mock.Anything, "cred-live", mock.Anything).Return(&domain.Credential{ID: "cred-live"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-live", cred.ID)
	assert.Equal(t, []float64{2.0, 0.0}, actualEmbedding)
}

func TestVerify_TieBreakNewestIssuedAt(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "ID1"},
		{Type: "student_id", Value: "ID2"},
	}, nil)

	now := time.Now()
	earlier := now.Add(-48 * time.Hour)
	newer := now.Add(-24 * time.Hour)

	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-old", IDs: []domain.ExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{1.0, 0.0}},
		{CredentialID: "cred-new", IDs: []domain.ExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{3.0, 0.0}},
	}, nil)

	m.credRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-old", RevokedAt: nil, IssuedAt: earlier},
		{ID: "cred-new", RevokedAt: nil, IssuedAt: newer},
	}, nil)

	var actualEmbedding []float64
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).
		Return(&pyai.VerifyResult{Verdict: "tampered", SimilarityScore: 0.3, SimilarityPercent: "30%"}, nil).
		Run(func(args mock.Arguments) {
			actualEmbedding = args.Get(2).([]float64)
		})

	m.credRepo.On("Find", mock.Anything, "cred-new", mock.Anything).Return(&domain.Credential{ID: "cred-new"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-new", cred.ID)
	assert.Equal(t, []float64{3.0, 0.0}, actualEmbedding)
}

package jobs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/ai/pyai"
	"CredChain_Golang/infrastructure/testutil/mocks"

	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWorkExtract_Success(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.pdf")
	assert.NoError(t, os.WriteFile(tmpFile, []byte("test-data"), 0644))

	credRepo := &mocks.MockCredentialRepository{}
	extRepo := &mocks.MockCredentialExtractionRepository{}
	aiClient := &mocks.MockPythonAIClient{}

	credRepo.On("Find", mock.Anything, "cred-id", mock.Anything).
		Return(&domain.Credential{FileHash: "0xabc"}, nil)
	aiClient.On("Extract", mock.Anything, mock.Anything).
		Return([]pyai.PythonExtractResult{
			{Text: "extracted-text", IDs: []pyai.ExtractedID{{Type: "id", Value: "123"}}, Embedding: []float64{0.1, 0.2}},
		}, nil)
	extRepo.On("Store", mock.Anything, mock.Anything).Return(nil)
	credRepo.On("Update", mock.Anything, mock.Anything).
		Return([]domain.Credential{{ID: "cred-id", ExtractStatus: domain.ExtractStatusSucceeded}}, nil)

	w := &CredentialExtractWorker{
		credRepo:       credRepo,
		extractionRepo: extRepo,
		aiClient:       aiClient,
		logger:         zap.NewNop(),
	}
	err := w.workExtract(context.Background(), CredentialExtractArgs{CredentialID: "cred-id", FileURI: tmpFile})
	assert.NoError(t, err)
	extRepo.AssertCalled(t, "Store", mock.Anything, mock.Anything)
	credRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestWorkExtract_FileNotFound(t *testing.T) {
	w := &CredentialExtractWorker{
		aiClient: &mocks.MockPythonAIClient{},
		logger:   zap.NewNop(),
	}
	err := w.workExtract(context.Background(), CredentialExtractArgs{CredentialID: "cred-id", FileURI: "/nonexistent/file.pdf"})
	assert.Error(t, err)
}

func TestWorkExtract_EmptyEmbedding(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.pdf")
	assert.NoError(t, os.WriteFile(tmpFile, []byte("test-data"), 0644))

	credRepo := &mocks.MockCredentialRepository{}
	extRepo := &mocks.MockCredentialExtractionRepository{}
	aiClient := &mocks.MockPythonAIClient{}

	aiClient.On("Extract", mock.Anything, mock.Anything).
		Return([]pyai.PythonExtractResult{{Text: "", IDs: nil, Embedding: nil}}, nil)

	w := &CredentialExtractWorker{
		credRepo:       credRepo,
		extractionRepo: extRepo,
		aiClient:       aiClient,
		logger:         zap.NewNop(),
	}
	err := w.workExtract(context.Background(), CredentialExtractArgs{CredentialID: "cred-id", FileURI: tmpFile})
	assert.Error(t, err)
	extRepo.AssertNotCalled(t, "Store", mock.Anything, mock.Anything)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestWorkExtract_StoreFails(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.pdf")
	assert.NoError(t, os.WriteFile(tmpFile, []byte("test-data"), 0644))

	credRepo := &mocks.MockCredentialRepository{}
	extRepo := &mocks.MockCredentialExtractionRepository{}
	aiClient := &mocks.MockPythonAIClient{}

	credRepo.On("Find", mock.Anything, "cred-id", mock.Anything).
		Return(&domain.Credential{FileHash: "0xabc"}, nil)
	aiClient.On("Extract", mock.Anything, mock.Anything).
		Return([]pyai.PythonExtractResult{
			{Text: "t", IDs: []pyai.ExtractedID{{Type: "a", Value: "b"}}, Embedding: []float64{0.5}},
		}, nil)
	extRepo.On("Store", mock.Anything, mock.Anything).Return(assert.AnError)

	w := &CredentialExtractWorker{
		credRepo:       credRepo,
		extractionRepo: extRepo,
		aiClient:       aiClient,
		logger:         zap.NewNop(),
	}
	err := w.workExtract(context.Background(), CredentialExtractArgs{CredentialID: "cred-id", FileURI: tmpFile})
	assert.Error(t, err)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestHandleError_TerminalFailure(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "cred-id"}}, nil)

	args, err := json.Marshal(CredentialExtractArgs{CredentialID: "cred-id", FileURI: "/tmp/x"})
	require.NoError(t, err)

	w := &CredentialExtractWorker{credRepo: credRepo, logger: zap.NewNop()}
	job := &rivertype.JobRow{
		Kind:        "credential_extract",
		Attempt:     5,
		MaxAttempts: 5,
		EncodedArgs: args,
	}
	w.HandleError(context.Background(), job, assert.AnError)
	credRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleError_NonTerminal(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	args, err := json.Marshal(CredentialExtractArgs{CredentialID: "cred-id"})
	require.NoError(t, err)
	w := &CredentialExtractWorker{credRepo: credRepo, logger: zap.NewNop()}
	job := &rivertype.JobRow{
		Kind:        "credential_extract",
		Attempt:     1,
		MaxAttempts: 5,
		EncodedArgs: args,
	}
	w.HandleError(context.Background(), job, assert.AnError)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestHandlePanic_Terminal(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "cred-id"}}, nil)
	args, err := json.Marshal(CredentialExtractArgs{CredentialID: "cred-id"})
	require.NoError(t, err)
	w := &CredentialExtractWorker{credRepo: credRepo, logger: zap.NewNop()}
	job := &rivertype.JobRow{
		Kind:        "credential_extract",
		Attempt:     5,
		MaxAttempts: 5,
		EncodedArgs: args,
	}
	w.HandlePanic(context.Background(), job, "ouch", "stacktrace")
	credRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

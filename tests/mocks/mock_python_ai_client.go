package mocks

import (
	"context"

	"CredChain_Golang/infrastructure/ai/pyai"

	"github.com/stretchr/testify/mock"
)

type MockPythonAIClient struct {
	mock.Mock
}

func (m *MockPythonAIClient) Extract(ctx context.Context, files ...pyai.ExtractFile) ([]pyai.PythonExtractResult, error) {
	args := m.Called(ctx, files)
	if v := args.Get(0); v != nil {
		return v.([]pyai.PythonExtractResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPythonAIClient) Verify(ctx context.Context, file pyai.ExtractFile, storedEmbedding []float64) (*pyai.VerifyResult, error) {
	args := m.Called(ctx, file, storedEmbedding)
	if v := args.Get(0); v != nil {
		return v.(*pyai.VerifyResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPythonAIClient) ExtractIDs(ctx context.Context, file pyai.ExtractFile) ([]pyai.ExtractedID, error) {
	args := m.Called(ctx, file)
	if v := args.Get(0); v != nil {
		return v.([]pyai.ExtractedID), args.Error(1)
	}
	return nil, args.Error(1)
}

var _ pyai.PythonAIClient = (*MockPythonAIClient)(nil)

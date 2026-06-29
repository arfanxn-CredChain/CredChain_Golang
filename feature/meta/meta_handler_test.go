package meta

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/response"
	"CredChain_Golang/infrastructure/testutil/gintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) Get(ctx context.Context) (*response.Meta, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*response.Meta), args.Error(1)
}

func TestHandlerGet_Success(t *testing.T) {
	svc := new(mockService)
	handler := NewMetaHandler(MetaHandlerParams{Svc: svc})

	ginCtx, rr := gintest.NewContext(t, gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)))

	svc.On("Get", mock.Anything).Return(&response.Meta{
		IssuingOrganizationName: "University of Indonesia",
		AuthorityContract:       "0xAAA",
		RegistryContract:        "0xBBB",
		ChainID:                 137,
		LastBlock:               42000000,
	}, nil)

	handler.Get(ginCtx)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, float64(domain.CodeMetaSuccess), body["code"])

	data := body["data"].(map[string]interface{})
	assert.Equal(t, "University of Indonesia", data["issuing_organization_name"])
	assert.Equal(t, "0xAAA", data["authority_contract"])
	assert.Equal(t, "0xBBB", data["registry_contract"])
	assert.Equal(t, float64(137), data["chain_id"])
	assert.Equal(t, float64(42000000), data["last_block"])
}

func TestHandlerGet_ServiceError(t *testing.T) {
	svc := new(mockService)
	handler := NewMetaHandler(MetaHandlerParams{Svc: svc})

	ginCtx, rr := gintest.NewContext(t, gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)))

	svc.On("Get", mock.Anything).Return(nil, domain.NewError(domain.CodeMetaInternal))

	handler.Get(ginCtx)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, float64(domain.CodeMetaInternal), body["code"])
}

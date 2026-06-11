package credential

import (
	"mime/multipart"
	"net/http"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/testutil/fixtures"
	"CredChain_Golang/infrastructure/testutil/gintest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestParseItemIndex(t *testing.T) {
	tests := []struct {
		key     string
		wantIdx int
		wantOK  bool
	}{
		{"items[0][holder_user_id]", 0, true},
		{"items[99][name]", 99, true},
		{"items[abc][x]", 0, false},
		{"not_items[0][x]", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseItemIndex(tt.key)
		assert.Equal(t, tt.wantOK, ok, "key=%s", tt.key)
		if tt.wantOK {
			assert.Equal(t, tt.wantIdx, got, "key=%s", tt.key)
		}
	}
}

func TestMapCredentialsToResponse(t *testing.T) {
	creds := []domain.Credential{
		{ID: "c1", Name: "n1"},
		{ID: "c2", Name: "n2"},
	}
	out := mapCredentialsToResponse(creds)
	assert.Len(t, out, 2)
	assert.Equal(t, "c1", out[0].ID)
	assert.Equal(t, "n2", out[1].Name)
}

func TestMapCredentialsToResponse_Empty(t *testing.T) {
	out := mapCredentialsToResponse([]domain.Credential{})
	assert.Len(t, out, 0)
}

func TestBuildIssueItems(t *testing.T) {
	form := &multipart.Form{
		Value: map[string][]string{
			"items[0][holder_user_id]": {"holder-1"},
			"items[0][name]":           {"Degree"},
			"items[1][holder_user_id]": {"holder-2"},
			"items[1][name]":           {"Diploma"},
		},
		File: map[string][]*multipart.FileHeader{},
	}
	items, err := buildIssueItems(form)
	assert.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "holder-1", items[0].HolderUserID)
	assert.Equal(t, "Degree", items[0].Name)
	assert.Equal(t, "holder-2", items[1].HolderUserID)
}

func TestHandler_Paginate_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("Paginate", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1", Name: "test"}}, 1, nil)
	h := &credentialHandler{credSvc: svc}
	h.Paginate(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_Paginate_ServiceError(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("Paginate", mock.Anything, mock.Anything).Return([]domain.Credential(nil), 0, assert.AnError)
	h := &credentialHandler{credSvc: svc}
	h.Paginate(c)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandler_Find_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/c1"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	svc := &mockCredentialService{}
	svc.On("Find", mock.Anything, "c1", mock.Anything).Return(&domain.Credential{ID: "c1"}, nil)
	h := &credentialHandler{credSvc: svc}
	h.Find(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_Find_NotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/missing"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: "missing"}}
	svc := &mockCredentialService{}
	svc.On("Find", mock.Anything, "missing", mock.Anything).Return(nil, domain.NewError(domain.CodeCredentialFetchNotFound))
	h := &credentialHandler{credSvc: svc}
	h.Find(c)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandler_SelfPaginate_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	svc := &mockCredentialService{}
	svc.On("SelfPaginate", mock.Anything, mock.Anything).Return([]domain.Credential{{ID: "c1", HolderUserID: user.Id}}, 1, nil)
	h := &credentialHandler{credSvc: svc}
	h.SelfPaginate(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_SelfFind_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithUser(&user),
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/c1"),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	svc := &mockCredentialService{}
	svc.On("SelfFind", mock.Anything, "c1", mock.Anything).Return(&domain.Credential{ID: "c1"}, nil)
	h := &credentialHandler{credSvc: svc}
	h.SelfFind(c)
	assert.Equal(t, http.StatusOK, rr.Code)
}

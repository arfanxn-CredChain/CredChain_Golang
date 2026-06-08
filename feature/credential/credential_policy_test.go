package credential

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/testutil/fixtures"

	"github.com/stretchr/testify/assert"
)

func ctxWithRole(role domain.Role) context.Context {
	user := fixtures.NewDomainUser(fixtures.WithRole(role))
	return context.WithValue(context.Background(), httpContext.UserKey, &user)
}

func TestCredentialPolicy_IssuePreFetch_Forbidden(t *testing.T) {
	p := &credentialPolicy{}
	err := p.IssuePreFetch(ctxWithRole(domain.RoleHolder), nil)
	assert.Error(t, err)
}

func TestCredentialPolicy_VerifyPreFetch_Forbidden(t *testing.T) {
	p := &credentialPolicy{}
	err := p.VerifyPreFetch(ctxWithRole(domain.RoleHolder))
	assert.Error(t, err)
}

func TestCredentialPolicy_ReExtractPreFetch_Forbidden(t *testing.T) {
	p := &credentialPolicy{}
	err := p.ReExtractPreFetch(ctxWithRole(domain.RoleHolder))
	assert.Error(t, err)
}

func TestCredentialPolicy_RevokePreFetch(t *testing.T) {
	p := &credentialPolicy{}
	t.Run("issuer ok", func(t *testing.T) {
		assert.NoError(t, p.RevokePreFetch(ctxWithRole(domain.RoleIssuer), []string{"id"}))
	})
	t.Run("holder forbidden", func(t *testing.T) {
		assert.Error(t, p.RevokePreFetch(ctxWithRole(domain.RoleHolder), []string{"id"}))
	})
}

func TestCredentialPolicy_RevokePostFetch(t *testing.T) {
	p := &credentialPolicy{}
	assert.NoError(t, p.RevokePostFetch(context.Background(), nil))
}

func TestNewCredentialPolicy(t *testing.T) {
	p := NewCredentialPolicy(CredentialPolicyParams{})
	assert.NotNil(t, p)
}

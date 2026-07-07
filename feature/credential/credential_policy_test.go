package credential

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCredentialPolicy_RevokePostFetch(t *testing.T) {
	p := &credentialPolicy{}
	assert.NoError(t, p.RevokePostFetch(context.Background(), nil))
}

func TestNewCredentialPolicy(t *testing.T) {
	p := NewCredentialPolicy(CredentialPolicyParams{})
	assert.NotNil(t, p)
}

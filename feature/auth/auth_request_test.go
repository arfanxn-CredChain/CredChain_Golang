package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthGoogleLoginRequest_Validate_Empty(t *testing.T) {
	r := AuthGoogleLoginRequest{}
	assert.Error(t, r.Validate())
}

func TestAuthGoogleLoginRequest_Validate_Valid(t *testing.T) {
	r := AuthGoogleLoginRequest{IdToken: "valid-token-string"}
	assert.NoError(t, r.Validate())
}

func TestAuthRefreshRequest_Validate_Empty(t *testing.T) {
	r := AuthRefreshRequest{}
	assert.Error(t, r.Validate())
}

func TestAuthRefreshRequest_Validate_Valid(t *testing.T) {
	r := AuthRefreshRequest{RefreshToken: "refresh-token-string"}
	assert.NoError(t, r.Validate())
}

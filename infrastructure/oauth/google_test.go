package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/idtoken"
)

func TestExtractEmailFromGoogleIdToken_Valid(t *testing.T) {
	p := &idtoken.Payload{
		Claims: map[string]any{"email": "alice@example.com"},
	}
	assert.Equal(t, "alice@example.com", ExtractEmailFromGoogleIdToken(p))
}

func TestExtractEmailFromGoogleIdToken_MissingClaim(t *testing.T) {
	p := &idtoken.Payload{Claims: map[string]any{}}
	assert.Equal(t, "", ExtractEmailFromGoogleIdToken(p))
}

func TestExtractEmailFromGoogleIdToken_NonStringClaim(t *testing.T) {
	p := &idtoken.Payload{Claims: map[string]any{"email": 12345}}
	assert.Equal(t, "", ExtractEmailFromGoogleIdToken(p))
}

func TestExtractEmailFromGoogleIdToken_EmptyString(t *testing.T) {
	p := &idtoken.Payload{Claims: map[string]any{"email": ""}}
	assert.Equal(t, "", ExtractEmailFromGoogleIdToken(p))
}

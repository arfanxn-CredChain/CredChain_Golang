package fixtures

import (
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/crypto"

	"github.com/oklog/ulid/v2"
)

type TokenOption func(*domain.UserToken)

func TokenWithID(id string) TokenOption                 { return func(t *domain.UserToken) { t.Id = id } }
func TokenWithUserID(userID string) TokenOption         { return func(t *domain.UserToken) { t.UserId = userID } }
func TokenWithType(tp domain.UserTokenType) TokenOption { return func(t *domain.UserToken) { t.Type = tp } }
func TokenWithRevoked(at time.Time) TokenOption         { return func(t *domain.UserToken) { t.RevokedAt = &at } }
func TokenWithExpiresAt(at time.Time) TokenOption       { return func(t *domain.UserToken) { t.ExpiresAt = &at } }
func TokenWithToken(s string) TokenOption               { return func(t *domain.UserToken) { t.Token = s } }

// NewDomainUserToken returns a refresh-type token expiring 168h from now.
func NewDomainUserToken(opts ...TokenOption) domain.UserToken {
	expires := time.Now().Add(168 * time.Hour)
	tok := domain.UserToken{
		Id:        ulid.Make().String(),
		UserId:    ulid.Make().String(),
		Type:      domain.UserTokenTypeRefresh,
		Token:     crypto.MustGenerateRandomHex32(),
		ExpiresAt: &expires,
		CreatedAt: time.Now(),
	}
	for _, opt := range opts {
		opt(&tok)
	}
	return tok
}

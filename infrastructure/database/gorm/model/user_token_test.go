package model

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestUserToken_RoundTrip_AllFields(t *testing.T) {
	now := time.Now()
	last := now.Add(-time.Hour)
	exp := now.Add(time.Hour)
	rev := now.Add(-time.Minute)
	upd := now.Add(time.Minute)

	d := domain.UserToken{
		Id: "t1", UserId: "u1", Type: domain.UserTokenTypeRefresh,
		Token: "abc", LastUsedAt: &last, ExpiresAt: &exp, RevokedAt: &rev,
		CreatedAt: now, UpdatedAt: &upd,
	}
	m := FromDomainUserToken(d)
	roundtrip := m.ToDomain()
	assert.Equal(t, d.Id, roundtrip.Id)
	assert.Equal(t, d.UserId, roundtrip.UserId)
	assert.Equal(t, d.Type, roundtrip.Type)
	assert.Equal(t, d.Token, roundtrip.Token)
	assert.Equal(t, d.LastUsedAt, roundtrip.LastUsedAt)
	assert.Equal(t, d.ExpiresAt, roundtrip.ExpiresAt)
	assert.Equal(t, d.RevokedAt, roundtrip.RevokedAt)
}

func TestUserToken_RoundTrip_NilOptionals(t *testing.T) {
	d := domain.UserToken{Id: "t2", UserId: "u2", Type: domain.UserTokenTypeRefresh, Token: "tok"}
	m2 := FromDomainUserToken(d)
	roundtrip := m2.ToDomain()
	assert.Nil(t, roundtrip.LastUsedAt)
	assert.Nil(t, roundtrip.ExpiresAt)
	assert.Nil(t, roundtrip.RevokedAt)
	assert.Nil(t, roundtrip.UpdatedAt)
}

func TestUserToken_TypeStringConversion(t *testing.T) {
	d := domain.UserToken{Type: domain.UserTokenTypeRefresh}
	m := FromDomainUserToken(d)
	assert.Equal(t, "refresh", m.Type)
	assert.Equal(t, domain.UserTokenTypeRefresh, m.ToDomain().Type)
}

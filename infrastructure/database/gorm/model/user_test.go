package model

import (
	"testing"
	"time"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestUser_RoundTrip_AllFields(t *testing.T) {
	name := "Alice"
	number := "12345"
	phone := "+62812"
	bd := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	upd := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	meta := map[string]any{"x": "y"}

	d := domain.User{
		Id: "u1", Name: &name, Number: &number, PhoneNumber: &phone,
		Email: "a@x.com", BirthDate: &bd, Meta: meta,
		Role: domain.RoleAdmin, WalletAddress: "0xaa",
		EncryptedWalletPrivateKey: "enc", CreatedAt: time.Now(), UpdatedAt: &upd,
	}
	m := FromDomainUser(d)
	roundtrip := m.ToDomain()

	assert.Equal(t, d.Id, roundtrip.Id)
	assert.Equal(t, &name, roundtrip.Name)
	assert.Equal(t, &number, roundtrip.Number)
	assert.Equal(t, &phone, roundtrip.PhoneNumber)
	assert.Equal(t, "a@x.com", roundtrip.Email)
	assert.Equal(t, &bd, roundtrip.BirthDate)
	assert.Equal(t, meta, roundtrip.Meta)
	assert.Equal(t, domain.RoleAdmin, roundtrip.Role)
	assert.Equal(t, "0xaa", roundtrip.WalletAddress)
	assert.Equal(t, "enc", roundtrip.EncryptedWalletPrivateKey)
	assert.Equal(t, &upd, roundtrip.UpdatedAt)
}

func TestUser_RoundTrip_NilOptionals(t *testing.T) {
	d := domain.User{
		Id: "u2", Email: "b@x.com",
		Role: domain.RoleHolder, WalletAddress: "0xbb",
		CreatedAt: time.Now(),
	}
	m2 := FromDomainUser(d)
	roundtrip := m2.ToDomain()
	assert.Nil(t, roundtrip.Name)
	assert.Nil(t, roundtrip.Number)
	assert.Nil(t, roundtrip.PhoneNumber)
	assert.Nil(t, roundtrip.BirthDate)
	assert.Nil(t, roundtrip.Meta)
	assert.Nil(t, roundtrip.UpdatedAt)
}

func TestUser_RoleStringConversion(t *testing.T) {
	d := domain.User{Role: domain.RoleSuperAdmin, Email: "s@x", WalletAddress: "0x", CreatedAt: time.Now()}
	m := FromDomainUser(d)
	assert.Equal(t, "super_admin", m.Role)
	back := m.ToDomain()
	assert.Equal(t, domain.RoleSuperAdmin, back.Role)
}

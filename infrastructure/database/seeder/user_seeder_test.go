package seeder_test

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/infrastructure/testutil/db"
	"CredChain_Golang/infrastructure/testutil/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestUserSeeder_Seeds15Users(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	userRepo := user.NewGormUserRepository(gormDB)
	ctx := context.Background()

	seedMnemonic := "test test test test test test test test test test test junk"
	encKey := string(fixtures.TestWalletEncryptionKey())
	s := seeder.NewUserSeeder(userRepo, seedMnemonic, encKey)

	err := s.Seed(ctx)
	assert.NoError(t, err)

	superAdmins, err := userRepo.FindByRole(ctx, domain.RoleSuperAdmin)
	assert.NoError(t, err)
	assert.Len(t, superAdmins, 1)
	assert.Equal(t, "arfan2173@gmail.com", superAdmins[0].Email)
	assert.Equal(t, "Muhammad Arfan", *superAdmins[0].Name)

	admins, err := userRepo.FindByRole(ctx, domain.RoleAdmin)
	assert.NoError(t, err)
	assert.Len(t, admins, 1)
	assert.Equal(t, "arfanforproject@gmail.com", admins[0].Email)

	issuers, err := userRepo.FindByRole(ctx, domain.RoleIssuer)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(issuers), 1)
	hasEdy := false
	for _, u := range issuers {
		if u.Email == "edysusilo17580@gmail.com" {
			hasEdy = true
			break
		}
	}
	assert.True(t, hasEdy, "Edy Susilo should be an issuer")

	holders, err := userRepo.FindByRole(ctx, domain.RoleHolder)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(holders), 2)
	assert.Equal(t, "liesbethsh19@gmail.com", holders[0].Email)
	assert.Equal(t, "annasorokin2173@gmail.com", holders[1].Email)

	total := len(superAdmins) + len(admins) + len(issuers) + len(holders)
	assert.Equal(t, 15, total)
}

func TestUserSeeder_Name(t *testing.T) {
	s := seeder.NewUserSeeder(nil, "", "")
	assert.Equal(t, "user", s.Name())
}

func TestUserSeeder_DeterministicRandomUsers(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)

	repo1 := user.NewGormUserRepository(gormDB1)
	repo2 := user.NewGormUserRepository(gormDB2)

	encKey := string(fixtures.TestWalletEncryptionKey())
	mnemonic := "test test test test test test test test test test test junk"

	ctx := context.Background()

	s1 := seeder.NewUserSeeder(repo1, mnemonic, encKey)
	err := s1.Seed(ctx)
	assert.NoError(t, err)

	s2 := seeder.NewUserSeeder(repo2, mnemonic, encKey)
	err = s2.Seed(ctx)
	assert.NoError(t, err)

	issuers1, _ := repo1.FindByRole(ctx, domain.RoleIssuer)
	issuers2, _ := repo2.FindByRole(ctx, domain.RoleIssuer)
	holders1, _ := repo1.FindByRole(ctx, domain.RoleHolder)
	holders2, _ := repo2.FindByRole(ctx, domain.RoleHolder)

	assert.Equal(t, len(issuers1), len(issuers2))
	assert.Equal(t, len(holders1), len(holders2))

	for i := range issuers1 {
		assert.Equal(t, issuers1[i].Email, issuers2[i].Email)
	}
	for i := range holders1 {
		assert.Equal(t, holders1[i].Email, holders2[i].Email)
	}
}

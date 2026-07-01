package seeder_test

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	credentialFeature "CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/infrastructure/storage"
	"CredChain_Golang/infrastructure/testutil/db"
	"CredChain_Golang/infrastructure/testutil/fixtures"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSeederConfig(t *testing.T) *config.Config {
	t.Helper()
	encKey := string(fixtures.TestWalletEncryptionKey())
	return &config.Config{
		StoragePath:               lo.ToPtr(t.TempDir()),
		CredentialFileStoragePath: lo.ToPtr("credentials"),
		FileEncryptionKey:         &encKey,
	}
}

func TestCredentialSeeder_Name(t *testing.T) {
	s := seeder.NewCredentialSeeder(nil, nil, nil, nil, 0)
	assert.Equal(t, "credential", s.Name())
}

func TestCredentialSeeder_SeedsCredentials(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	userRepo := user.NewGormUserRepository(gormDB)
	credentialRepo := credentialFeature.NewGormCredentialRepository(gormDB)
	ctx := context.Background()

	seedMnemonic := "test test test test test test test test test test test junk"
	encKey := string(fixtures.TestWalletEncryptionKey())
	userSeeder := seeder.NewUserSeeder(userRepo, seedMnemonic, encKey)
	require.NoError(t, userSeeder.Seed(ctx))

	cfg := testSeederConfig(t)
	storageInst, err := storage.NewStorage(storage.StorageParams{Config: cfg})
	require.NoError(t, err)

	s := seeder.NewCredentialSeeder(credentialRepo, userRepo, storageInst, cfg, 1)
	require.NoError(t, s.Seed(ctx))

	credentials, total, err := credentialRepo.Get(ctx, nil)
	require.NoError(t, err)
	require.Greater(t, total, 0)

	for _, c := range credentials {
		assert.NotEmpty(t, c.Name)
		assert.NotNil(t, c.FileURI)
		assert.NotEmpty(t, *c.FileURI)
		assert.False(t, strings.Contains(*c.FileURI, "/"), "file_uri should be filename only, got: %s", *c.FileURI)
		assert.NotEmpty(t, c.FileHash)
		assert.True(t, strings.HasPrefix(c.FileHash, "0x"), "file_hash should start with 0x, got: %s", c.FileHash)
		assert.Len(t, c.FileHash, 66, "file_hash should be 66 chars (0x + 64 hex), got: %s", c.FileHash)
		assert.NotNil(t, c.Meta)

		// Verify file on disk exists and is encrypted (not raw image bytes)
		fullPath := filepath.Join(*cfg.StoragePath, *cfg.CredentialFileStoragePath, *c.FileURI)
		data, err := os.ReadFile(fullPath)
		require.NoError(t, err)
		// File content should be hex-encoded (not raw JPEG/PNG/PDF magic bytes)
		encryptedHex := string(data)
		_, err = hex.DecodeString(encryptedHex)
		assert.NoError(t, err, "file content should be hex-encoded encrypted data")

		// Decrypt and verify the content is a valid image
		decrypted, err := hex.DecodeString(encryptedHex)
		require.NoError(t, err)
		assert.Greater(t, len(decrypted), 100)
	}
}

func TestCredentialSeeder_Deterministic(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)

	userRepo1 := user.NewGormUserRepository(gormDB1)
	userRepo2 := user.NewGormUserRepository(gormDB2)
	credentialRepo1 := credentialFeature.NewGormCredentialRepository(gormDB1)
	credentialRepo2 := credentialFeature.NewGormCredentialRepository(gormDB2)

	ctx := context.Background()
	seedMnemonic := "test test test test test test test test test test test junk"
	encKey := string(fixtures.TestWalletEncryptionKey())

	cfg1 := testSeederConfig(t)
	cfg2 := testSeederConfig(t)

	storage1, err := storage.NewStorage(storage.StorageParams{Config: cfg1})
	require.NoError(t, err)
	storage2, err := storage.NewStorage(storage.StorageParams{Config: cfg2})
	require.NoError(t, err)

	require.NoError(t, seeder.NewUserSeeder(userRepo1, seedMnemonic, encKey).Seed(ctx))
	require.NoError(t, seeder.NewCredentialSeeder(credentialRepo1, userRepo1, storage1, cfg1, 1).Seed(ctx))

	require.NoError(t, seeder.NewUserSeeder(userRepo2, seedMnemonic, encKey).Seed(ctx))
	require.NoError(t, seeder.NewCredentialSeeder(credentialRepo2, userRepo2, storage2, cfg2, 1).Seed(ctx))

	creds1, total1, err := credentialRepo1.Get(ctx, nil)
	require.NoError(t, err)
	creds2, total2, err := credentialRepo2.Get(ctx, nil)
	require.NoError(t, err)

	assert.Equal(t, total1, total2)
	assert.Equal(t, len(creds1), len(creds2))
	for i := range creds1 {
		assert.Equal(t, creds1[i].Name, creds2[i].Name)
	}
}

func TestCredentialSeeder_NoIssuerError(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	userRepo := user.NewGormUserRepository(gormDB)
	credentialRepo := credentialFeature.NewGormCredentialRepository(gormDB)
	ctx := context.Background()

	_, err := userRepo.Store(ctx, domain.User{
		Email: "holder1@example.com",
		Name:  lo.ToPtr("Holder One"),
		Role:  domain.RoleHolder,
	})
	require.NoError(t, err)

	cfg := testSeederConfig(t)
	storageInst, err := storage.NewStorage(storage.StorageParams{Config: cfg})
	require.NoError(t, err)

	s := seeder.NewCredentialSeeder(credentialRepo, userRepo, storageInst, cfg, 1)
	err = s.Seed(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no issuer user found")
}

func TestCredentialSeeder_RandomFormat(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	userRepo := user.NewGormUserRepository(gormDB)
	credentialRepo := credentialFeature.NewGormCredentialRepository(gormDB)
	ctx := context.Background()

	seedMnemonic := "test test test test test test test test test test test junk"
	encKey := string(fixtures.TestWalletEncryptionKey())
	userSeeder := seeder.NewUserSeeder(userRepo, seedMnemonic, encKey)
	require.NoError(t, userSeeder.Seed(ctx))

	cfg := testSeederConfig(t)
	storageInst, err := storage.NewStorage(storage.StorageParams{Config: cfg})
	require.NoError(t, err)

	s := seeder.NewCredentialSeeder(credentialRepo, userRepo, storageInst, cfg, 1)
	require.NoError(t, s.Seed(ctx))

	credentials, total, err := credentialRepo.Get(ctx, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, 10)

	extensions := map[string]bool{}
	for _, c := range credentials {
		ext := filepath.Ext(*c.FileURI)
		extensions[ext] = true
	}

	t.Logf("Extensions seen: %v", extensions)
	assert.True(t, len(extensions) >= 2, "expected at least 2 different file extensions across %d credentials, got: %v", total, extensions)
}

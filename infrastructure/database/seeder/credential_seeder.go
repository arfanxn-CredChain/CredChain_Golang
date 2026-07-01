package seeder

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"path/filepath"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	infraCrypto "CredChain_Golang/infrastructure/crypto"
	"CredChain_Golang/infrastructure/storage"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/oklog/ulid/v2"
)

type credFormat struct {
	ext      string
	renderer func(content diplomaContent) ([]byte, error)
}

var credentialFormats = []credFormat{
	{".jpg", SeedRenderDiplomaJPEG},
	{".jpeg", SeedRenderDiplomaJPEG},
	{".png", SeedRenderDiplomaPNG},
	{".pdf", func(content diplomaContent) ([]byte, error) {
		pngBytes, err := SeedRenderDiplomaPNG(content)
		if err != nil {
			return nil, err
		}
		return SeedRenderDiplomaPDF(content, pngBytes)
	}},
}

type CredentialSeeder struct {
	credentialRepo domain.CredentialRepository
	userRepo       domain.UserRepository
	localFS        *storage.Storage
	cfg            *config.Config
	count          int
}

func NewCredentialSeeder(credentialRepo domain.CredentialRepository, userRepo domain.UserRepository, fs *storage.Storage, cfg *config.Config, count int) *CredentialSeeder {
	return &CredentialSeeder{credentialRepo: credentialRepo, userRepo: userRepo, localFS: fs, cfg: cfg, count: count}
}

func (s *CredentialSeeder) Name() string { return "credential" }

func (s *CredentialSeeder) Seed(ctx context.Context) error {
	users, _, err := s.userRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("credential seeder: fetch users: %w", err)
	}

	issuer := s.findIssuer(users)
	if issuer == nil {
		return fmt.Errorf("credential seeder: no issuer user found")
	}

	var allCredentials []domain.Credential

	for _, user := range users {
		for i := 0; i < s.count; i++ {
			userSeed := hashToSeed(user.Id + ":" + fmt.Sprintf("%d", i))
			diplomaRng := rand.New(rand.NewSource(userSeed))

			content := SeedBuildDiplomaContent(user, *issuer, i, diplomaRng)

			cf := credentialFormats[diplomaRng.Intn(len(credentialFormats))]

			fileBytes, err := cf.renderer(content)
			if err != nil {
				return fmt.Errorf("credential seeder: render %s: %w", cf.ext, err)
			}

			encryptedHex, encErr := infraCrypto.Encrypt(fileBytes, []byte(*s.cfg.FileEncryptionKey))
			if encErr != nil {
				return fmt.Errorf("credential seeder: encrypt: %w", encErr)
			}

			filename := ulid.Make().String() + cf.ext
			filePath := filepath.Join(*s.cfg.CredentialFileStoragePath, filename)
			if _, err := s.localFS.SaveBytes([]byte(encryptedHex), filePath); err != nil {
				return fmt.Errorf("credential seeder: save file: %w", err)
			}

			fileHash := "0x" + hex.EncodeToString(ethCrypto.Keccak256(fileBytes))
			metaText := SeedBuildDiplomaText(content)

			cred := domain.Credential{
				Name:         content.CredentialName,
				FileURI:      &filename,
				FileHash:     fileHash,
				Meta:         map[string]any{"content": metaText},
				TokenID:      nil,
				IssuerUserID: issuer.Id,
				HolderUserID: user.Id,
				IssuedAt:     time.Now(),
			}

			allCredentials = append(allCredentials, cred)
		}
	}

	_, err = s.credentialRepo.Store(ctx, allCredentials...)
	if err != nil {
		return fmt.Errorf("credential seeder: store: %w", err)
	}

	return nil
}

func (s *CredentialSeeder) findIssuer(users []domain.User) *domain.User {
	for i := range users {
		if users[i].Role == domain.RoleIssuer {
			return &users[i]
		}
	}
	return nil
}

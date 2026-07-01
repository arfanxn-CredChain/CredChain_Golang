package seeder

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
)

type CredentialExtractionSeeder struct {
	extractionRepo domain.CredentialExtractionRepository
	credentialRepo domain.CredentialRepository
}

func NewCredentialExtractionSeeder(extractionRepo domain.CredentialExtractionRepository, credentialRepo domain.CredentialRepository) *CredentialExtractionSeeder {
	return &CredentialExtractionSeeder{extractionRepo: extractionRepo, credentialRepo: credentialRepo}
}

func (s *CredentialExtractionSeeder) Name() string { return "credential_extraction" }

func (s *CredentialExtractionSeeder) Seed(ctx context.Context) error {
	query := &domainQuery.Query{
		Includes: []string{"holder", "issuer"},
	}
	credentials, _, err := s.credentialRepo.Get(ctx, query)
	if err != nil {
		return fmt.Errorf("credential extraction seeder: fetch credentials: %w", err)
	}

	nameToIndex := make(map[string]int, len(seedCredentialNames))
	for i, n := range seedCredentialNames {
		nameToIndex[n.Name] = i
	}

	for i := range credentials {
		credential := &credentials[i]
		if credential.ExtractStatus != "" {
			continue
		}
		if credential.Holder == nil || credential.Issuer == nil {
			return fmt.Errorf("credential extraction seeder: credential %s: holder or issuer not preloaded", credential.ID)
		}

		credentialIndex, ok := nameToIndex[credential.Name]
		if !ok {
			credentialIndex = 0
		}

		credentialSeed := hashToSeed(credential.ID)
		credentialRng := rand.New(rand.NewSource(credentialSeed))

		content := SeedBuildDiplomaContent(*credential.Holder, *credential.Issuer, credentialIndex, credentialRng)
		text := SeedBuildDiplomaText(content)
		ids := SeedBuildDiplomaExtractionIDs(content)

		hash := sha256.Sum256([]byte(text))
		fileHash := fmt.Sprintf("%x", hash)

		now := time.Now()
		extraction := domain.CredentialExtraction{
			CredentialID: credential.ID,
			FileHash:     fileHash,
			Text:         text,
			IDs:          ids,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := s.extractionRepo.Store(ctx, extraction); err != nil {
			return fmt.Errorf("credential extraction seeder: store credential %s: %w", credential.ID, err)
		}

		credential.ExtractStatus = domain.ExtractStatusSucceeded
		credential.ExtractedAt = &now
	}

	return nil
}

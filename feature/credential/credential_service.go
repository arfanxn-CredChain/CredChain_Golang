package credential

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	pyai "CredChain_Golang/infrastructure/ai/pyai"
	"CredChain_Golang/infrastructure/chain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	infraJobs "CredChain_Golang/infrastructure/jobs"
	"CredChain_Golang/infrastructure/storage"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ── Interface ─────────────────────────────────────────────────────────────

// CredentialService is the business-logic layer for credential operations.
// It orchestrates the GORM repository, the on-chain CredentialRegistry
// (via chain.RegistryService), the Python AI service (via pyai.PythonAIClient),
// local file storage, and the asynchronous extract worker queue.
type CredentialService interface {
	Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error)
	Find(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error)
	Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, error)
	Revoke(ctx context.Context, ids ...string) ([]domain.Credential, error)
	Verify(ctx context.Context, credentialID string, file pyai.ExtractFile) (string, float64, string, domain.Credential, error)
}

// CredentialIssuance is the service-layer input for one credential issuance. File
// bytes are already in memory (the handler reads multipart upload bytes
// before calling the service).
type CredentialIssuance struct {
	HolderUserID string
	Name         string
	Meta         map[string]any
	Filename     string
	MIMEType     string
	FileBytes    []byte
}

// ── Implementation struct & constructor ───────────────────────────────────

type credentialService struct {
	repo            domain.CredentialRepository
	uow             domain.UnitOfWork
	cfg             *config.Config
	registryService chain.RegistryService
	aiClient        pyai.PythonAIClient
	storage         *storage.Storage
	policy          CredentialPolicy
	userRepo        domain.UserRepository
	logger          *zap.Logger
}

type CredentialServiceParams struct {
	fx.In
	Repo            domain.CredentialRepository
	UoW             domain.UnitOfWork
	Config          *config.Config
	RegistryService chain.RegistryService
	AIClient        pyai.PythonAIClient
	Storage         *storage.Storage
	Policy          CredentialPolicy
	UserRepo        domain.UserRepository
	Logger          *zap.Logger
}

// NewCredentialService is the exported factory for FX injection.
func NewCredentialService(p CredentialServiceParams) CredentialService {
	return &credentialService{
		repo:            p.Repo,
		uow:             p.UoW,
		cfg:             p.Config,
		registryService: p.RegistryService,
		aiClient:        p.AIClient,
		storage:         p.Storage,
		policy:          p.Policy,
		userRepo:        p.UserRepo,
		logger:          p.Logger,
	}
}

// ── Paginate ──────────────────────────────────────────────────────────────

// Paginate returns a paginated list of credentials, optionally including
// holder/issuer/revoker user expansions via query.Includes.
func (s *credentialService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	return s.repo.Get(ctx, query)
}

// ── Find ──────────────────────────────────────────────────────────────────

// Find retrieves a single credential by ID with optional user expansions
// from query.Includes.
func (s *credentialService) Find(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	c, err := s.repo.Find(ctx, id, query)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewError(domain.CodeCredentialFetchNotFound,
				domain.WithMetadata("credential_id", id))
		}
		return nil, err
	}
	return c, nil
}

// ── Issue ─────────────────────────────────────────────────────────────────

// Issue performs the synchronous batch credential issuance flow.
//
// Architecture (Option A — sync chain, async embeddings):
//  1. IssuePreFetch policy (signer is Issuer+)
//  2. Compute keccak256(file_bytes) → file_hash for each item
//  3. Persist files to local storage → file_uri
//  4. Validate holders exist
//  5. Validate no duplicate file_hash among active credentials
//  6. IssuePostFetch policy
//  7. INSERT credential rows (extract_status=pending) inside UoW
//  8. Sync to chain via RegistryService.IssueCredentials → token IDs
//  9. UPDATE credentials SET token_id = ...
//
// 10. Enqueue one credential_extract_jobs row per credential in the same tx
func (s *credentialService) Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, error) {
	if len(items) == 0 {
		return []domain.Credential{}, nil
	}

	authUser := httpContext.MustGetUser(ctx)

	domainItems := make([]domain.Credential, len(items))
	hashes := make([]string, len(items))
	fileURIs := make([]string, len(items))
	holderIDs := make([]string, len(items))
	for i, it := range items {
		domainItems[i] = domain.Credential{
			ID:            ulid.Make().String(),
			HolderUserID:  it.HolderUserID,
			IssuerUserID:  authUser.Id,
			Name:          it.Name,
			Meta:          it.Meta,
			ExtractStatus: domain.ExtractStatusPending,
		}
		hashes[i] = "0x" + hex.EncodeToString(ethCrypto.Keccak256(it.FileBytes))
		domainItems[i].FileHash = hashes[i]
		holderIDs[i] = it.HolderUserID

		ext := strings.ToLower(filepath.Ext(it.Filename))
		if ext == "" {
			ext = ".bin"
		}
		path, err := s.storage.SaveBytes(it.FileBytes, ext)
		if err != nil {
			s.cleanupOrphanFiles(fileURIs[:i])
			return nil, domain.NewError(domain.CodeCredentialIssueStorageFailed,
				domain.WithError(err))
		}
		fileURIs[i] = path
		domainItems[i].FileURI = &path
	}

	if err := s.policy.IssuePreFetch(ctx, domainItems); err != nil {
		s.cleanupOrphanFiles(fileURIs)
		return nil, err
	}

	holders, err := s.userRepo.FindByIds(ctx, holderIDs...)
	if err != nil {
		s.cleanupOrphanFiles(fileURIs)
		return nil, err
	}
	holderIds := lo.Map(holders, func(h domain.User, _ int) string { return h.Id })
	missing, _ := lo.Difference(holderIDs, holderIds)
	if len(missing) > 0 {
		s.cleanupOrphanFiles(fileURIs)
		return nil, domain.NewError(domain.CodeCredentialIssueHolderNotFound,
			domain.WithMetadata("holder_user_ids", missing))
	}

	if err := s.policy.IssuePostFetch(ctx, domainItems, holders); err != nil {
		s.cleanupOrphanFiles(fileURIs)
		return nil, err
	}

	existing, err := s.repo.FindByFileHashes(ctx, hashes...)
	if err != nil {
		s.cleanupOrphanFiles(fileURIs)
		return nil, err
	}
	dupHashes := []string{}
	for _, e := range existing {
		if e.RevokedAt == nil {
			dupHashes = append(dupHashes, e.FileHash)
		}
	}
	if len(dupHashes) > 0 {
		s.cleanupOrphanFiles(fileURIs)
		return nil, domain.NewError(domain.CodeCredentialIssueDuplicateFileHash,
			domain.WithMetadata("file_hashes", dupHashes))
	}

	holderAddr := lo.SliceToMap(holders, func(h domain.User) (string, string) { return h.Id, h.WalletAddress })

	signerWallet := domain.WalletFromUser(*authUser)

	var stored []domain.Credential
	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		s2, err := uow.Credential().Store(ctx, domainItems...)
		if err != nil {
			return err
		}
		stored = s2

		issuances := make([]chain.CredentialIssuance, len(stored))
		for i, c := range stored {
			issuances[i] = chain.CredentialIssuance{
				HolderAddress: holderAddr[c.HolderUserID],
				Hash:          c.FileHash,
				URI:           lo.FromPtrOr(c.FileURI, ""),
			}
		}

		tokenIds, err := s.syncBlockchainIssue(ctx, signerWallet, issuances)
		if err != nil {
			return err
		}

		updates := make([]domain.Credential, len(stored))
		for i, c := range stored {
			tokIDStr := tokenIds[i].String()
			updates[i] = domain.Credential{ID: c.ID, TokenID: &tokIDStr}
			stored[i].TokenID = &tokIDStr
		}
		if _, err := uow.Credential().Update(ctx, updates...); err != nil {
			return err
		}

		for _, c := range stored {
			if c.FileURI == nil {
				continue
			}
			if err := infraJobs.EnqueueExtractJob(ctx, uow.CredentialExtractJob(), c.ID, *c.FileURI); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.cleanupOrphanFiles(fileURIs)
		return nil, err
	}

	return stored, nil
}

// ── Revoke ────────────────────────────────────────────────────────────────

// Revoke batch-revokes credentials by ID. Sets revoked_at, revoker_user_id
// in the database and syncs the revocation on-chain via the CredentialRegistry.
// Uses Update (CASE-based) — there is no separate Revoke method on the repository.
func (s *credentialService) Revoke(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	if err := s.policy.RevokePreFetch(ctx, ids); err != nil {
		return nil, err
	}

	authUser := httpContext.MustGetUser(ctx)
	now := time.Now()
	revokerID := authUser.Id

	var revoked []domain.Credential
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.Credential().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}

		targetIds := lo.Map(targets, func(c domain.Credential, _ int) string { return c.ID })
		missing, _ := lo.Difference(ids, targetIds)
		if len(missing) > 0 {
			return domain.NewError(domain.CodeCredentialRevokeNotFound,
				domain.WithMetadata("credential_ids", missing))
		}

		alreadyRevoked := []string{}
		for _, t := range targets {
			if t.RevokedAt != nil {
				alreadyRevoked = append(alreadyRevoked, t.ID)
			}
		}
		if len(alreadyRevoked) > 0 {
			return domain.NewError(domain.CodeCredentialRevokeAlreadyRevoked,
				domain.WithMetadata("credential_ids", alreadyRevoked))
		}

		if err := s.policy.RevokePostFetch(ctx, targets); err != nil {
			return err
		}

		updates := make([]domain.Credential, len(targets))
		tokenIds := make([]string, 0, len(targets))
		for i, t := range targets {
			updates[i] = domain.Credential{
				ID:            t.ID,
				RevokedAt:     &now,
				RevokerUserID: &revokerID,
			}
			if t.TokenID != nil {
				tokenIds = append(tokenIds, *t.TokenID)
			}
		}

		revoked, err = uow.Credential().Update(ctx, updates...)
		if err != nil {
			return err
		}

		if len(tokenIds) > 0 {
			signerWallet := domain.WalletFromUser(*authUser)
			if err := s.syncBlockchainRevoke(ctx, signerWallet, tokenIds); err != nil {
				return err
			}
		}
		return nil
	})
	return revoked, err
}

// ── Verify ────────────────────────────────────────────────────────────────

// Verify calls Python /verify with the uploaded file and the credential's
// stored embeddings. It gates on the credential's extract_status:
// pending → 409, failed → 422, succeeded → proceed.
func (s *credentialService) Verify(ctx context.Context, credentialID string, file pyai.ExtractFile) (string, float64, string, domain.Credential, error) {
	if err := s.policy.VerifyPreFetch(ctx); err != nil {
		return "", 0, "", domain.Credential{}, err
	}

	query := &domainQuery.Query{}
	cred, err := s.repo.Find(ctx, credentialID, query)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, "", domain.Credential{}, domain.NewError(domain.CodeCredentialVerifyCredentialNotFound,
				domain.WithMetadata("credential_id", credentialID))
		}
		return "", 0, "", domain.Credential{}, err
	}

	switch cred.ExtractStatus {
	case domain.ExtractStatusPending:
		return "", 0, "", domain.Credential{}, domain.NewError(domain.CodeCredentialVerifyExtractNotReady)
	case domain.ExtractStatusFailed:
		return "", 0, "", domain.Credential{}, domain.NewError(domain.CodeCredentialVerifyExtractFailed,
			domain.WithMetadata("extract_error", lo.FromPtrOr(cred.ExtractError, "")))
	}

	// TODO(Task 4.3): Verify is rewritten as a cache→exact→fuzzy pipeline; embedding now comes from MongoDB extraction.
	result, err := s.aiClient.Verify(ctx, file, nil)
	if err != nil {
		return "", 0, "", domain.Credential{}, domain.NewError(domain.CodeCredentialVerifyAiServiceFailed,
			domain.WithError(err))
	}

	return result.Verdict, result.SimilarityScore, result.SimilarityPercent, *cred, nil
}

// ── Blockchain sync helpers ───────────────────────────────────────────────

// syncBlockchainIssue calls RegistryService.IssueCredentials and translates
// raw chain errors into a domain code so the UoW transaction rolls back.
func (s *credentialService) syncBlockchainIssue(ctx context.Context, signer domain.Wallet, issuances []chain.CredentialIssuance) ([]*big.Int, error) {
	tokenIds, err := s.registryService.IssueCredentials(ctx, signer, issuances...)
	if err != nil {
		return nil, domain.NewError(domain.CodeCredentialIssueBlockchainSyncFailed,
			domain.WithError(err))
	}
	return tokenIds, nil
}

// syncBlockchainRevoke calls RegistryService.RevokeCredentials with the
// given token ID strings (decimal) and translates raw chain errors so the
// UoW transaction rolls back the DB revoke.
func (s *credentialService) syncBlockchainRevoke(ctx context.Context, signer domain.Wallet, tokenIDStrings []string) error {
	if len(tokenIDStrings) == 0 {
		return nil
	}
	tokenIDs := make([]*big.Int, 0, len(tokenIDStrings))
	for _, idStr := range tokenIDStrings {
		bi, ok := new(big.Int).SetString(idStr, 10)
		if !ok {
			return domain.NewError(domain.CodeCredentialRevokeBlockchainSyncFailed,
				domain.WithError(fmt.Errorf("invalid token id: %s", idStr)))
		}
		tokenIDs = append(tokenIDs, bi)
	}
	if err := s.registryService.RevokeCredentials(ctx, signer, tokenIDs...); err != nil {
		return domain.NewError(domain.CodeCredentialRevokeBlockchainSyncFailed,
			domain.WithError(err))
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────

// cleanupOrphanFiles deletes files that were persisted to storage but whose
// credential rows were never committed (e.g. because of a validation error
// or a chain failure rollback). Best-effort — log-and-continue.
func (s *credentialService) cleanupOrphanFiles(paths []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := os.Remove(p); err != nil {
			s.logger.Warn("failed to clean up orphan file",
				zap.String("path", p),
				zap.Error(err))
		}
	}
}

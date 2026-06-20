package credential

import (
	"context"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"

	"go.uber.org/fx"
)

// CredentialPolicy holds the authorization rules for credential operations.
// Following the user feature pattern: pre-fetch (no DB) + post-fetch (target rows in hand).
type CredentialPolicy interface {
	// IssuePreFetch checks signer rank only (Issuer+).
	IssuePreFetch(ctx context.Context, items []domain.Credential) error
	// IssuePostFetch runs after we know which holders/file-hashes already exist.
	IssuePostFetch(ctx context.Context, items []domain.Credential, holders []domain.User) error
	// RevokePreFetch checks signer rank only (Issuer+).
	RevokePreFetch(ctx context.Context, ids []string) error
	// RevokePostFetch runs after target credentials are loaded.
	RevokePostFetch(ctx context.Context, targets []domain.Credential) error
	// VerifyPreFetch checks signer rank only (Issuer+).
	VerifyPreFetch(ctx context.Context) error
	// ReExtractPreFetch checks signer rank only (Issuer+).
	ReExtractPreFetch(ctx context.Context) error
	// DownloadFilePreFetch checks holder owns credential OR signer is Issuer+.
	DownloadFilePreFetch(ctx context.Context, target domain.Credential) error
}

type credentialPolicy struct{}

type CredentialPolicyParams struct{ fx.In }

func NewCredentialPolicy(p CredentialPolicyParams) CredentialPolicy {
	return &credentialPolicy{}
}

func (p *credentialPolicy) IssuePreFetch(ctx context.Context, items []domain.Credential) error {
	if !signerIsIssuerOrAbove(ctx) {
		return domain.NewError(domain.CodeAuthForbidden)
	}
	return nil
}

func (p *credentialPolicy) IssuePostFetch(ctx context.Context, items []domain.Credential, holders []domain.User) error {
	// Hook for future per-target rules (e.g. holder must be Holder role).
	// Currently no additional checks beyond what the service does (existence,
	// duplicate hash) — left here so service stays free of role logic.
	return nil
}

func (p *credentialPolicy) RevokePreFetch(ctx context.Context, ids []string) error {
	if !signerIsIssuerOrAbove(ctx) {
		return domain.NewError(domain.CodeAuthForbidden)
	}
	return nil
}

func (p *credentialPolicy) RevokePostFetch(ctx context.Context, targets []domain.Credential) error {
	return nil
}

func (p *credentialPolicy) VerifyPreFetch(ctx context.Context) error {
	return nil // public endpoint — external verifiers (HR, employers) need no auth
}

func (p *credentialPolicy) ReExtractPreFetch(ctx context.Context) error {
	if !signerIsIssuerOrAbove(ctx) {
		return domain.NewError(domain.CodeAuthForbidden)
	}
	return nil
}

func (p *credentialPolicy) DownloadFilePreFetch(ctx context.Context, target domain.Credential) error {
	user := httpContext.MustGetUser(ctx)
	if target.HolderUserID == user.Id {
		return nil
	}
	if user.Role.Rank() >= domain.RoleIssuer.Rank() {
		return nil
	}
	return domain.NewError(domain.CodeCredentialFileDownloadForbidden)
}

func signerIsIssuerOrAbove(ctx context.Context) bool {
	user := httpContext.MustGetUser(ctx)
	return user.Role.Rank() >= domain.RoleIssuer.Rank()
}

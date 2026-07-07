package credential

import (
	"context"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"

	"go.uber.org/fx"
)

// CredentialPolicy holds the authorization rules for credential operations.
// Route-level role gates (IssuerRoleMiddleware on issue/revoke/reextract,
// AuthMiddleware on download) handle rank enforcement. Keep post-fetch methods
// as hooks for future per-target rules. Verify is public (no auth middleware).
type CredentialPolicy interface {
	IssuePostFetch(ctx context.Context, items []domain.Credential, holders []domain.User) error
	RevokePostFetch(ctx context.Context, targets []domain.Credential) error
	DownloadFilePreFetch(ctx context.Context, target domain.Credential) error
}

type credentialPolicy struct{}

type CredentialPolicyParams struct{ fx.In }

func NewCredentialPolicy(p CredentialPolicyParams) CredentialPolicy {
	return &credentialPolicy{}
}

func (p *credentialPolicy) IssuePostFetch(ctx context.Context, items []domain.Credential, holders []domain.User) error {
	return nil
}

func (p *credentialPolicy) RevokePostFetch(ctx context.Context, targets []domain.Credential) error {
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

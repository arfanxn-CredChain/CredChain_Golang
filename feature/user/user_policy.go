package user

import (
	"context"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain"
	httpContext "CredChain_Golang/infrastructure/http/context"

	"go.uber.org/fx"
)

type UserPolicy interface {
	Store(ctx context.Context, users ...domain.User) error
	UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) error
	Delete(ctx context.Context, ids ...string) error
}

type userPolicy struct {
	roleService chain.RoleService
}

type UserPolicyParams struct {
	fx.In
	RoleService chain.RoleService
}

func NewUserPolicy(p UserPolicyParams) UserPolicy {
	return &userPolicy{roleService: p.RoleService}
}

func (p *userPolicy) Store(ctx context.Context, users ...domain.User) error {
	authUser := httpContext.MustGetUser(ctx)
	if err := p.roleService.Verify(ctx, authUser.WalletAddress, domain.RoleAdmin); err != nil {
		return err
	}
	for _, user := range users {
		if user.Role == domain.RoleSuperAdmin {
			return domain.NewError(domain.CodeUserStoreSuperAdminForbidden)
		}
		if authUser.Role == domain.RoleAdmin && user.Role.Rank() >= domain.RoleAdmin.Rank() {
			return domain.NewError(domain.CodeUserStoreAdminCreateAdminForbidden)
		}
	}
	return nil
}

func (p *userPolicy) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) error {
	authUser := httpContext.MustGetUser(ctx)
	if err := p.roleService.Verify(ctx, authUser.WalletAddress, domain.RoleAdmin); err != nil {
		return err
	}
	if authUser.Role == domain.RoleAdmin {
		for _, update := range updates {
			_ = update
		}
	}
	return nil
}

func (p *userPolicy) Delete(ctx context.Context, ids ...string) error {
	authUser := httpContext.MustGetUser(ctx)
	if err := p.roleService.Verify(ctx, authUser.WalletAddress, domain.RoleAdmin); err != nil {
		return err
	}
	for _, id := range ids {
		if id == authUser.Id {
			return domain.NewError(domain.CodeAuthForbidden)
		}
	}
	return nil
}

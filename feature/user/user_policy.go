package user

import (
	"context"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
)

type UserPolicy interface {
	Store(ctx context.Context, users ...domain.User) error
	UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) error
	Delete(ctx context.Context, ids ...string) error
}

type userPolicy struct{}

func NewUserPolicy() UserPolicy {
	return &userPolicy{}
}

func (p *userPolicy) Store(ctx context.Context, users ...domain.User) error {
	authUser := httpContext.MustGetUser(ctx)
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
	_ = authUser
	_ = updates
	return nil
}

func (p *userPolicy) Delete(ctx context.Context, ids ...string) error {
	authUser := httpContext.MustGetUser(ctx)
	for _, id := range ids {
		if id == authUser.Id {
			return domain.NewError(domain.CodeAuthForbidden)
		}
	}
	return nil
}

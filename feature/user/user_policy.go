package user

import (
	"context"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
)

type UserPolicy interface {
	Store(ctx context.Context, users ...domain.User) error
	UpdatePreFetch(ctx context.Context, users ...domain.User) error
	UpdatePostFetch(ctx context.Context, targets []domain.User, updates []domain.User) error
	UpdateRolePreFetch(ctx context.Context, updates ...domain.UserRoleUpdate) error
	UpdateRolePostFetch(ctx context.Context, targets []domain.User, updates ...domain.UserRoleUpdate) error
	DeletePreFetch(ctx context.Context, ids ...string) error
	DeletePostFetch(ctx context.Context, targets []domain.User) error
}

type userPolicy struct{}

func NewUserPolicy() UserPolicy { return &userPolicy{} }

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

func (p *userPolicy) UpdatePreFetch(ctx context.Context, users ...domain.User) error {
	authUser := httpContext.MustGetUser(ctx)
	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden)
	}
	for _, u := range users {
		if u.Id == authUser.Id {
			return domain.NewError(domain.CodeUserUpdateSelfForbidden, domain.WithMetadata("user_id", authUser.Id))
		}
	}
	return nil
}

func (p *userPolicy) UpdatePostFetch(ctx context.Context, targets []domain.User, updates []domain.User) error {
	authUser := httpContext.MustGetUser(ctx)
	targetMap := make(map[string]domain.User, len(targets))
	for _, t := range targets {
		targetMap[t.Id] = t
	}
	for _, t := range targets {
		if t.DeletedAt != nil {
			return domain.NewError(domain.CodeUserUpdateTrashedForbidden,
				domain.WithMetadata("user_id", t.Id))
		}
		if t.Role == domain.RoleSuperAdmin {
			return domain.NewError(domain.CodeUserUpdateSuperAdminForbidden, domain.WithMetadata("user_id", t.Id))
		}
		if authUser.Role == domain.RoleAdmin && t.Role.Rank() >= domain.RoleAdmin.Rank() {
			return domain.NewError(domain.CodeUserUpdatePeerAdminForbidden,
				domain.WithMetadata("auth_user_id", authUser.Id),
				domain.WithMetadata("target_user_id", t.Id))
		}
	}
	for _, u := range updates {
		if u.Role == "" {
			continue
		}
		if u.Role == domain.RoleSuperAdmin {
			return domain.NewError(domain.CodeUserRoleSuperAdminBatchForbidden,
				domain.WithMetadata("user_id", u.Id),
				domain.WithMetadata("attempted_role", "super_admin"))
		}
		if authUser.Role == domain.RoleAdmin && u.Role.Rank() >= domain.RoleAdmin.Rank() {
			return domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden,
				domain.WithMetadata("auth_user_id", authUser.Id),
				domain.WithMetadata("attempted_role", u.Role.String()))
		}
	}
	return nil
}

func (p *userPolicy) UpdateRolePreFetch(ctx context.Context, updates ...domain.UserRoleUpdate) error {
	authUser := httpContext.MustGetUser(ctx)
	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden)
	}
	for _, u := range updates {
		if u.Role == domain.RoleSuperAdmin {
			return domain.NewError(domain.CodeUserRoleSuperAdminBatchForbidden,
				domain.WithMetadata("user_id", u.UserID),
				domain.WithMetadata("attempted_role", "super_admin"))
		}
		if u.UserID == authUser.Id {
			return domain.NewError(domain.CodeUserRoleSelfTargetForbidden,
				domain.WithMetadata("user_id", authUser.Id))
		}
	}
	return nil
}

func (p *userPolicy) UpdateRolePostFetch(ctx context.Context, targets []domain.User, updates ...domain.UserRoleUpdate) error {
	authUser := httpContext.MustGetUser(ctx)
	targetMap := make(map[string]domain.User, len(targets))
	for _, t := range targets {
		targetMap[t.Id] = t
	}
	for _, u := range updates {
		target, ok := targetMap[u.UserID]
		if !ok {
			return domain.NewError(domain.CodeUserFetchNotFound, domain.WithMetadata("user_id", u.UserID))
		}
		if target.DeletedAt != nil {
			return domain.NewError(domain.CodeUserRoleTrashedForbidden,
				domain.WithMetadata("user_id", u.UserID))
		}
		if target.Role == u.Role {
			return domain.NewError(domain.CodeUserRoleSameRoleUpdateForbidden,
				domain.WithMetadata("user_id", u.UserID),
				domain.WithMetadata("current_role", target.Role.String()))
		}
		if authUser.Role == domain.RoleAdmin {
			if target.Role.Rank() >= domain.RoleAdmin.Rank() {
				return domain.NewError(domain.CodeUserRoleAdminUpdatePeerForbidden,
					domain.WithMetadata("auth_user_id", authUser.Id),
					domain.WithMetadata("target_user_id", u.UserID))
			}
			if u.Role.Rank() >= domain.RoleAdmin.Rank() {
				return domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden,
					domain.WithMetadata("auth_user_id", authUser.Id),
					domain.WithMetadata("attempted_role", u.Role.String()))
			}
		}
	}
	return nil
}

func (p *userPolicy) DeletePreFetch(ctx context.Context, ids ...string) error {
	authUser := httpContext.MustGetUser(ctx)
	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return domain.NewError(domain.CodeUserRoleSignerAdminRequiredForbidden)
	}
	for _, id := range ids {
		if id == authUser.Id {
			return domain.NewError(domain.CodeUserDeleteSelfTargetForbidden, domain.WithMetadata("user_id", authUser.Id))
		}
	}
	return nil
}

func (p *userPolicy) DeletePostFetch(ctx context.Context, targets []domain.User) error {
	authUser := httpContext.MustGetUser(ctx)
	if authUser.Role.Rank() != domain.RoleAdmin.Rank() {
		return nil
	}
	for _, target := range targets {
		if target.Role.Rank() >= domain.RoleAdmin.Rank() {
			return domain.NewError(domain.CodeUserDeleteAdminForbidden,
				domain.WithMetadata("auth_user_id", authUser.Id),
				domain.WithMetadata("target_user_id", target.Id),
				domain.WithMetadata("target_role", target.Role.String()))
		}
	}
	return nil
}

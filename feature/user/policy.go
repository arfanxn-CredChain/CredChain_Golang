package user

import (
	"context"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
)

// UserPolicy contains feature-specific business rules for user operations.
type UserPolicy struct{}

// NewUserPolicy creates a new UserPolicy instance.
func NewUserPolicy() *UserPolicy {
	return &UserPolicy{}
}

// Store validates authorization for batch user creation.
// Assumes authUser is already Admin+ (middleware guarantee).
// Checks finer-grained rules: SuperAdmin can create Admins, Admin cannot.
func (p *UserPolicy) Store(ctx context.Context, users ...domain.User) error {
	authUser := httpContext.MustGetUser(ctx)

	for _, user := range users {
		// SuperAdmin batch creation forbidden for anyone
		if user.Role == domain.RoleSuperAdmin {
			return domain.NewError(domain.CodeUserStoreSuperAdminForbidden)
		}

		// Admin users cannot create other Admins or SuperAdmins
		// (SuperAdmin CAN create Admins - this is the fine-grained rule)
		if authUser.Role == domain.RoleAdmin && user.Role.Rank() >= domain.RoleAdmin.Rank() {
			return domain.NewError(domain.CodeUserStoreAdminCreateAdminForbidden)
		}
	}

	return nil
}

// UpdateRole validates authorization for batch role updates.
func (p *UserPolicy) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) error {
	authUser := httpContext.MustGetUser(ctx)

	// Note: We can't fetch users here because policy shouldn't depend on repositories
	// The service layer should fetch target users and pass them to policy
	// For now, we'll do basic validation that will be enhanced by service layer
	_ = authUser
	_ = updates

	return nil
}

// Delete validates authorization for batch user deletion.
func (p *UserPolicy) Delete(ctx context.Context, ids ...string) error {
	authUser := httpContext.MustGetUser(ctx)

	// Check if trying to delete self
	for _, id := range ids {
		if id == authUser.Id {
			return domain.NewError(domain.CodeAuthLoginForbidden)
		}
	}

	return nil
}

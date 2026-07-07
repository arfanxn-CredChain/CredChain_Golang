package user

import (
	"context"
	"testing"
	"time"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/testutil/fixtures"

	"github.com/stretchr/testify/assert"
)

func ctxWithUser(u *domain.User) context.Context {
	return context.WithValue(context.Background(), httpContext.UserKey, u)
}

func TestUserPolicy_Store_RejectsSuperAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	target := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))

	err := p.Store(ctxWithUser(&auth), target)
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserStoreSuperAdminForbidden, de.Code)
}

func TestUserPolicy_Store_AdminCannotCreateAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))

	err := p.Store(ctxWithUser(&auth), target)
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserStoreAdminCreateAdminForbidden, de.Code)
}

func TestUserPolicy_Store_AdminCanCreateIssuer(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))

	err := p.Store(ctxWithUser(&auth), target)
	assert.NoError(t, err)
}

func TestUserPolicy_Store_SuperAdminCanCreateAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	target := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))

	err := p.Store(ctxWithUser(&auth), target)
	assert.NoError(t, err)
}

func TestUserPolicy_UpdateRolePreFetch_AlwaysNil(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	err := p.UpdateRolePreFetch(ctxWithUser(&auth),
		domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	assert.NoError(t, err)
}

func TestUserPolicy_DeletePreFetch_RejectsSelf(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))

	err := p.DeletePreFetch(ctxWithUser(&auth), "self")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserDeleteSelfTargetForbidden, de.Code)
}

func TestUserPolicy_DeletePreFetch_AllowsOthers(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))

	err := p.DeletePreFetch(ctxWithUser(&auth), "other-id-1", "other-id-2")
	assert.NoError(t, err)
}

func TestUserPolicy_DeletePostFetch_AdminDeletingAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("a1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleAdmin))
	err := p.DeletePostFetch(ctxWithUser(&auth), []domain.User{target})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserDeleteAdminForbidden, de.Code)
}

func TestUserPolicy_DeletePostFetch_AdminDeletingSuperAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("a1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleSuperAdmin))
	err := p.DeletePostFetch(ctxWithUser(&auth), []domain.User{target})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserDeleteAdminForbidden, de.Code)
}

func TestUserPolicy_DeletePostFetch_AllowsHolder(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	err := p.DeletePostFetch(ctxWithUser(&auth), []domain.User{target})
	assert.NoError(t, err)
}

func TestUserPolicy_DeletePostFetch_SuperAdminCanDeleteAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleAdmin))
	err := p.DeletePostFetch(ctxWithUser(&auth), []domain.User{target})
	assert.NoError(t, err)
}

func TestUserPolicy_UpdateRolePreFetch_RejectsSuperAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	err := p.UpdateRolePreFetch(ctxWithUser(&auth), domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleSuperAdmin})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSuperAdminBatchForbidden, de.Code)
}

func TestUserPolicy_UpdateRolePreFetch_RejectsSelf(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))
	err := p.UpdateRolePreFetch(ctxWithUser(&auth), domain.UserRoleUpdate{UserID: "self", Role: domain.RoleIssuer})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSelfTargetForbidden, de.Code)
}

func TestUserPolicy_UpdateRolePostFetch_AdminPeerBlocked(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("a1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleAdmin))
	err := p.UpdateRolePostFetch(ctxWithUser(&auth), []domain.User{target},
		domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleAdminUpdatePeerForbidden, de.Code)
}

func TestUserPolicy_UpdateRolePostFetch_SameRole(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleIssuer))
	err := p.UpdateRolePostFetch(ctxWithUser(&auth), []domain.User{target},
		domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSameRoleUpdateForbidden, de.Code)
}

func TestUserPolicy_UpdateRolePostFetch_TargetNotFound(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	err := p.UpdateRolePostFetch(ctxWithUser(&auth), nil,
		domain.UserRoleUpdate{UserID: "missing", Role: domain.RoleIssuer})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserFetchNotFound, de.Code)
}

func TestUserPolicy_UpdatePreFetch_Self(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))
	err := p.UpdatePreFetch(ctxWithUser(&auth), domain.User{Id: "self"})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdateSelfForbidden, de.Code)
}

func TestUserPolicy_UpdatePreFetch_SelfAllowedForSuperAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("sa"), fixtures.WithRole(domain.RoleSuperAdmin))
	err := p.UpdatePreFetch(ctxWithUser(&auth), domain.User{Id: "sa"})
	assert.NoError(t, err, "SuperAdmin must be allowed to self-edit profile via batch")
}

func TestUserPolicy_UpdatePostFetch_SelfEmailForbidden(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("sa"), fixtures.WithRole(domain.RoleSuperAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("sa"), fixtures.WithRole(domain.RoleSuperAdmin))
	err := p.UpdatePostFetch(ctxWithUser(&auth),
		[]domain.User{target},
		[]domain.User{{Id: "sa", Email: "new@example.com"}})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdateSelfEmailForbidden, de.Code,
		"SuperAdmin must be blocked from changing own email via batch — use /users/self/email")
}

func TestUserPolicy_UpdatePostFetch_SuperAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleSuperAdmin))
	err := p.UpdatePostFetch(ctxWithUser(&auth), []domain.User{target}, []domain.User{{Id: "u1"}})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdateSuperAdminForbidden, de.Code)
}

func TestUserPolicy_UpdatePostFetch_AdminPeer(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleAdmin))
	err := p.UpdatePostFetch(ctxWithUser(&auth), []domain.User{target}, []domain.User{{Id: "u1"}})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdatePeerAdminForbidden, de.Code)
}

func TestUserPolicy_UpdatePostFetch_AllowsHolder(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	err := p.UpdatePostFetch(ctxWithUser(&auth), []domain.User{target}, []domain.User{{Id: "u1"}})
	assert.NoError(t, err)
}

func TestUserPolicy_UpdatePostFetch_AdminPromotingToAdminForbidden(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	err := p.UpdatePostFetch(ctxWithUser(&auth), []domain.User{target},
		[]domain.User{{Id: "u1", Role: domain.RoleAdmin}})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSignerAdminRequiredForbidden, de.Code)
}

func TestUserPolicy_UpdatePostFetch_AssigningSuperAdminForbidden(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	err := p.UpdatePostFetch(ctxWithUser(&auth), []domain.User{target},
		[]domain.User{{Id: "u1", Role: domain.RoleSuperAdmin}})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSuperAdminBatchForbidden, de.Code)
}

func TestUserPolicy_UpdatePostFetch_AdminPromotingToIssuerAllowed(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	err := p.UpdatePostFetch(ctxWithUser(&auth), []domain.User{target},
		[]domain.User{{Id: "u1", Role: domain.RoleIssuer}})
	assert.NoError(t, err)
}

func TestUserPolicy_UpdatePostFetch_TrashedTargetForbidden(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	target := fixtures.NewDomainUser(fixtures.WithID("trashed"), fixtures.WithRole(domain.RoleHolder))
	target.DeletedAt = &deletedAt
	newName := "Renamed"
	err := p.UpdatePostFetch(ctxWithUser(&auth), []domain.User{target},
		[]domain.User{{Id: "trashed", Name: &newName}})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdateTrashedForbidden, de.Code)
	assert.Equal(t, "trashed", de.Metadata["user_id"])
}

func TestUserPolicy_UpdateRolePostFetch_TrashedTargetForbidden(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	target := fixtures.NewDomainUser(fixtures.WithID("trashed"), fixtures.WithRole(domain.RoleHolder))
	target.DeletedAt = &deletedAt
	err := p.UpdateRolePostFetch(ctxWithUser(&auth), []domain.User{target},
		domain.UserRoleUpdate{UserID: "trashed", Role: domain.RoleIssuer})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleTrashedForbidden, de.Code)
	assert.Equal(t, "trashed", de.Metadata["user_id"])
}

func TestUserPolicy_TransferSuperAdminPreFetch_DifferentIds_OK(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	ctx := ctxWithUser(&auth)

	err := p.TransferSuperAdminPreFetch(ctx, "some-other-id")
	assert.NoError(t, err)
}

func TestUserPolicy_TransferSuperAdminPreFetch_SameId_Forbidden(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	ctx := ctxWithUser(&auth)

	err := p.TransferSuperAdminPreFetch(ctx, auth.Id)
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserTransferSuperAdminSelfTargetForbidden, de.Code)
}

package user

import (
	"context"
	"testing"

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
	assert.Equal(t, domain.CodeAuthForbidden, de.Code)
}

func TestUserPolicy_DeletePreFetch_AllowsOthers(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))

	err := p.DeletePreFetch(ctxWithUser(&auth), "other-id-1", "other-id-2")
	assert.NoError(t, err)
}

func TestUserPolicy_DeletePreFetch_BelowAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	err := p.DeletePreFetch(ctxWithUser(&auth), "u1")
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSignerAdminRequiredForbidden, de.Code)
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

func TestUserPolicy_UpdateRolePreFetch_BelowAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	err := p.UpdateRolePreFetch(ctxWithUser(&auth), domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSignerAdminRequiredForbidden, de.Code)
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
	assert.Equal(t, domain.CodeAuthForbidden, de.Code)
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

func TestUserPolicy_UpdatePreFetch_BelowAdmin(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	err := p.UpdatePreFetch(ctxWithUser(&auth), domain.User{Id: "u1"})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSignerAdminRequiredForbidden, de.Code)
}

func TestUserPolicy_UpdatePreFetch_Self(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))
	err := p.UpdatePreFetch(ctxWithUser(&auth), domain.User{Id: "self"})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdateSelfForbidden, de.Code)
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

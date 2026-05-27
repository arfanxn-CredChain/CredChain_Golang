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

func TestUserPolicy_UpdateRole_AlwaysNil(t *testing.T) {
	p := NewUserPolicy()
	err := p.UpdateRole(context.Background(),
		domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	assert.NoError(t, err)
}

func TestUserPolicy_Delete_RejectsSelf(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("self"))

	err := p.Delete(ctxWithUser(&auth), "self")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeAuthForbidden, de.Code)
}

func TestUserPolicy_Delete_AllowsOthers(t *testing.T) {
	p := NewUserPolicy()
	auth := fixtures.NewDomainUser(fixtures.WithID("self"))

	err := p.Delete(ctxWithUser(&auth), "other-id-1", "other-id-2")
	assert.NoError(t, err)
}

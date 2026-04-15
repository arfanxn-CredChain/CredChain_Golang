package user

import (
	"context"
	"fmt"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)


func TestService_BatchUpdateRole(t *testing.T) {
	mockRepo := new(MockUserRepository)
	cfg := &config.Config{JWTSecret: "super-secret-key"}
	svc := NewService(UserServiceParams{UserRepo: mockRepo, Config: cfg})
	ctx := context.Background()

	// Setup users for test
	targetSuperAdmin := domain.User{ID: "super_admin_id", Role: domain.RoleSuperAdmin}
	targetUser := domain.User{ID: "target_id", Role: domain.RoleHolder}

	// 1. Issuer or Holder cannot promote someone (Requires Admin+)
	t.Run("Non-Admin Cannot Promote", func(t *testing.T) {
		err := svc.BatchUpdateRole(ctx, domain.RoleHolder, []domain.UserRoleUpdate{{UserID: "target_id", Role: domain.RoleAdmin}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("%d", domain.CodeUserRoleSignerAdminRequiredForbidden))
	})

	// 2. Admin cannot update SuperAdmin
	t.Run("Admin Cannot Update SuperAdmin", func(t *testing.T) {
		mockRepo.On("GetUsersByIDs", ctx, []string{"super_admin_id"}).Return([]domain.User{targetSuperAdmin}, nil).Once()

		err := svc.BatchUpdateRole(ctx, domain.RoleAdmin, []domain.UserRoleUpdate{{UserID: "super_admin_id", Role: domain.RoleHolder}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("%d", domain.CodeUserRoleAdminUpdatePeerForbidden))
	})

	// 3. Admin cannot promote Holder to Admin
	t.Run("Admin Cannot Promote To Admin", func(t *testing.T) {
		mockRepo.On("GetUsersByIDs", ctx, []string{"target_id"}).Return([]domain.User{targetUser}, nil).Once()

		err := svc.BatchUpdateRole(ctx, domain.RoleAdmin, []domain.UserRoleUpdate{{UserID: "target_id", Role: domain.RoleAdmin}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("%d", domain.CodeUserRoleSignerAdminRequiredForbidden))
	})

	// 4. SuperAdmin can promote Holder to Admin
	t.Run("SuperAdmin Can Promote To Admin", func(t *testing.T) {
		updates := []domain.UserRoleUpdate{{UserID: "target_id", Role: domain.RoleAdmin}}
		mockRepo.On("GetUsersByIDs", ctx, []string{"target_id"}).Return([]domain.User{targetUser}, nil).Once()
		mockRepo.On("BatchUpdateRole", ctx, updates).Return(nil).Once()

		err := svc.BatchUpdateRole(ctx, domain.RoleSuperAdmin, updates)
		assert.NoError(t, err)
	})
}

func TestService_CreateUsers(t *testing.T) {
	mockRepo := new(MockUserRepository)
	cfg := &config.Config{JWTSecret: "test-secret-key-32-bytes-long!!!"} // Needs >= 16 bytes for AES usually, let's use 32 bytes
	svc := NewService(UserServiceParams{UserRepo: mockRepo, Config: cfg})
	ctx := context.Background()

	t.Run("Successfully Create Users", func(t *testing.T) {
		reqs := []CreateUserRequest{
			{Name: "Alice", Email: "alice@example.com", Role: domain.RoleHolder},
		}

		mockRepo.On("BatchCreate", ctx, mock.AnythingOfType("[]domain.User")).Return([]domain.User{
			{ID: "dummy_id", Email: "alice@example.com", Role: domain.RoleHolder},
		}, nil).Once()

		res, err := svc.CreateUsers(ctx, reqs)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		mockRepo.AssertExpectations(t)
	})
}

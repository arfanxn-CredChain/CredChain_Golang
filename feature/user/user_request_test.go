package user

import (
	"testing"

	"CredChain_Golang/domain"
	"github.com/stretchr/testify/assert"
)

func TestUserStoreInput_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UserStoreInput
		shouldErr bool
	}{
		{name: "Valid Request", req: UserStoreInput{Name: "John Doe", Email: "john@example.com", Role: domain.RoleHolder}, shouldErr: false},
		{name: "Missing Name", req: UserStoreInput{Email: "john@example.com", Role: domain.RoleHolder}, shouldErr: true},
		{name: "Invalid Email", req: UserStoreInput{Name: "John Doe", Email: "john-example.com", Role: domain.RoleHolder}, shouldErr: true},
		{name: "Invalid Role", req: UserStoreInput{Name: "John Doe", Email: "john@example.com", Role: domain.Role("invalid_role")}, shouldErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserUpdateEmailRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UserUpdateEmailRequest
		shouldErr bool
	}{
		{name: "Valid Email", req: UserUpdateEmailRequest{Email: "alice@gmail.com"}, shouldErr: false},
		{name: "Invalid Email", req: UserUpdateEmailRequest{Email: "alicetest.com"}, shouldErr: true},
		{name: "Empty Email", req: UserUpdateEmailRequest{Email: ""}, shouldErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserRoleUpdateRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UserRoleUpdateRequest
		shouldErr bool
	}{
		{name: "Valid Role Update", req: UserRoleUpdateRequest{UserID: "01HXXXXX...", Role: domain.RoleIssuer}, shouldErr: false},
		{name: "Missing UserID", req: UserRoleUpdateRequest{Role: domain.RoleIssuer}, shouldErr: true},
		{name: "Invalid Role", req: UserRoleUpdateRequest{UserID: "01HXXXXX...", Role: domain.Role("fake_role")}, shouldErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

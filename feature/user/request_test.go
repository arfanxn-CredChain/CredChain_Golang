package user

import (
	"testing"

	"CredChain_Golang/domain"
	"github.com/stretchr/testify/assert"
)


func TestStoreUserInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     StoreUserInput
		shouldErr bool
	}{
		{
			name: "Valid Request",
			req: StoreUserInput{
				Name:  "John Doe",
				Email: "john@example.com",
				Role:  domain.RoleHolder,
			},
			shouldErr: false,
		},
		{
			name: "Missing Name",
			req: StoreUserInput{
				Email: "john@example.com",
				Role:  domain.RoleHolder,
			},
			shouldErr: true,
		},
		{
			name: "Invalid Email",
			req: StoreUserInput{
				Name:  "John Doe",
				Email: "john-example.com",
				Role:  domain.RoleHolder,
			},
			shouldErr: true,
		},
		{
			name: "Invalid Role",
			req: StoreUserInput{
				Name:  "John Doe",
				Email: "john@example.com",
				Role:  domain.Role("invalid_role"),
			},
			shouldErr: true,
		},
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

func TestUpdateEmailRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UpdateEmailRequest
		shouldErr bool
	}{
		{
			name:      "Valid Email",
			req:       UpdateEmailRequest{Email: "alice@gmail.com"},
			shouldErr: false,
		},
		{
			name:      "Invalid Email",
			req:       UpdateEmailRequest{Email: "alicetest.com"},
			shouldErr: true,
		},
		{
			name:      "Empty Email",
			req:       UpdateEmailRequest{Email: ""},
			shouldErr: true,
		},
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
		{
			name: "Valid Role Update",
			req: UserRoleUpdateRequest{
				UserID: "01HXXXXX...",
				Role:   domain.RoleIssuer,
			},
			shouldErr: false,
		},
		{
			name: "Missing UserID",
			req: UserRoleUpdateRequest{
				Role: domain.RoleIssuer,
			},
			shouldErr: true,
		},
		{
			name: "Invalid Role",
			req: UserRoleUpdateRequest{
				UserID: "01HXXXXX...",
				Role:   domain.Role("fake_role"),
			},
			shouldErr: true,
		},
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

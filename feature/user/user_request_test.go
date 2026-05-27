package user

import (
	"testing"
	"time"

	"CredChain_Golang/domain"
	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string { return &s }

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
		{
			name: "Valid with all optional fields",
			req: UserStoreInput{
				Name:        "Jane Doe",
				Email:       "jane@example.com",
				Role:        domain.RoleIssuer,
				Number:      strPtr("12345"),
				PhoneNumber: strPtr("+62812345678"),
				BirthDate:   strPtr("1990-01-01"),
				Meta:        map[string]any{"key": "value"},
			},
			shouldErr: false,
		},
		{
			name: "Invalid BirthDate format",
			req: UserStoreInput{
				Name:      "John Doe",
				Email:     "john@example.com",
				Role:      domain.RoleHolder,
				BirthDate: strPtr("1990/01/01"),
			},
			shouldErr: true,
		},
		{
			name: "Invalid BirthDate not a date",
			req: UserStoreInput{
				Name:      "John Doe",
				Email:     "john@example.com",
				Role:      domain.RoleHolder,
				BirthDate: strPtr("not-a-date"),
			},
			shouldErr: true,
		},
		{
			name: "Empty BirthDate string is valid",
			req: UserStoreInput{
				Name:      "John Doe",
				Email:     "john@example.com",
				Role:      domain.RoleHolder,
				BirthDate: strPtr(""),
			},
			shouldErr: false,
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

func TestUserStoreInput_ToDomain(t *testing.T) {
	t.Run("All fields set", func(t *testing.T) {
		number := "12345"
		phone := "+62812345678"
		birthDate := "1990-01-01"
		meta := map[string]any{"key": "value"}
		input := UserStoreInput{
			Name:        "John Doe",
			Email:       "john@example.com",
			Role:        domain.RoleHolder,
			Number:      &number,
			PhoneNumber: &phone,
			BirthDate:   &birthDate,
			Meta:        meta,
		}

		got := input.ToDomain()

		assert.NotNil(t, got.Name)
		assert.Equal(t, "John Doe", *got.Name)
		assert.Equal(t, "john@example.com", got.Email)
		assert.Equal(t, domain.RoleHolder, got.Role)
		assert.Equal(t, &number, got.Number)
		assert.Equal(t, &phone, got.PhoneNumber)
		assert.NotNil(t, got.BirthDate)
		assert.Equal(t, time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), *got.BirthDate)
		assert.Equal(t, meta, got.Meta)
	})

	t.Run("Only required fields", func(t *testing.T) {
		input := UserStoreInput{
			Name:  "John Doe",
			Email: "john@example.com",
			Role:  domain.RoleHolder,
		}

		got := input.ToDomain()

		assert.NotNil(t, got.Name)
		assert.Equal(t, "John Doe", *got.Name)
		assert.Equal(t, "john@example.com", got.Email)
		assert.Equal(t, domain.RoleHolder, got.Role)
		assert.Nil(t, got.Number)
		assert.Nil(t, got.PhoneNumber)
		assert.Nil(t, got.BirthDate)
		assert.Nil(t, got.Meta)
	})

	t.Run("Empty BirthDate string yields nil domain BirthDate", func(t *testing.T) {
		empty := ""
		input := UserStoreInput{
			Name:      "John Doe",
			Email:     "john@example.com",
			Role:      domain.RoleHolder,
			BirthDate: &empty,
		}

		got := input.ToDomain()

		assert.Nil(t, got.BirthDate)
	})
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

func TestUserStoreRequest_Validate_EmptyUsers(t *testing.T) {
	r := UserStoreRequest{Users: nil}
	assert.Error(t, r.Validate())
}

func TestUserStoreRequest_Validate_InvalidNested(t *testing.T) {
	r := UserStoreRequest{Users: []UserStoreInput{
		{Name: "Bob", Email: "not-an-email", Role: domain.RoleHolder},
	}}
	assert.Error(t, r.Validate())
}

func TestUserStoreRequest_Validate_Valid(t *testing.T) {
	r := UserStoreRequest{Users: []UserStoreInput{
		{Name: "Bob", Email: "bob@x.com", Role: domain.RoleHolder},
	}}
	assert.NoError(t, r.Validate())
}

func TestUserStoreRequest_ToDomain_NilSlice(t *testing.T) {
	r := UserStoreRequest{Users: nil}
	got := r.ToDomain()
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestUserStoreRequest_ToDomain_MultiUser(t *testing.T) {
	r := UserStoreRequest{Users: []UserStoreInput{
		{Name: "Alice", Email: "a@x.com", Role: domain.RoleHolder},
		{Name: "Bob", Email: "b@x.com", Role: domain.RoleIssuer},
	}}
	got := r.ToDomain()
	assert.Len(t, got, 2)
	assert.Equal(t, "a@x.com", got[0].Email)
	assert.Equal(t, "b@x.com", got[1].Email)
}

func TestUserUpdateProfileRequest_Validate_AlwaysNil(t *testing.T) {
	r := UserUpdateProfileRequest{}
	assert.NoError(t, r.Validate())
	name := "x"
	r2 := UserUpdateProfileRequest{Name: &name}
	assert.NoError(t, r2.Validate())
}

func TestUserBatchUpdateRoleRequest_Validate_Empty(t *testing.T) {
	r := UserBatchUpdateRoleRequest{UserRoles: nil}
	assert.Error(t, r.Validate())
}

func TestUserBatchUpdateRoleRequest_Validate_Valid(t *testing.T) {
	r := UserBatchUpdateRoleRequest{UserRoles: []UserRoleUpdateRequest{
		{UserID: "u1", Role: domain.RoleIssuer},
	}}
	assert.NoError(t, r.Validate())
}

func TestUserBatchDeleteRequest_Validate_Empty(t *testing.T) {
	r := UserBatchDeleteRequest{UserIDs: nil}
	assert.Error(t, r.Validate())
}

func TestUserBatchDeleteRequest_Validate_Valid(t *testing.T) {
	r := UserBatchDeleteRequest{UserIDs: []string{"u1", "u2"}}
	assert.NoError(t, r.Validate())
}

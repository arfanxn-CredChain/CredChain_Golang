package user

import (
	"strings"
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

func TestUserUpdateSelfEmailRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UserUpdateSelfEmailRequest
		shouldErr bool
	}{
		{name: "Valid Email with IdToken", req: UserUpdateSelfEmailRequest{Email: "alice@gmail.com", IdToken: "token"}, shouldErr: false},
		{name: "Invalid Email", req: UserUpdateSelfEmailRequest{Email: "alicetest.com", IdToken: "token"}, shouldErr: true},
		{name: "Empty Email", req: UserUpdateSelfEmailRequest{Email: "", IdToken: "token"}, shouldErr: true},
		{name: "Missing IdToken", req: UserUpdateSelfEmailRequest{Email: "alice@gmail.com"}, shouldErr: true},
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

func TestUserRoleUpdateInput_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UserRoleUpdateInput
		shouldErr bool
	}{
		{name: "Valid Role Update", req: UserRoleUpdateInput{UserID: "01HXXXXX...", Role: domain.RoleIssuer}, shouldErr: false},
		{name: "Missing UserID", req: UserRoleUpdateInput{Role: domain.RoleIssuer}, shouldErr: true},
		{name: "Invalid Role", req: UserRoleUpdateInput{UserID: "01HXXXXX...", Role: domain.Role("fake_role")}, shouldErr: true},
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

func TestUserUpdateSelfProfileRequest_Validate_AlwaysNil(t *testing.T) {
	r := UserUpdateSelfProfileRequest{}
	assert.NoError(t, r.Validate())
	name := "x"
	r2 := UserUpdateSelfProfileRequest{Name: &name}
	assert.NoError(t, r2.Validate())
}

func TestUserUpdateRoleRequest_Validate_Empty(t *testing.T) {
	r := UserUpdateRoleRequest{UserRoles: nil}
	assert.Error(t, r.Validate())
}

func TestUserUpdateRoleRequest_Validate_Valid(t *testing.T) {
	r := UserUpdateRoleRequest{UserRoles: []UserRoleUpdateInput{
		{UserID: "u1", Role: domain.RoleIssuer},
	}}
	assert.NoError(t, r.Validate())
}

func TestUserDeleteRequest_Validate_Empty(t *testing.T) {
	r := UserDeleteRequest{Ids: nil}
	assert.Error(t, r.Validate())
}

func TestUserDeleteRequest_Validate_Valid(t *testing.T) {
	r := UserDeleteRequest{Ids: []string{"u1", "u2"}}
	assert.NoError(t, r.Validate())
}

func TestUserStoreInput_NameOverMax(t *testing.T) {
	over := strings.Repeat("a", 257)
	in := UserStoreInput{Name: over, Email: "ok@x.com", Role: domain.RoleHolder}
	assert.Error(t, in.Validate())
}

func TestUserStoreInput_NameAtMax(t *testing.T) {
	atMax := strings.Repeat("a", 256)
	in := UserStoreInput{Name: atMax, Email: "ok@x.com", Role: domain.RoleHolder}
	assert.NoError(t, in.Validate())
}

func TestUserStoreInput_PhoneOverMax(t *testing.T) {
	over := "+1" + strings.Repeat("9", 18)
	in := UserStoreInput{Name: "n", Email: "ok@x.com", Role: domain.RoleHolder, PhoneNumber: &over}
	assert.Error(t, in.Validate())
}

func TestUserStoreInput_PhoneNotE164(t *testing.T) {
	bad := "abc-123"
	in := UserStoreInput{Name: "n", Email: "ok@x.com", Role: domain.RoleHolder, PhoneNumber: &bad}
	assert.Error(t, in.Validate())
}

func TestUserStoreInput_PhoneBareCountryCode(t *testing.T) {
	bad := "+62"
	in := UserStoreInput{Name: "n", Email: "ok@x.com", Role: domain.RoleHolder, PhoneNumber: &bad}
	assert.Error(t, in.Validate(), "+62 (bare country code) must be rejected by strict E.164")
}

func TestUserStoreInput_PhoneValidE164(t *testing.T) {
	ok := "+6281234567890"
	in := UserStoreInput{Name: "n", Email: "ok@x.com", Role: domain.RoleHolder, PhoneNumber: &ok}
	assert.NoError(t, in.Validate())
}

func TestUserStoreInput_EmailOverMax(t *testing.T) {
	longLocal := strings.Repeat("a", 250)
	in := UserStoreInput{Name: "n", Email: longLocal + "@x.com", Role: domain.RoleHolder}
	assert.Error(t, in.Validate())
}

func TestUserUpdateSelfProfileRequest_PhoneInvalid(t *testing.T) {
	bad := "12-34"
	r := UserUpdateSelfProfileRequest{PhoneNumber: &bad}
	assert.Error(t, r.Validate())
}

func TestUserUpdateSelfProfileRequest_NameAtMax(t *testing.T) {
	n := strings.Repeat("a", 256)
	r := UserUpdateSelfProfileRequest{Name: &n}
	assert.NoError(t, r.Validate())
}

func TestUserUpdateSelfProfileRequest_NameOverMax(t *testing.T) {
	n := strings.Repeat("a", 257)
	r := UserUpdateSelfProfileRequest{Name: &n}
	assert.Error(t, r.Validate())
}

func TestUserUpdateSelfEmailRequest_EmailOverMax(t *testing.T) {
	longLocal := strings.Repeat("a", 250)
	r := UserUpdateSelfEmailRequest{Email: longLocal + "@x.com"}
	assert.Error(t, r.Validate())
}

func TestUserUpdateSelfProfileRequest_BirthDateInvalid(t *testing.T) {
	bad := "1990/01/01"
	r := UserUpdateSelfProfileRequest{BirthDate: &bad}
	assert.Error(t, r.Validate())
}

func TestUserUpdateSelfProfileRequest_BirthDateValid(t *testing.T) {
	ok := "1990-01-01"
	r := UserUpdateSelfProfileRequest{BirthDate: &ok}
	assert.NoError(t, r.Validate())
}

func TestUserUpdateSelfEmailRequest_Validate_RequiresIdToken(t *testing.T) {
	r := UserUpdateSelfEmailRequest{Email: "ok@x.com", IdToken: ""}
	assert.Error(t, r.Validate())
}

func TestUserUpdateSelfEmailRequest_Validate_ValidWithIdToken(t *testing.T) {
	r := UserUpdateSelfEmailRequest{Email: "ok@x.com", IdToken: "some-token"}
	assert.NoError(t, r.Validate())
}

func TestUserUpdateInput_Validate_RequiresId(t *testing.T) {
	in := UserUpdateInput{}
	assert.Error(t, in.Validate())
}

func TestUserUpdateInput_Validate_Valid(t *testing.T) {
	n := "Alice"
	in := UserUpdateInput{Id: "u1", Name: &n}
	assert.NoError(t, in.Validate())
}

func TestUserUpdateInput_Validate_PhoneInvalid(t *testing.T) {
	bad := "not-a-phone"
	in := UserUpdateInput{Id: "u1", PhoneNumber: &bad}
	assert.Error(t, in.Validate())
}

func TestUserUpdateRequest_Validate_Empty(t *testing.T) {
	r := UserUpdateRequest{Users: nil}
	assert.Error(t, r.Validate())
}

func TestUserUpdateRequest_ToDomain(t *testing.T) {
	n := "Alice"
	r := UserUpdateRequest{Users: []UserUpdateInput{{Id: "u1", Name: &n}}}
	users := r.ToDomain()
	assert.Len(t, users, 1)
	assert.Equal(t, "u1", users[0].Id)
}

func TestUserUpdateInput_Validate_EmailValid(t *testing.T) {
	email := "ok@x.com"
	in := UserUpdateInput{Id: "u1", Email: &email}
	assert.NoError(t, in.Validate())
}

func TestUserUpdateInput_Validate_EmailInvalid(t *testing.T) {
	email := "not-an-email"
	in := UserUpdateInput{Id: "u1", Email: &email}
	assert.Error(t, in.Validate())
}

func TestUserUpdateInput_Validate_RoleValid(t *testing.T) {
	role := domain.RoleIssuer
	in := UserUpdateInput{Id: "u1", Role: &role}
	assert.NoError(t, in.Validate())
}

func TestUserUpdateInput_Validate_RoleSuperAdminRejected(t *testing.T) {
	role := domain.RoleSuperAdmin
	in := UserUpdateInput{Id: "u1", Role: &role}
	assert.Error(t, in.Validate())
}

func TestUserUpdateInput_ToDomain_PropagatesEmailAndRole(t *testing.T) {
	email := "new@x.com"
	role := domain.RoleIssuer
	in := UserUpdateInput{Id: "u1", Email: &email, Role: &role}
	u := in.ToDomain()
	assert.Equal(t, "new@x.com", u.Email)
	assert.Equal(t, domain.RoleIssuer, u.Role)
}

func TestUserTransferSuperAdminRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UserTransferSuperAdminRequest
		wantErr bool
	}{
		{"valid UUID", UserTransferSuperAdminRequest{Id: "123e4567-e89b-12d3-a456-426614174000"}, false},
		{"empty id", UserTransferSuperAdminRequest{Id: ""}, true},
		{"non-UUID string", UserTransferSuperAdminRequest{Id: "not-a-uuid"}, true},
		{"plain number", UserTransferSuperAdminRequest{Id: "12345"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

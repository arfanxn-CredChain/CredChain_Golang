package domain

import (
	"context"
	"time"
)

// Role defines the allowed role types mapped to the Postgres ENUM
type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleIssuer     Role = "issuer"
	RoleHolder     Role = "holder"
)

// Rank returns the numeric hierarchy level of the role (higher is more privileged)
func (r Role) Rank() int {
	switch r {
	case RoleSuperAdmin:
		return 4
	case RoleAdmin:
		return 3
	case RoleIssuer:
		return 2
	case RoleHolder:
		return 1
	default:
		return 0
	}
}

// UserRoleUpdate defines a single target role update for a user
type UserRoleUpdate struct {
	UserID string
	Role   Role
}

// User represents a row in the users table
type User struct {
	ID               string     `db:"id" json:"id"`
	Name             *string    `db:"name" json:"name"`
	Number           *string    `db:"number" json:"number"`
	PhoneNumber      *string    `db:"phone_number" json:"phone_number"`
	Email            string     `db:"email" json:"email"`
	BirthDate        *time.Time `db:"birth_date" json:"birth_date"`
	Meta             *JSONB     `db:"meta" json:"meta"`
	Role             Role       `db:"role" json:"role"`
	WalletAddress    string     `db:"wallet_address" json:"wallet_address"`
	WalletPrivateKey string     `db:"wallet_private_key" json:"-"` // never expose private key in json
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        *time.Time `db:"updated_at" json:"updated_at"`
}

// UserRepository defines the database contract for the User Domain
type UserRepository interface {
	GetUsers(ctx context.Context, query Query) ([]User, int, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUsersByIDs(ctx context.Context, ids []string) ([]User, error)
	UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta *JSONB) (*User, error)
	UpdateEmail(ctx context.Context, id string, email string) (string, error)
	BatchUpdateRole(ctx context.Context, updates []UserRoleUpdate) error
	BatchCreate(ctx context.Context, users []User) ([]User, error)
	DeleteUsersBatch(ctx context.Context, ids []string) error
}

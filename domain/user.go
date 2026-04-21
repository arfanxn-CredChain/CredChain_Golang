package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
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
	Id               string     `json:"id"`
	Name             *string    `json:"name"`
	Number           *string    `json:"number"`
	PhoneNumber      *string    `json:"phone_number"`
	Email            string     `json:"email"`
	BirthDate        *time.Time `json:"birth_date"`
	Meta             *JSONB     `json:"meta"`
	Role             Role       `json:"role"`
	WalletAddress    string     `json:"wallet_address"`
	WalletPrivateKey string     `json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

// UserRepository defines the database contract for the User Domain
type UserRepository interface {
	// Query-based retrieval
	Get(ctx context.Context, query *domainQuery.Query) ([]User, int, error)

	// Single item lookups
	Find(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)

	// Multiple item lookups
	FindByIds(ctx context.Context, ids ...string) ([]User, error)

	// CRUD operations
	Update(ctx context.Context, user User) (*User, error)
	Store(ctx context.Context, users ...User) ([]User, error)
	Destroy(ctx context.Context, ids ...string) error

	// Specialized operations
	UpdateRole(ctx context.Context, users ...User) error
}

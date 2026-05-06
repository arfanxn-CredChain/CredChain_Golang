package response

import (
	"CredChain_Golang/domain"
	"time"
)

// User is the response DTO for user data.
// Excludes sensitive fields like EncryptedWalletPrivateKey.
type User struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	Role          domain.Role `json:"role"`
	WalletAddress string     `json:"wallet_address"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// FromDomainUser converts a domain User entity to a response User DTO.
func FromDomainUser(u domain.User) User {
	name := ""
	if u.Name != nil {
		name = *u.Name
	}
	return User{
		ID:            u.Id,
		Name:          name,
		Email:         u.Email,
		Role:          u.Role,
		WalletAddress: u.WalletAddress,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

// ToDomain converts a response User DTO to a domain User entity.
func (u *User) ToDomain() domain.User {
	return domain.User{
		Id:            u.ID,
		Name:          &u.Name,
		Email:         u.Email,
		Role:          u.Role,
		WalletAddress: u.WalletAddress,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

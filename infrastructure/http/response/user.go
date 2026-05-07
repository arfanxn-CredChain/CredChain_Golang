package response

import (
	"CredChain_Golang/domain"
	"time"
)

// User is the response DTO for user data.
// Excludes sensitive fields like EncryptedWalletPrivateKey.
type User struct {
	ID            string        `json:"id"`
	Name          *string       `json:"name"`
	Number        *string       `json:"number"`
	PhoneNumber   *string       `json:"phone_number"`
	Email         string        `json:"email"`
	BirthDate     *time.Time    `json:"birth_date"`
	Meta          *domain.JSONB `json:"meta"`
	Role          domain.Role   `json:"role"`
	WalletAddress string        `json:"wallet_address"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     *time.Time    `json:"updated_at"`
}

// FromDomainUser converts a domain User entity to a response User DTO.
func FromDomainUser(u domain.User) User {
	return User{
		ID:            u.Id,
		Name:          u.Name,
		Number:        u.Number,
		PhoneNumber:   u.PhoneNumber,
		Email:         u.Email,
		BirthDate:     u.BirthDate,
		Meta:          u.Meta,
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
		Name:          u.Name,
		Number:        u.Number,
		PhoneNumber:   u.PhoneNumber,
		Email:         u.Email,
		BirthDate:     u.BirthDate,
		Meta:          u.Meta,
		Role:          u.Role,
		WalletAddress: u.WalletAddress,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}
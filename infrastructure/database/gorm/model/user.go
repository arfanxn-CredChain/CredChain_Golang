package model

import (
	"CredChain_Golang/domain"
	"time"
)

type User struct {
	Id               string         `gorm:"primaryKey;type:varchar(255);column:id" json:"id"`
	Name             *string        `gorm:"type:varchar(255);column:name" json:"name"`
	Number           *string        `gorm:"type:varchar(50);column:number" json:"number"`
	PhoneNumber      *string        `gorm:"type:varchar(50);column:phone_number" json:"phone_number"`
	Email            string         `gorm:"type:varchar(255);uniqueIndex;column:email" json:"email"`
	BirthDate        *time.Time     `gorm:"column:birth_date" json:"birth_date"`
	Meta             map[string]any `gorm:"type:jsonb;serializer:json;column:meta" json:"meta"`
	Role             string         `gorm:"type:varchar(50);column:role" json:"role"`
	WalletAddress    string         `gorm:"type:varchar(255);column:wallet_address" json:"wallet_address"`
	EncryptedWalletPrivateKey string         `gorm:"type:varchar(255);column:encrypted_wallet_private_key" json:"-"`
	CreatedAt        time.Time      `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt        *time.Time     `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`
}

func (m *User) ToDomain() domain.User {
	return domain.User{
		Id:               m.Id,
		Name:             m.Name,
		Number:           m.Number,
		PhoneNumber:      m.PhoneNumber,
		Email:            m.Email,
		BirthDate:        m.BirthDate,
		Meta:             m.Meta,
		Role:             domain.Role(m.Role),
		WalletAddress:    m.WalletAddress,
		EncryptedWalletPrivateKey: m.EncryptedWalletPrivateKey,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func FromDomainUser(u domain.User) User {
	return User{
		Id:               u.Id,
		Name:             u.Name,
		Number:           u.Number,
		PhoneNumber:      u.PhoneNumber,
		Email:            u.Email,
		BirthDate:        u.BirthDate,
		Meta:             u.Meta,
		Role:             string(u.Role),
		WalletAddress:    u.WalletAddress,
		EncryptedWalletPrivateKey: u.EncryptedWalletPrivateKey,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

package model

import (
	"CredChain_Golang/domain"
	"time"
)

type UserToken struct {
	Id         string     `gorm:"primaryKey;type:varchar(255);column:id" json:"id"`
	UserId     string     `gorm:"type:varchar(255);index;column:user_id" json:"user_id"`
	Type       string     `gorm:"type:user_token_type;column:type" json:"type"`
	Token      string     `gorm:"type:varchar(512);uniqueIndex;column:token" json:"token"`
	LastUsedAt *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
	ExpiresAt  *time.Time `gorm:"column:expires_at" json:"expires_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at" json:"revoked_at"`
	CreatedAt  time.Time  `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`
}

func (m *UserToken) ToDomain() domain.UserToken {
	return domain.UserToken{
		Id:         m.Id,
		UserId:     m.UserId,
		Type:       domain.UserTokenType(m.Type),
		Token:      m.Token,
		LastUsedAt: m.LastUsedAt,
		ExpiresAt:  m.ExpiresAt,
		RevokedAt:  m.RevokedAt,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func FromDomainUserToken(t domain.UserToken) UserToken {
	return UserToken{
		Id:         t.Id,
		UserId:     t.UserId,
		Type:       string(t.Type),
		Token:      t.Token,
		LastUsedAt: t.LastUsedAt,
		ExpiresAt:  t.ExpiresAt,
		RevokedAt:  t.RevokedAt,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

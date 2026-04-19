package model

import (
	"CredChain_Golang/domain"
	"time"
)

type Credential struct {
	Id        string    `gorm:"primaryKey;type:varchar(255);column:id"`
	HolderId  string    `gorm:"type:varchar(255);index;column:holder_user_id"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at"`
}

func (m *Credential) ToDomain() domain.Credential {
	return domain.Credential{}
}

func FromDomainCredential(c domain.Credential) Credential {
	return Credential{}
}

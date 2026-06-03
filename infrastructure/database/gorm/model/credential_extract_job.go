package model

import (
	"time"

	"CredChain_Golang/domain"
)

// CredentialExtractJob is the GORM model for the credential_extract_jobs
// Postgres-backed work queue. Workers poll for pending rows with
// FOR UPDATE SKIP LOCKED.
type CredentialExtractJob struct {
	Id           string     `gorm:"primaryKey;type:char(26);column:id"`
	CredentialId string     `gorm:"type:char(26);column:credential_id;index"`
	FileURI      string     `gorm:"type:text;column:file_uri;not null"`
	Status       string     `gorm:"type:credential_extract_job_status;column:status;not null;default:pending"`
	AttemptCount int        `gorm:"column:attempt_count;not null;default:0"`
	Errors       []string   `gorm:"type:text[];column:errors;default:'{}'"`
	AvailableAt  time.Time  `gorm:"column:available_at;not null;default:CURRENT_TIMESTAMP"`
	ReservedAt   *time.Time `gorm:"column:reserved_at"`
	CreatedAt    time.Time  `gorm:"autoCreateTime;column:created_at"`
}

func (CredentialExtractJob) TableName() string { return "credential_extract_jobs" }

// ToDomain converts a GORM credential_extract_jobs row to its domain entity.
func (m CredentialExtractJob) ToDomain() domain.CredentialExtractJob {
	errors := m.Errors
	if errors == nil {
		errors = []string{}
	}
	return domain.CredentialExtractJob{
		ID:           m.Id,
		CredentialID: m.CredentialId,
		FileURI:      m.FileURI,
		Status:       m.Status,
		AttemptCount: m.AttemptCount,
		Errors:       errors,
		AvailableAt:  m.AvailableAt,
		ReservedAt:   m.ReservedAt,
		CreatedAt:    m.CreatedAt,
	}
}

// FromDomainCredentialExtractJob converts a domain entity to a GORM model.
func FromDomainCredentialExtractJob(j domain.CredentialExtractJob) CredentialExtractJob {
	return CredentialExtractJob{
		Id:           j.ID,
		CredentialId: j.CredentialID,
		FileURI:      j.FileURI,
		Status:       j.Status,
		AttemptCount: j.AttemptCount,
		Errors:       j.Errors,
		AvailableAt:  j.AvailableAt,
		ReservedAt:   j.ReservedAt,
		CreatedAt:    j.CreatedAt,
	}
}

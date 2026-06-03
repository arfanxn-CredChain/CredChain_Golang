package gorm

import (
	"CredChain_Golang/domain"
	"context"
	"gorm.io/gorm"
)

// RepositoryFactory defines the signature for creating repositories from a GORM DB connection
type RepositoryFactory[T any] func(db *gorm.DB) T

// UserRepositoryFactory creates a UserRepository from a GORM DB connection
type UserRepositoryFactory = RepositoryFactory[domain.UserRepository]

// CredentialRepositoryFactory creates a CredentialRepository from a GORM DB connection
type CredentialRepositoryFactory = RepositoryFactory[domain.CredentialRepository]

// UserTokenRepositoryFactory creates a UserTokenRepository from a GORM DB connection
type UserTokenRepositoryFactory = RepositoryFactory[domain.UserTokenRepository]

// CredentialExtractJobRepositoryFactory creates a CredentialExtractJobRepository from a GORM DB connection
type CredentialExtractJobRepositoryFactory = RepositoryFactory[domain.CredentialExtractJobRepository]

// GormUnitOfWork implements domain.UnitOfWork using GORM transactions
type GormUnitOfWork struct {
	db                          *gorm.DB
	userRepository              domain.UserRepository
	credentialRepository        domain.CredentialRepository
	userTokenRepository         domain.UserTokenRepository
	credentialExtractJobRepo    domain.CredentialExtractJobRepository
	newUserRepository           UserRepositoryFactory
	newCredentialRepository     CredentialRepositoryFactory
	newUserTokenRepository      UserTokenRepositoryFactory
	newCredentialExtractJobRepo CredentialExtractJobRepositoryFactory
}

// NewGormUnitOfWork creates a new UnitOfWork instance with repository factories
func NewGormUnitOfWork(
	db *gorm.DB,
	newUserRepository UserRepositoryFactory,
	newCredentialRepository CredentialRepositoryFactory,
	newUserTokenRepository UserTokenRepositoryFactory,
	newCredentialExtractJobRepo CredentialExtractJobRepositoryFactory,
) domain.UnitOfWork {
	return &GormUnitOfWork{
		db:                          db,
		newUserRepository:           newUserRepository,
		newCredentialRepository:     newCredentialRepository,
		newUserTokenRepository:      newUserTokenRepository,
		newCredentialExtractJobRepo: newCredentialExtractJobRepo,
	}
}

// Execute runs a function within a transaction
// All repository operations within the function share the same transaction
func (uow *GormUnitOfWork) Execute(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
	return uow.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := uow.newUserRepository(tx)
		txCredRepo := uow.newCredentialRepository(tx)
		txTokenRepo := uow.newUserTokenRepository(tx)
		txCredExtractJobRepo := uow.newCredentialExtractJobRepo(tx)

		txUow := &GormUnitOfWork{
			db:                       tx,
			userRepository:           txUserRepo,
			credentialRepository:     txCredRepo,
			userTokenRepository:      txTokenRepo,
			credentialExtractJobRepo: txCredExtractJobRepo,
		}

		return fn(txUow)
	})
}

// User returns the UserRepository for this transaction
func (uow *GormUnitOfWork) User() domain.UserRepository {
	return uow.userRepository
}

// Credential returns the CredentialRepository for this transaction
func (uow *GormUnitOfWork) Credential() domain.CredentialRepository {
	return uow.credentialRepository
}

// UserToken returns the UserTokenRepository for this transaction
func (uow *GormUnitOfWork) UserToken() domain.UserTokenRepository {
	return uow.userTokenRepository
}

// CredentialExtractJob returns the CredentialExtractJobRepository for this transaction
func (uow *GormUnitOfWork) CredentialExtractJob() domain.CredentialExtractJobRepository {
	return uow.credentialExtractJobRepo
}

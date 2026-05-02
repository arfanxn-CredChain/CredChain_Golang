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

// GormUnitOfWork implements domain.UnitOfWork using GORM transactions
type GormUnitOfWork struct {
	db                      *gorm.DB
	userRepository          domain.UserRepository
	credentialRepository    domain.CredentialRepository
	userTokenRepository     domain.UserTokenRepository
	newUserRepository       UserRepositoryFactory
	newCredentialRepository CredentialRepositoryFactory
	newUserTokenRepository  UserTokenRepositoryFactory
}

// NewGormUnitOfWork creates a new UnitOfWork instance with repository factories
func NewGormUnitOfWork(
	db *GormDB,
	newUserRepository UserRepositoryFactory,
	newCredentialRepository CredentialRepositoryFactory,
	newUserTokenRepository UserTokenRepositoryFactory,
) domain.UnitOfWork {
	return &GormUnitOfWork{
		db:                      db.DB,
		newUserRepository:       newUserRepository,
		newCredentialRepository: newCredentialRepository,
		newUserTokenRepository:  newUserTokenRepository,
	}
}

// Execute runs a function within a transaction
// All repository operations within the function share the same transaction
func (uow *GormUnitOfWork) Execute(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
	return uow.db.Transaction(func(tx *gorm.DB) error {
		// Create NEW repositories with transaction DB using factories
		txUserRepo := uow.newUserRepository(tx)
		txCredRepo := uow.newCredentialRepository(tx)
		txTokenRepo := uow.newUserTokenRepository(tx)

		txUow := &GormUnitOfWork{
			db:                   tx,
			userRepository:       txUserRepo,
			credentialRepository: txCredRepo,
			userTokenRepository:  txTokenRepo,
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

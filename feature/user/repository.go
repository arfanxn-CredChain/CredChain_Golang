package user

import (
	"context"
	"strings"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database"

	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"
)

type PostgresUserRepository struct {
	db *database.DB
}

type UserRepoParams struct {
	fx.In
	DB *database.DB
}

func NewRepository(p UserRepoParams) domain.UserRepository {
	return &PostgresUserRepository{db: p.DB}
}

func (r *PostgresUserRepository) GetUsers(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	query := `SELECT id, name, number, phone_number, email, birth_date, meta, role, wallet_address, wallet_private_key, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT 50`
	err := r.db.SelectContext(ctx, &users, query)
	return users, err
}

func (r *PostgresUserRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	query := `SELECT id, name, number, phone_number, email, birth_date, meta, role, wallet_address, wallet_private_key, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	return &user, err
}

func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT id, name, number, phone_number, email, birth_date, meta, role, wallet_address, wallet_private_key, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	return &user, err
}

func (r *PostgresUserRepository) UpdateProfile(ctx context.Context, id string, name, number, phoneNumber *string, meta *domain.JSONB) (*domain.User, error) {
	query := `
		UPDATE users 
		SET name = COALESCE($1, name),
		    number = COALESCE($2, number),
		    phone_number = COALESCE($3, phone_number),
			meta = COALESCE($4, meta),
			updated_at = NOW()
		WHERE id = $5
		RETURNING id, name, number, phone_number, meta, updated_at, email, birth_date, role, wallet_address, wallet_private_key, created_at
	`
	var user domain.User
	err := r.db.QueryRowxContext(ctx, query, name, number, phoneNumber, meta, id).StructScan(&user)
	return &user, err
}

func (r *PostgresUserRepository) UpdateEmail(ctx context.Context, id string, email string) (string, error) {
	query := `UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2 RETURNING email`
	var newEmail string
	err := r.db.GetContext(ctx, &newEmail, query, strings.ToLower(email), id)
	return newEmail, err
}

func (r *PostgresUserRepository) GetUsersByIDs(ctx context.Context, ids []string) ([]domain.User, error) {
	if len(ids) == 0 {
		return []domain.User{}, nil
	}
	
	query, args, err := sqlx.In("SELECT id, name, number, phone_number, email, birth_date, meta, role, wallet_address, wallet_private_key, created_at, updated_at FROM users WHERE id IN (?)", ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var users []domain.User
	err = r.db.SelectContext(ctx, &users, query, args...)
	return users, err
}

func (r *PostgresUserRepository) BatchUpdateRole(ctx context.Context, updates []domain.UserRoleUpdate) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, update := range updates {
		_, err := tx.ExecContext(ctx, "UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2", update.Role, update.UserID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresUserRepository) BatchCreate(ctx context.Context, users []domain.User) ([]domain.User, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	insertQuery := `
		INSERT INTO users (id, name, email, role, wallet_address, wallet_private_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, email, role, wallet_address, wallet_private_key, created_at, updated_at
	`
	var createdUsers []domain.User
	for _, u := range users {
		var created domain.User
		err = tx.QueryRowxContext(ctx, insertQuery,
			u.ID, u.Name, u.Email, u.Role, u.WalletAddress, u.WalletPrivateKey,
		).StructScan(&created)
		if err != nil {
			return nil, err
		}
		createdUsers = append(createdUsers, created)
	}

	return createdUsers, tx.Commit()
}

package user

import (
	"context"
	"fmt"
	"strings"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
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

func (r *PostgresUserRepository) GetUsers(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	var users []domain.User
	var total int

	baseQuery := `FROM users`
	whereClause := ""
	var args []interface{}
	argId := 1

	if query == nil {
		countQuery := `SELECT COUNT(id) ` + baseQuery
		err := r.db.GetContext(ctx, &total, countQuery)
		if err != nil {
			return nil, 0, err
		}

		dataQuery := `SELECT id, name, number, phone_number, email, birth_date, meta, role, wallet_address, wallet_private_key, created_at, updated_at ` + baseQuery + ` ORDER BY created_at DESC`
		err = r.db.SelectContext(ctx, &users, dataQuery)
		return users, total, err
	}

	if query.HasSearch() {
		whereClause = fmt.Sprintf(` WHERE (name ILIKE $%d OR email ILIKE $%d)`, argId, argId+1)
		args = append(args, "%"+query.Search+"%", "%"+query.Search+"%")
		argId += 2
	}

	for _, filter := range query.Filters {
		if whereClause == "" {
			whereClause = " WHERE "
		} else {
			whereClause += " AND "
		}

		switch filter.Operator {
		case domainQuery.OperatorEqual:
			whereClause += fmt.Sprintf(`"%s" = $%d`, filter.Column, argId)
			args = append(args, filter.Values[0])
			argId++

		case domainQuery.OperatorNotEqual:
			whereClause += fmt.Sprintf(`"%s" != $%d`, filter.Column, argId)
			args = append(args, filter.Values[0])
			argId++

		case domainQuery.OperatorGreaterThan:
			whereClause += fmt.Sprintf(`"%s" > $%d`, filter.Column, argId)
			args = append(args, filter.Values[0])
			argId++

		case domainQuery.OperatorLessThan:
			whereClause += fmt.Sprintf(`"%s" < $%d`, filter.Column, argId)
			args = append(args, filter.Values[0])
			argId++

		case domainQuery.OperatorGreaterThanOrEqual:
			whereClause += fmt.Sprintf(`"%s" >= $%d`, filter.Column, argId)
			args = append(args, filter.Values[0])
			argId++

		case domainQuery.OperatorLessThanOrEqual:
			whereClause += fmt.Sprintf(`"%s" <= $%d`, filter.Column, argId)
			args = append(args, filter.Values[0])
			argId++

		case domainQuery.OperatorLike:
			value := filter.Values[0]
			if !strings.Contains(value, "%") {
				value = "%" + value + "%"
			}
			whereClause += fmt.Sprintf(`"%s" LIKE $%d`, filter.Column, argId)
			args = append(args, value)
			argId++

		case domainQuery.OperatorILike:
			value := filter.Values[0]
			if !strings.Contains(value, "%") {
				value = "%" + value + "%"
			}
			whereClause += fmt.Sprintf(`"%s" ILIKE $%d`, filter.Column, argId)
			args = append(args, value)
			argId++

		case domainQuery.OperatorNotLike:
			value := filter.Values[0]
			if !strings.Contains(value, "%") {
				value = "%" + value + "%"
			}
			whereClause += fmt.Sprintf(`"%s" NOT LIKE $%d`, filter.Column, argId)
			args = append(args, value)
			argId++

		case domainQuery.OperatorNotILike:
			value := filter.Values[0]
			if !strings.Contains(value, "%") {
				value = "%" + value + "%"
			}
			whereClause += fmt.Sprintf(`"%s" NOT ILIKE $%d`, filter.Column, argId)
			args = append(args, value)
			argId++

		case domainQuery.OperatorIn:
			placeholders := make([]string, len(filter.Values))
			for i, v := range filter.Values {
				placeholders[i] = fmt.Sprintf("$%d", argId+i)
				args = append(args, v)
			}
			whereClause += fmt.Sprintf(`"%s" IN (%s)`, filter.Column, strings.Join(placeholders, ", "))
			argId += len(filter.Values)

		case domainQuery.OperatorNotIn:
			placeholders := make([]string, len(filter.Values))
			for i, v := range filter.Values {
				placeholders[i] = fmt.Sprintf("$%d", argId+i)
				args = append(args, v)
			}
			whereClause += fmt.Sprintf(`"%s" NOT IN (%s)`, filter.Column, strings.Join(placeholders, ", "))
			argId += len(filter.Values)

		case domainQuery.OperatorBetween:
			whereClause += fmt.Sprintf(`"%s" BETWEEN $%d AND $%d`, filter.Column, argId, argId+1)
			args = append(args, filter.Values[0], filter.Values[1])
			argId += 2

		case domainQuery.OperatorNotBetween:
			whereClause += fmt.Sprintf(`"%s" NOT BETWEEN $%d AND $%d`, filter.Column, argId, argId+1)
			args = append(args, filter.Values[0], filter.Values[1])
			argId += 2

		case domainQuery.OperatorNull:
			whereClause += fmt.Sprintf(`"%s" IS NULL`, filter.Column)

		case domainQuery.OperatorNotNull:
			whereClause += fmt.Sprintf(`"%s" IS NOT NULL`, filter.Column)
		}
	}

	countQuery := `SELECT COUNT(id) ` + baseQuery + whereClause
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	orderClause := " ORDER BY created_at DESC"
	if query.HasSorts() {
		var orderParts []string
		for _, sort := range query.Sorts {
			if sort.Column == "name" || sort.Column == "created_at" {
				orderParts = append(orderParts, fmt.Sprintf(`"%s" %s`, sort.Column, sort.Order))
			}
		}
		if len(orderParts) > 0 {
			orderClause = " ORDER BY " + strings.Join(orderParts, ", ")
		}
	}

	limitClause := ""
	if query.HasPagination() {
		limitClause = fmt.Sprintf(` LIMIT %d OFFSET %d`, query.Limit, query.Offset())
	}

	dataQuery := `SELECT id, name, number, phone_number, email, birth_date, meta, role, wallet_address, wallet_private_key, created_at, updated_at ` + baseQuery + whereClause + orderClause + limitClause

	err = r.db.SelectContext(ctx, &users, dataQuery, args...)
	return users, total, err
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

func (r *PostgresUserRepository) BatchDeleteUsers(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query, args, err := sqlx.In("DELETE FROM users WHERE id IN (?)", ids)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

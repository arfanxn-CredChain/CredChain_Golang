package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type gormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) domain.UserRepository {
	return &gormUserRepository{db: db}
}

// Get retrieves users with pagination and search support (batch operation)
// Returns: ([]User, int, error) - paginated slice, total count matching search criteria, error
func (r *gormUserRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	db := r.db.WithContext(ctx).Model(&model.User{})

	// Search only (filters, includes, groups skipped for now)
	if query.HasSearch() {
		db = db.Where("LOWER(name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?)",
			"%"+query.Search+"%", "%"+query.Search+"%")
	}

	// Count total (respects search, excludes limit/offset)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Ordering - check query.Sorts for created_at or name
	if query.HasSorts() {
		for _, sort := range query.Sorts {
			switch sort.Column {
			case "created_at":
				db = db.Order(fmt.Sprintf("created_at %s", sort.Order))
			case "name":
				db = db.Order(fmt.Sprintf("name %s", sort.Order))
			}
		}
	} else {
		// Default ordering
		db = db.Order("created_at DESC")
	}

	// Pagination
	if query.HasPagination() {
		db = db.Limit(query.Limit).Offset(query.Offset())
	}

	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	// Map to domain
	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}

	return domainUsers, int(total), nil
}

// Find retrieves a single user by ID
// Returns: (*User, error) - single entity lookup
func (r *gormUserRepository) Find(ctx context.Context, id string) (*domain.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	domainUser := user.ToDomain()
	return &domainUser, nil
}

// FindByEmails retrieves users by multiple emails (batch operation)
// Returns: ([]User, error) - empty slice if no matches
func (r *gormUserRepository) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) {
	if len(emails) == 0 {
		return []domain.User{}, nil
	}
	var users []model.User
	if err := r.db.WithContext(ctx).Where("email IN ?", emails).Find(&users).Error; err != nil {
		return nil, err
	}
	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}
	return domainUsers, nil
}

// FindByRole retrieves all users with the specified role
// Returns: ([]User, error) - empty slice if no matches
func (r *gormUserRepository) FindByRole(ctx context.Context, role domain.Role) ([]domain.User, error) {
	var users []model.User
	if err := r.db.WithContext(ctx).Where("role = ?", role).Find(&users).Error; err != nil {
		return nil, err
	}
	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}
	return domainUsers, nil
}

// FindByIds retrieves multiple users by IDs (batch operation)
// Returns: ([]User, error) - batch lookup, empty slice if no matches
func (r *gormUserRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	if len(ids) == 0 {
		return []domain.User{}, nil
	}

	var users []model.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}

	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}

	return domainUsers, nil
}

// Update updates one or more users (partial update via non-zero fields)
// Returns: ([]User, error) - updated users, error
func (r *gormUserRepository) Update(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	if len(users) == 0 {
		return []domain.User{}, nil
	}
	if err := r.updateBatchCase(ctx, users); err != nil {
		return nil, err
	}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}
	return r.FindByIds(ctx, ids...)
}

// updateBatchCase builds and executes a single UPDATE statement using CASE expressions
// for each column that at least one user provides. Users that don't provide a column
// fall through to ELSE column (preserving the existing value). Eliminates the N+1
// per-user UPDATE pattern. Users sorted by Id for deterministic arg ordering.
func (r *gormUserRepository) updateBatchCase(ctx context.Context, users []domain.User) error {
	sort.Slice(users, func(i, j int) bool { return users[i].Id < users[j].Id })

	var setClauses []string
	var setArgs []interface{}

	addCaseClause := func(col string, getValue func(domain.User) (interface{}, bool)) {
		var caseArgs []interface{}
		for _, u := range users {
			if v, ok := getValue(u); ok {
				caseArgs = append(caseArgs, u.Id, v)
			}
		}
		if len(caseArgs) == 0 {
			return
		}
		caseSQL := "CASE id"
		for i := 0; i < len(caseArgs)/2; i++ {
			caseSQL += " WHEN ? THEN ?"
		}
		caseSQL += " ELSE " + col + " END"
		setClauses = append(setClauses, col+" = "+caseSQL)
		setArgs = append(setArgs, caseArgs...)
	}

	addCaseClause("name", func(u domain.User) (interface{}, bool) {
		if u.Name != nil {
			return *u.Name, true
		}
		return nil, false
	})
	addCaseClause("number", func(u domain.User) (interface{}, bool) {
		if u.Number != nil {
			return *u.Number, true
		}
		return nil, false
	})
	addCaseClause("phone_number", func(u domain.User) (interface{}, bool) {
		if u.PhoneNumber != nil {
			return *u.PhoneNumber, true
		}
		return nil, false
	})
	addCaseClause("email", func(u domain.User) (interface{}, bool) {
		if u.Email != "" {
			return u.Email, true
		}
		return nil, false
	})
	addCaseClause("birth_date", func(u domain.User) (interface{}, bool) {
		if u.BirthDate != nil {
			return *u.BirthDate, true
		}
		return nil, false
	})
	addCaseClause("meta", func(u domain.User) (interface{}, bool) {
		if u.Meta == nil {
			return nil, false
		}
		b, err := json.Marshal(u.Meta)
		if err != nil {
			return nil, false
		}
		return string(b), true
	})
	addCaseClause("role", func(u domain.User) (interface{}, bool) {
		if u.Role != "" {
			return string(u.Role), true
		}
		return nil, false
	})
	addCaseClause("wallet_address", func(u domain.User) (interface{}, bool) {
		if u.WalletAddress != "" {
			return u.WalletAddress, true
		}
		return nil, false
	})
	addCaseClause("encrypted_wallet_private_key", func(u domain.User) (interface{}, bool) {
		if u.EncryptedWalletPrivateKey != "" {
			return u.EncryptedWalletPrivateKey, true
		}
		return nil, false
	})

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	ids := make([]interface{}, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}

	sql := "UPDATE users SET " + strings.Join(setClauses, ", ") + " WHERE id IN (?)"
	finalArgs := append(setArgs, ids)
	return r.db.WithContext(ctx).Exec(sql, finalArgs...).Error
}

// UpdateRole batch updates roles for multiple users using efficient CASE statement
// Returns: ([]User, int64, error) - updated users slice, rows affected count, error
func (r *gormUserRepository) UpdateRole(ctx context.Context, users ...domain.User) ([]domain.User, int64, error) {
	if len(users) == 0 {
		return []domain.User{}, 0, nil
	}

	db := r.db.WithContext(ctx)

	// Efficient batch update using CASE statement
	caseStmt := "CASE id "
	args := make([]interface{}, 0, len(users)*2)

	for _, user := range users {
		caseStmt += "WHEN ? THEN ? "
		args = append(args, user.Id, user.Role)
	}
	caseStmt += "ELSE role END"

	ids := make([]interface{}, len(users))
	for i, user := range users {
		ids[i] = user.Id
	}

	query := "UPDATE users SET role = " + caseStmt + " WHERE id IN (?)"
	finalArgs := append(args, ids)

	result := db.Exec(query, finalArgs...)
	if err := result.Error; err != nil {
		return nil, 0, err
	}

	// Fetch updated users
	userIDs := make([]string, len(users))
	for i, user := range users {
		userIDs[i] = user.Id
	}
	updatedUsers, err := r.FindByIds(ctx, userIDs...)
	if err != nil {
		return nil, 0, err
	}

	return updatedUsers, result.RowsAffected, nil
}

// Store persists multiple users (batch operation)
// Returns: ([]User, error) - created users with generated IDs, error
func (r *gormUserRepository) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	// Generate ULID for users without ID
	for i := range users {
		if users[i].Id == "" {
			users[i].Id = ulid.Make().String()
		}
	}

	// Convert to model and batch INSERT
	modelUsers := make([]model.User, len(users))
	for i, u := range users {
		modelUsers[i] = model.FromDomainUser(u)
	}

	if err := r.db.WithContext(ctx).Create(&modelUsers).Error; err != nil {
		// Translate Postgres unique violation (23505) to domain code.
		// Catches concurrent batch creates with same email that raced past
		// the pre-check in userService.storeValidateEmails.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			emails := make([]string, len(users))
			for i, u := range users {
				emails[i] = u.Email
			}
			return nil, domain.NewError(
				domain.CodeUserStoreEmailDuplicateInDatabase,
				domain.WithMetadata("emails", emails),
				domain.WithError(err),
			)
		}
		return nil, err
	}

	// Convert back to domain and return
	created := make([]domain.User, len(modelUsers))
	for i, m := range modelUsers {
		created[i] = m.ToDomain()
	}
	return created, nil
}

// Delete deletes multiple users by IDs (batch operation)
// Returns: (int64, error) - rows affected count, error
func (r *gormUserRepository) Delete(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).Delete(&model.User{}, "id IN ?", ids)
	return result.RowsAffected, result.Error
}

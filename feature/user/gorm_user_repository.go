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
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type gormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) domain.UserRepository {
	return &gormUserRepository{db: db}
}

// allowedFilterColumns whitelists user columns clients may filter on.
// Excludes wallet_address, encrypted_wallet_private_key, and meta
// to prevent leaking secrets, bypassing soft delete, or running expensive
// JSONB predicates from untrusted input.
// deleted_at is intentionally included to enable trashed-user pagination
// via filters/sorts on the deleted_at column.
var allowedFilterColumns = map[string]bool{
	"id":           true,
	"name":         true,
	"email":        true,
	"role":         true,
	"number":       true,
	"phone_number": true,
	"birth_date":   true,
	"gender":       true,
	"created_at":   true,
	"updated_at":   true,
	"deleted_at":   true,
}

// allowedSortColumns whitelists user columns clients may sort on.
// deleted_at is intentionally included to enable sorting trashed users
// by their deletion timestamp.
var allowedSortColumns = map[string]bool{
	"id":         true,
	"name":       true,
	"email":      true,
	"role":       true,
	"gender":     true,
	"created_at": true,
	"updated_at": true,
	"deleted_at": true,
}

// referencesDeletedAt reports whether any filter or sort touches the
// deleted_at column. When true, Get bypasses GORM's soft-delete scope
// so trashed users can be listed and ordered by their deletion timestamp.
func referencesDeletedAt(q *domainQuery.Query) bool {
	return lo.ContainsBy(q.Filters, func(f domainQuery.Filter) bool { return f.Column == "deleted_at" }) ||
		lo.ContainsBy(q.Sorts, func(s domainQuery.Sort) bool { return s.Column == "deleted_at" })
}

// Get retrieves users with pagination, search, filters, and sorts (batch operation)
// Returns: ([]User, int, error) - paginated slice, total count matching criteria, error
func (r *gormUserRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	db := r.db.WithContext(ctx).Model(&model.User{})
	if referencesDeletedAt(query) {
		db = db.Unscoped()
	}

	if query.HasSearch() {
		db = db.Where("LOWER(name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?)",
			"%"+query.Search+"%", "%"+query.Search+"%")
	}

	if query.HasFilters() {
		db = applyUserFilters(db, query.Filters)
	}

	// Count total (respects search + filters, excludes limit/offset)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if query.HasSorts() {
		for _, s := range query.Sorts {
			if allowedSortColumns[s.Column] {
				db = db.Order(fmt.Sprintf("%s %s", s.Column, s.Order))
			}
		}
	} else {
		db = db.Order("created_at DESC")
	}

	if query.HasPagination() {
		db = db.Limit(query.Limit).Offset(query.Offset())
	}

	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}

	return domainUsers, int(total), nil
}

// applyUserFilters maps domainQuery.Filter operators to GORM Where clauses.
// Filters on columns not present in allowedFilterColumns are silently dropped
// (validator/parser already accepted them; here we enforce a column-level
// allowlist to prevent SQL injection via the column field). Pattern operators
// use LOWER(col) LIKE LOWER(?) for dialect-agnostic case-insensitivity
// (Postgres + SQLite).
func applyUserFilters(db *gorm.DB, filters []domainQuery.Filter) *gorm.DB {
	for _, f := range filters {
		if !allowedFilterColumns[f.Column] {
			continue
		}
		col := f.Column
		switch f.Operator {
		case domainQuery.OperatorEqual:
			db = db.Where(col+" = ?", f.GetValue())
		case domainQuery.OperatorNotEqual:
			db = db.Where(col+" != ?", f.GetValue())
		case domainQuery.OperatorGreaterThan:
			db = db.Where(col+" > ?", f.GetValue())
		case domainQuery.OperatorLessThan:
			db = db.Where(col+" < ?", f.GetValue())
		case domainQuery.OperatorGreaterThanOrEqual:
			db = db.Where(col+" >= ?", f.GetValue())
		case domainQuery.OperatorLessThanOrEqual:
			db = db.Where(col+" <= ?", f.GetValue())
		case domainQuery.OperatorLike, domainQuery.OperatorILike:
			db = db.Where("LOWER("+col+") LIKE LOWER(?)", "%"+f.GetValue()+"%")
		case domainQuery.OperatorNotLike, domainQuery.OperatorNotILike:
			db = db.Where("LOWER("+col+") NOT LIKE LOWER(?)", "%"+f.GetValue()+"%")
		case domainQuery.OperatorIn:
			if len(f.Values) > 0 {
				db = db.Where(col+" IN ?", f.Values)
			}
		case domainQuery.OperatorNotIn:
			if len(f.Values) > 0 {
				db = db.Where(col+" NOT IN ?", f.Values)
			}
		case domainQuery.OperatorBetween:
			if len(f.Values) == 2 {
				db = db.Where(col+" BETWEEN ? AND ?", f.Values[0], f.Values[1])
			}
		case domainQuery.OperatorNotBetween:
			if len(f.Values) == 2 {
				db = db.Where(col+" NOT BETWEEN ? AND ?", f.Values[0], f.Values[1])
			}
		case domainQuery.OperatorNull:
			db = db.Where(col + " IS NULL")
		case domainQuery.OperatorNotNull:
			db = db.Where(col + " IS NOT NULL")
		}
	}
	return db
}

// Find retrieves a single user by ID, including soft-deleted users.
// Returns: (*User, error) - single entity lookup
func (r *gormUserRepository) Find(ctx context.Context, id string) (*domain.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Unscoped().First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	domainUser := user.ToDomain()
	return &domainUser, nil
}

// FindByEmails retrieves users by multiple emails, including soft-deleted users (batch operation)
// Returns: ([]User, error) - empty slice if no matches
func (r *gormUserRepository) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) {
	if len(emails) == 0 {
		return []domain.User{}, nil
	}
	var users []model.User
	if err := r.db.WithContext(ctx).Unscoped().Where("email IN ?", emails).Find(&users).Error; err != nil {
		return nil, err
	}
	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}
	return domainUsers, nil
}

// FindByRole retrieves all users with the specified role, including soft-deleted users
// Returns: ([]User, error) - empty slice if no matches
func (r *gormUserRepository) FindByRole(ctx context.Context, role domain.Role) ([]domain.User, error) {
	var users []model.User
	if err := r.db.WithContext(ctx).Unscoped().Where("role = ?", role).Find(&users).Error; err != nil {
		return nil, err
	}
	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}
	return domainUsers, nil
}

// FindByIds retrieves multiple users by IDs, including soft-deleted users (batch operation)
// Returns: ([]User, error) - batch lookup, empty slice if no matches
func (r *gormUserRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	if len(ids) == 0 {
		return []domain.User{}, nil
	}

	var users []model.User
	if err := r.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&users).Error; err != nil {
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
	addCaseClause("gender", func(u domain.User) (interface{}, bool) {
		if u.Gender != nil {
			return string(*u.Gender), true
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

package user

import (
	"context"
	"fmt"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	gormInfra "CredChain_Golang/infrastructure/gorm"
	"CredChain_Golang/infrastructure/gorm/model"
	"github.com/oklog/ulid/v2"
)

type GormUserRepository struct {
	db *gormInfra.GormDB
}

func NewGormUserRepository(db *gormInfra.GormDB) domain.UserRepository {
	return &GormUserRepository{db: db}
}

// Get retrieves users with pagination and search support (batch operation)
// Returns: ([]User, int, error) - paginated slice, total count matching search criteria, error
func (r *GormUserRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	db := r.db.WithContext(ctx).Model(&model.User{})

	// Search only (filters, includes, groups skipped for now)
	if query.HasSearch() {
		db = db.Where("name ILIKE ? OR email ILIKE ?",
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
func (r *GormUserRepository) Find(ctx context.Context, id string) (*domain.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	domainUser := user.ToDomain()
	return &domainUser, nil
}

// FindByEmails retrieves users by multiple emails (batch operation)
// Returns: ([]User, error) - empty slice if no matches
func (r *GormUserRepository) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) {
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

// FindByIds retrieves multiple users by IDs (batch operation)
// Returns: ([]User, error) - batch lookup, empty slice if no matches
func (r *GormUserRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
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

// Update updates a single user (accepts domain model)
// Returns: (*User, error) - updated entity
func (r *GormUserRepository) Update(ctx context.Context, user domain.User) (*domain.User, error) {
	modelUser := model.FromDomainUser(user)

	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.Id).Updates(modelUser).Error; err != nil {
		return nil, err
	}

	// Fetch updated user
	return r.Find(ctx, user.Id)
}

// UpdateRole batch updates roles for multiple users using efficient CASE statement
// Returns: ([]User, int64, error) - updated users slice, rows affected count, error
func (r *GormUserRepository) UpdateRole(ctx context.Context, users ...domain.User) ([]domain.User, int64, error) {
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
	caseStmt += "END"

	ids := make([]interface{}, len(users))
	for i, user := range users {
		ids[i] = user.Id
	}

	query := "UPDATE users SET role = " + caseStmt + " WHERE id IN (?)"
	finalArgs := append(args, ids...)

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
func (r *GormUserRepository) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
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
		return nil, err
	}

	// Convert back to domain and return
	created := make([]domain.User, len(modelUsers))
	for i, m := range modelUsers {
		created[i] = m.ToDomain()
	}
	return created, nil
}

// Destroy deletes multiple users by IDs (batch operation)
// Returns: (int64, error) - rows affected count, error
func (r *GormUserRepository) Destroy(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).Delete(&model.User{}, "id IN ?", ids)
	return result.RowsAffected, result.Error
}

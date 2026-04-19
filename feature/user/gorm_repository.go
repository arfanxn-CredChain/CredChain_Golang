package user

import (
	"context"
	"fmt"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	gormInfra "CredChain_Golang/infrastructure/gorm"
	"CredChain_Golang/infrastructure/gorm/model"

	"go.uber.org/fx"
	"gorm.io/gorm"
)

type GormUserRepository struct {
	db *gormInfra.GormDB
}

type GormUserRepositoryParams struct {
	fx.In
	DB *gormInfra.GormDB
}

func NewGormUserRepository(p GormUserRepositoryParams) domain.UserRepository {
	return &GormUserRepository{db: p.DB}
}

// Get implements query-based retrieval (Search + optional sorts for now)
func (r *GormUserRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	db := r.db.WithContext(ctx).Model(&model.User{})

	// Search only (filters, includes, groups skipped for now)
	if query.HasSearch() {
		db = db.Where("name ILIKE ? OR email ILIKE ?",
			"%"+query.Search+"%", "%"+query.Search+"%")
	}

	// Count total
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Ordering - check query.Sorts for created_at or name
	if query.HasSorts() {
		for _, sort := range query.Sorts {
			if sort.Column == "created_at" {
				db = db.Order(fmt.Sprintf("created_at %s", sort.Order))
			} else if sort.Column == "name" {
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

// Find retrieves single user by ID
func (r *GormUserRepository) Find(ctx context.Context, id string) (*domain.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	domainUser := user.ToDomain()
	return &domainUser, nil
}

// FindByEmail retrieves single user by email
func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	domainUser := user.ToDomain()
	return &domainUser, nil
}

// FindByIds retrieves multiple users by IDs
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

// Update updates a user (accepts domain model)
func (r *GormUserRepository) Update(ctx context.Context, user domain.User) (*domain.User, error) {
	modelUser := model.FromDomainUser(user)

	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.Id).Updates(modelUser).Error; err != nil {
		return nil, err
	}

	// Fetch updated user
	return r.Find(ctx, user.Id)
}

// UpdateRole updates roles for multiple users
func (r *GormUserRepository) UpdateRole(ctx context.Context, updates []domain.UserRoleUpdate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			if err := tx.Model(&model.User{}).Where("id = ?", update.UserID).Update("role", update.Role).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Store stores multiple users
func (r *GormUserRepository) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	return nil, nil // TODO: Implement later
}

// Destroy deletes multiple users by IDs
func (r *GormUserRepository) Destroy(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Delete(&model.User{}, "id IN ?", ids).Error
	})
}

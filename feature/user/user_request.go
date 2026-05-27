package user

import (
	"time"

	"CredChain_Golang/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type UserStoreInput struct {
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	Role        domain.Role    `json:"role"`
	Number      *string        `json:"number"`
	PhoneNumber *string        `json:"phone_number"`
	BirthDate   *string        `json:"birth_date"`
	Meta        map[string]any `json:"meta"`
}

func (n UserStoreInput) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.Name, validation.Required),
		validation.Field(&n.Email, validation.Required, is.Email),
		validation.Field(&n.Role, validation.Required, validation.In(domain.RoleAdmin, domain.RoleIssuer, domain.RoleHolder)),
		validation.Field(&n.BirthDate, validation.Date("2006-01-02")),
	)
}

func (n UserStoreInput) ToDomain() domain.User {
	var birthDate *time.Time
	if n.BirthDate != nil && *n.BirthDate != "" {
		if t, err := time.Parse("2006-01-02", *n.BirthDate); err == nil {
			birthDate = &t
		}
	}
	return domain.User{
		Name:        &n.Name,
		Email:       n.Email,
		Role:        n.Role,
		Number:      n.Number,
		PhoneNumber: n.PhoneNumber,
		BirthDate:   birthDate,
		Meta:        n.Meta,
	}
}

type UserStoreRequest struct {
	Users []UserStoreInput `json:"users"`
}

func (r UserStoreRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Users, validation.Required, validation.Each(validation.By(func(value any) error {
			u := value.(UserStoreInput)
			return u.Validate()
		}))),
	)
}

func (r UserStoreRequest) ToDomain() []domain.User {
	if r.Users == nil {
		return []domain.User{}
	}
	users := make([]domain.User, len(r.Users))
	for i, u := range r.Users {
		users[i] = u.ToDomain()
	}
	return users
}

type UserUpdateProfileRequest struct {
	Name        *string       `json:"name"`
	Number      *string       `json:"number"`
	PhoneNumber *string       `json:"phone_number"`
	Meta        map[string]any `json:"meta"`
}

func (r UserUpdateProfileRequest) Validate() error { return nil }

type UserUpdateEmailRequest struct {
	Email string `json:"email"`
}

func (r UserUpdateEmailRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email, validation.Required, is.Email),
	)
}

type UserRoleUpdateRequest struct {
	UserID string      `json:"user_id"`
	Role   domain.Role `json:"role"`
}

func (r UserRoleUpdateRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserID, validation.Required),
		validation.Field(&r.Role, validation.Required, validation.In(domain.RoleSuperAdmin, domain.RoleAdmin, domain.RoleIssuer, domain.RoleHolder)),
	)
}

type UserBatchUpdateRoleRequest struct {
	UserRoles []UserRoleUpdateRequest `json:"user_roles"`
}

func (r UserBatchUpdateRoleRequest) Validate() error {
	return validation.ValidateStruct(&r, validation.Field(&r.UserRoles, validation.Required))
}

type UserBatchDeleteRequest struct {
	UserIDs []string `json:"user_ids"`
}

func (r UserBatchDeleteRequest) Validate() error {
	return validation.ValidateStruct(&r, validation.Field(&r.UserIDs, validation.Required))
}

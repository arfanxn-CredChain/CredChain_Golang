package user

import (
	"CredChain_Golang/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type CreateUserRequest struct {
	Name  string      `json:"name"`
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
}

func (n CreateUserRequest) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.Name, validation.Required.Error("validation_required")),
		validation.Field(&n.Email, validation.Required.Error("validation_required"), is.Email.Error("validation_is_email")),
		validation.Field(&n.Role, validation.Required.Error("validation_required"), validation.In(domain.RoleSuperAdmin, domain.RoleAdmin, domain.RoleIssuer, domain.RoleHolder).Error("validation_invalid_role")),
	)
}

type BatchCreateUsersRequest struct {
	Users []CreateUserRequest `json:"users"`
}

func (r BatchCreateUsersRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Users, validation.Required.Error("validation_required"), validation.Each(validation.By(func(value any) error {
			u := value.(CreateUserRequest)
			return u.Validate()
		}))),
	)
}

type UpdateProfileRequest struct {
	Name        *string       `json:"name"`
	Number      *string       `json:"number"`
	PhoneNumber *string       `json:"phone_number"`
	Meta        *domain.JSONB `json:"meta"`
}

func (r UpdateProfileRequest) Validate() error {
	return nil
}

type UpdateEmailRequest struct {
	Email string `json:"email"`
}

func (r UpdateEmailRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email, validation.Required.Error("validation_required"), is.Email.Error("validation_is_email")),
	)
}

type UserRoleUpdateRequest struct {
	UserID string      `json:"user_id"`
	Role   domain.Role `json:"role"`
}

func (r UserRoleUpdateRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserID, validation.Required.Error("validation_required")),
		validation.Field(&r.Role, validation.Required.Error("validation_required"), validation.In(domain.RoleSuperAdmin, domain.RoleAdmin, domain.RoleIssuer, domain.RoleHolder).Error("validation_invalid_role")),
	)
}

type BatchUpdateRoleRequest struct {
	UserRoles []UserRoleUpdateRequest `json:"user_roles"`
}

func (r BatchUpdateRoleRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserRoles, validation.Required.Error("validation_required")),
	)
}

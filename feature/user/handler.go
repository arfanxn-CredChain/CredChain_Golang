package user

import (
	"errors"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	queryRequest "CredChain_Golang/infrastructure/http/request/query"
	"CredChain_Golang/infrastructure/http/responder"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Handler struct {
	userSvc  UserService
	credRepo domain.CredentialRepository
}

type UserHandlerParams struct {
	fx.In
	UserSvc  UserService
	CredRepo domain.CredentialRepository
}

func NewUserHandler(p UserHandlerParams) *Handler {
	return &Handler{userSvc: p.UserSvc, credRepo: p.CredRepo}
}

// ... Logic ...

func (h *Handler) Paginate(c *gin.Context) {
	var req queryRequest.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	query, err := req.ToDomain()
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	users, total, err := h.userSvc.Paginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.SendPaginated(c, domain.CodeUserFetchSuccess, users, total)
}

func (h *Handler) Find(c *gin.Context) {
	claims := httpContext.MustGetUserClaims(c.Request.Context())
	user, err := h.userSvc.Find(c.Request.Context(), claims.Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserFetchSuccess, user)
}

func (h *Handler) GetSelfCredentials(c *gin.Context) {
	claims := httpContext.MustGetUserClaims(c.Request.Context())
	creds, err := h.credRepo.FindByHolder(c.Request.Context(), claims.Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if creds == nil {
		creds = []domain.Credential{}
	}
	responder.Send(c, domain.CodeUserCredentialsFetchSuccess, creds)
}

func (h *Handler) FindByAdmin(c *gin.Context) {
	id := c.Param("id")
	user, err := h.userSvc.Find(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserFetchSuccess, user)
}

func (h *Handler) BatchCreateUsers(c *gin.Context) {
	var req BatchCreateUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	domainUsers := make([]domain.User, len(req.Users))
	for i, u := range req.Users {
		domainUsers[i] = domain.User{
			Name:  &u.Name,
			Email: u.Email,
			Role:  u.Role,
		}
	}
	created, err := h.userSvc.Store(c.Request.Context(), domainUsers...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserCreateSuccess, created)
}

func (h *Handler) UpdateSelfProfile(c *gin.Context) {
	claims := httpContext.MustGetUserClaims(c.Request.Context())
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	user, err := h.userSvc.UpdateProfile(c.Request.Context(), claims.Id, req.Name, req.Number, req.PhoneNumber, req.Meta)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserProfileSuccess, user)
}

func (h *Handler) UpdateSelfEmail(c *gin.Context) {
	claims := httpContext.MustGetUserClaims(c.Request.Context())
	var req UpdateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	newEmail, err := h.userSvc.UpdateEmail(c.Request.Context(), claims.Id, req.Email)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserEmailSuccess, gin.H{"email": newEmail})
}

func (h *Handler) BatchUpdateRole(c *gin.Context) {
	var req BatchUpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	var domainUpdates []domain.UserRoleUpdate
	for _, u := range req.UserRoles {
		if err := u.Validate(); err != nil {
			responder.SendValidationError(c, err)
			return
		}
		domainUpdates = append(domainUpdates, domain.UserRoleUpdate{
			UserID: u.UserID,
			Role:   u.Role,
		})
	}

	err := h.userSvc.UpdateRole(c.Request.Context(), domainUpdates...)
	if err != nil {
		c.Error(err)
		var appErr *domain.Error
		if errors.As(err, &appErr) {
			switch appErr.Code {
			case domain.CodeUserRoleAdminUpdatePeerForbidden, domain.CodeUserRoleSignerAdminRequiredForbidden:
				responder.SendError(c, err)
				return
			}
		}
		responder.SendError(c, err)
		return
	}
	responder.Send[any](c, domain.CodeUserRoleSuccess, nil)
}

func (h *Handler) BatchDeleteUsers(c *gin.Context) {
	var req BatchDeleteUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if validationErr := req.Validate(); validationErr != nil {
		responder.SendValidationError(c, validationErr)
		return
	}

	err := h.userSvc.Destroy(c.Request.Context(), req.UserIDs...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send[any](c, domain.CodeUserBatchDeleteSuccess, nil)
}

package user

import (
	"errors"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	queryRequest "CredChain_Golang/infrastructure/http/request/query"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type UserHandler interface {
	Paginate(c *gin.Context)
	Find(c *gin.Context)
	GetSelfCredentials(c *gin.Context)
	FindByAdmin(c *gin.Context)
	Store(c *gin.Context)
	UpdateSelfProfile(c *gin.Context)
	UpdateSelfEmail(c *gin.Context)
	BatchUpdateRole(c *gin.Context)
	BatchDeleteUsers(c *gin.Context)
}

type userHandler struct {
	userSvc  UserService
	credRepo domain.CredentialRepository
}

type UserHandlerParams struct {
	fx.In
	UserSvc  UserService
	CredRepo domain.CredentialRepository
}

func NewUserHandler(p UserHandlerParams) UserHandler {
	return &userHandler{userSvc: p.UserSvc, credRepo: p.CredRepo}
}

func (h *userHandler) Paginate(c *gin.Context) {
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
	responseUsers := make([]response.User, len(users))
	for i, u := range users {
		responseUsers[i] = response.FromDomainUser(u)
	}
	responder.SendPagination(c, domain.CodeUserFetchSuccess, responseUsers, total)
}

func (h *userHandler) Find(c *gin.Context) {
	authUser := httpContext.MustGetUser(c.Request.Context())
	user, err := h.userSvc.Find(c.Request.Context(), authUser.Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserFetchSuccess, response.FromDomainUser(*user))
}

func (h *userHandler) GetSelfCredentials(c *gin.Context) {
	authUser := httpContext.MustGetUser(c.Request.Context())
	creds, err := h.credRepo.FindByHolder(c.Request.Context(), authUser.Id)
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

func (h *userHandler) FindByAdmin(c *gin.Context) {
	id := c.Param("id")
	user, err := h.userSvc.Find(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserFetchSuccess, response.FromDomainUser(*user))
}

func (h *userHandler) Store(c *gin.Context) {
	var req UserStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	domainUsers := req.ToDomain()
	created, err := h.userSvc.Store(c.Request.Context(), domainUsers...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	users := make([]response.User, len(created))
	for i, u := range created {
		users[i] = response.FromDomainUser(u)
	}
	responder.Send(c, domain.CodeUserStoreSuccess, users)
}

func (h *userHandler) UpdateSelfProfile(c *gin.Context) {
	authUser := httpContext.MustGetUser(c.Request.Context())
	var req UserUpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	user, err := h.userSvc.UpdateProfile(c.Request.Context(), authUser.Id, req.Name, req.Number, req.PhoneNumber, req.Meta)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserProfileSuccess, response.FromDomainUser(*user))
}

func (h *userHandler) UpdateSelfEmail(c *gin.Context) {
	authUser := httpContext.MustGetUser(c.Request.Context())
	var req UserUpdateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	newEmail, err := h.userSvc.UpdateEmail(c.Request.Context(), authUser.Id, req.Email)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserEmailSuccess, gin.H{"email": newEmail})
}

func (h *userHandler) BatchUpdateRole(c *gin.Context) {
	var req UserBatchUpdateRoleRequest
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
		domainUpdates = append(domainUpdates, domain.UserRoleUpdate{UserID: u.UserID, Role: u.Role})
	}
	updatedUsers, _, err := h.userSvc.UpdateRole(c.Request.Context(), domainUpdates...)
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
	responseUsers := make([]response.User, len(updatedUsers))
	for i, u := range updatedUsers {
		responseUsers[i] = response.FromDomainUser(u)
	}
	responder.Send(c, domain.CodeUserRoleSuccess, responseUsers)
}

func (h *userHandler) BatchDeleteUsers(c *gin.Context) {
	var req UserBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if validationErr := req.Validate(); validationErr != nil {
		responder.SendValidationError(c, validationErr)
		return
	}
	deletedCount, err := h.userSvc.Destroy(c.Request.Context(), req.UserIDs...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserBatchDeleteSuccess, gin.H{"deleted_count": deletedCount})
}

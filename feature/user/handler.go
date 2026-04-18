package user

import (
	"fmt"
	"strings"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/middleware"
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

func NewHandler(p UserHandlerParams) *Handler {
	return &Handler{userSvc: p.UserSvc, credRepo: p.CredRepo}
}

// ... Logic ...

func (h *Handler) GetUsers(c *gin.Context) {
	var req queryRequest.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(fmt.Errorf("GetUsers: bind failed: %w", err))
		responder.SendError(c, domain.CodeSystemValidation)
		return
	}

	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	query, err := req.ToDomain()
	if err != nil {
		c.Error(fmt.Errorf("GetUsers: ToDomain failed: %w", err))
		responder.SendError(c, domain.CodeSystemValidation)
		return
	}

	users, total, err := h.userSvc.GetUsers(c.Request.Context(), query)
	if err != nil {
		c.Error(fmt.Errorf("GetUsers: %w", err))
		responder.SendError(c, domain.CodeSystemInternal)
		return
	}

	responder.SendPaginated(c, domain.CodeUserFetchSuccess, users, total, *query)
}

func (h *Handler) GetSelf(c *gin.Context) {
	claims := middleware.GetUserClaims(c)
	if claims == nil {
		responder.SendError(c, domain.CodeAuthLoginUnauthorized)
		return
	}
	user, err := h.userSvc.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		c.Error(fmt.Errorf("GetSelf: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeUserFetchNotFound)
		return
	}
	responder.Send(c, domain.CodeUserFetchSuccess, user)
}

func (h *Handler) GetSelfCredentials(c *gin.Context) {
	claims := middleware.GetUserClaims(c)
	if claims == nil {
		responder.SendError(c, domain.CodeAuthLoginUnauthorized)
		return
	}
	creds, err := h.credRepo.GetCredentialsByHolder(c.Request.Context(), claims.UserID)
	if err != nil {
		c.Error(fmt.Errorf("GetSelfCredentials: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeUserCredsFetchFailed)
		return
	}
	if creds == nil {
		creds = []domain.Credential{}
	}
	responder.Send(c, domain.CodeUserCredsFetchSuccess, creds)
}

func (h *Handler) GetUserByID(c *gin.Context) {
	id := c.Param("id")
	user, err := h.userSvc.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.Error(fmt.Errorf("GetUserByID: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeUserFetchNotFound)
		return
	}
	responder.Send(c, domain.CodeUserFetchSuccess, user)
}

func (h *Handler) BatchCreateUsers(c *gin.Context) {
	var req BatchCreateUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(fmt.Errorf("BatchCreateUsers: bind failed: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeSystemValidation)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	created, err := h.userSvc.CreateUsers(c.Request.Context(), req.Users)
	if err != nil {
		c.Error(fmt.Errorf("BatchCreateUsers: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeUserCreateFailed)
		return
	}
	responder.Send(c, domain.CodeUserCreateSuccess, created)
}

func (h *Handler) UpdateSelfProfile(c *gin.Context) {
	claims := middleware.GetUserClaims(c)
	if claims == nil {
		responder.SendError(c, domain.CodeAuthLoginUnauthorized)
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(fmt.Errorf("UpdateSelfProfile: bind failed: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeSystemValidation)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	user, err := h.userSvc.UpdateProfile(c.Request.Context(), claims.UserID, req.Name, req.Number, req.PhoneNumber, req.Meta)
	if err != nil {
		c.Error(fmt.Errorf("UpdateSelfProfile: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeUserProfileFailed)
		return
	}
	responder.Send(c, domain.CodeUserProfileSuccess, user)
}

func (h *Handler) UpdateSelfEmail(c *gin.Context) {
	claims := middleware.GetUserClaims(c)
	if claims == nil {
		responder.SendError(c, domain.CodeAuthLoginUnauthorized)
		return
	}
	var req UpdateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(fmt.Errorf("UpdateSelfEmail: bind failed: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeSystemValidation)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	newEmail, err := h.userSvc.UpdateEmail(c.Request.Context(), claims.UserID, req.Email)
	if err != nil {
		c.Error(fmt.Errorf("UpdateSelfEmail: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeUserEmailConflict)
		return
	}
	responder.Send(c, domain.CodeUserEmailSuccess, gin.H{"email": newEmail})
}

func (h *Handler) BatchUpdateRole(c *gin.Context) {
	claims := middleware.GetUserClaims(c)
	if claims == nil {
		responder.SendError(c, domain.CodeAuthLoginUnauthorized)
		return
	}

	var req BatchUpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(fmt.Errorf("BatchUpdateRole: bind failed: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeSystemValidation)
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

	err := h.userSvc.BatchUpdateRole(c.Request.Context(), claims.UserID, domainUpdates)
	if err != nil {
		c.Error(fmt.Errorf("BatchUpdateRole: %w", err)) //nolint:errcheck
		if strings.Contains(err.Error(), fmt.Sprintf("%d", domain.CodeUserRoleAdminUpdatePeerForbidden)) {
			responder.SendError(c, domain.CodeUserRoleAdminUpdatePeerForbidden)
			return
		}
		if strings.Contains(err.Error(), fmt.Sprintf("%d", domain.CodeUserRoleSignerAdminRequiredForbidden)) {
			responder.SendError(c, domain.CodeUserRoleSignerAdminRequiredForbidden)
			return
		}
		responder.SendError(c, domain.CodeUserRoleFailed)
		return
	}
	responder.Send(c, domain.CodeUserRoleSuccess, nil)
}

func (h *Handler) BatchDeleteUsers(c *gin.Context) {
	claims := middleware.GetUserClaims(c)
	if claims == nil {
		responder.SendError(c, domain.CodeAuthLoginUnauthorized)
		return
	}

	var req BatchDeleteUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(fmt.Errorf("BatchDeleteUsers: bind failed: %w", err))
		responder.SendError(c, domain.CodeSystemValidation)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	err := h.userSvc.BatchDeleteUsers(c.Request.Context(), claims.UserID, req.UserIDs)
	if err != nil {
		c.Error(fmt.Errorf("BatchDeleteUsers: %w", err))
		responder.SendError(c, domain.CodeUserBatchDeleteFailed)
		return
	}

	responder.Send(c, domain.CodeUserBatchDeleteSuccess, nil)
}

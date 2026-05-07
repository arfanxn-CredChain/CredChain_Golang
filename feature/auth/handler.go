package auth

import (
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"
	httpContext "CredChain_Golang/infrastructure/http/context"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// Handler handles all auth-related HTTP routes
type Handler struct {
	service *Service
	config  *config.Config
}

type AuthHandlerParams struct {
	fx.In
	Service *Service
	Config  *config.Config
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(p AuthHandlerParams) *Handler {
	return &Handler{service: p.Service, config: p.Config}
}

// sendAuthResponse sends a standardized auth response with user data and tokens.
func (h *Handler) sendAuthResponse(c *gin.Context, code int, user domain.User, refreshToken domain.UserToken, accessToken string) {
	refreshExpirySec := h.config.JWTRefreshExpiryHours * int(time.Hour.Seconds())
	accessExpirySec := h.config.JWTAccessExpiryMinutes * int(time.Minute.Seconds())
	responder.Send(c, code, response.NewAuth(
		response.FromDomainUser(user),
		accessToken,
		refreshToken.Token,
		accessExpirySec,
		refreshExpirySec,
	))
}

// GoogleLogin processes Google OAuth login and returns access + refresh tokens
func (h *Handler) GoogleLogin(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	user, refreshToken, accessToken, err := h.service.GoogleLogin(c.Request.Context(), req.IdToken)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	h.sendAuthResponse(c, domain.CodeAuthGoogleLoginSuccess, user, refreshToken, accessToken)
}

// GoogleRefresh validates refresh token and issues new token pair
func (h *Handler) GoogleRefresh(c *gin.Context) {
	var req GoogleRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	user, refreshToken, accessToken, err := h.service.GoogleRefresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	h.sendAuthResponse(c, domain.CodeAuthGoogleRefreshSuccess, user, refreshToken, accessToken)
}

// Logout revokes all refresh tokens for the authenticated user
func (h *Handler) Logout(c *gin.Context) {
	authUser := httpContext.MustGetUser(c.Request.Context())

	err := h.service.Logout(c.Request.Context(), authUser.Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send[any](c, domain.CodeAuthLogoutSuccess, nil)
}

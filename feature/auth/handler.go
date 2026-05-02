package auth

import (
	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"

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

	responder.Send(c, domain.CodeAuthGoogleLoginSuccess, response.GoogleLogin{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    h.config.JWTAccessExpiryMinutes * 60,
		Role:         string(user.Role),
	})
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

	_, refreshToken, accessToken, err := h.service.GoogleRefresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send(c, domain.CodeAuthGoogleRefreshSuccess, response.GoogleRefresh{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    h.config.JWTAccessExpiryMinutes * 60,
	})
}

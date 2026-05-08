package auth

import (
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type AuthHandler interface {
	GoogleLogin(c *gin.Context)
	Refresh(c *gin.Context)
	Logout(c *gin.Context)
}

type authHandler struct {
	service AuthService
	config  *config.Config
}

type AuthHandlerParams struct {
	fx.In
	Service AuthService
	Config  *config.Config
}

func NewAuthHandler(p AuthHandlerParams) AuthHandler {
	return &authHandler{service: p.Service, config: p.Config}
}

func (h *authHandler) sendAuthResponse(c *gin.Context, code int, user domain.User, refreshToken domain.UserToken, accessToken string) {
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

func (h *authHandler) GoogleLogin(c *gin.Context) {
	var req AuthGoogleLoginRequest
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

func (h *authHandler) Refresh(c *gin.Context) {
	var req AuthRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	user, refreshToken, accessToken, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	h.sendAuthResponse(c, domain.CodeAuthRefreshSuccess, user, refreshToken, accessToken)
}

func (h *authHandler) Logout(c *gin.Context) {
	authUser := httpContext.MustGetUser(c.Request.Context())

	err := h.service.Logout(c.Request.Context(), authUser.Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send[any](c, domain.CodeAuthLogoutSuccess, nil)
}

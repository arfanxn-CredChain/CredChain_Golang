package auth

import (
	"context"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/security"

	"github.com/gin-gonic/gin"
	validationozzo "github.com/go-ozzo/ozzo-validation/v4"
	"go.uber.org/fx"
	"google.golang.org/api/idtoken"
)

// Handler handles all auth-related HTTP routes
type Handler struct {
	userRepo  domain.UserRepository
	jwtSecret string
}

type AuthHandlerParams struct {
	fx.In
	UserRepo domain.UserRepository
	Config   *config.Config
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(p AuthHandlerParams) *Handler {
	return &Handler{userRepo: p.UserRepo, jwtSecret: p.Config.JWTSecret}
}

// AuthGoogleRequest represents the incoming JSON payload
type AuthGoogleRequest struct {
	IDToken string `json:"id_token"`
}

// Validate performs structural validation
func (r AuthGoogleRequest) Validate() error {
	return validationozzo.ValidateStruct(&r,
		validationozzo.Field(&r.IDToken, validationozzo.Required),
	)
}

// AuthGoogleResponse represents the response containing the issued JWT
type AuthGoogleResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
}

// HandleGoogleLogin processes Google id_token and authenticates user
func (h *Handler) HandleGoogleLogin(c *gin.Context) {
	var req AuthGoogleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	ctx := context.Background()
	payload, err := idtoken.Validate(ctx, req.IDToken, "")
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		c.Error(domain.NewError(domain.CodeAuthLoginInvalidToken))
		responder.SendError(c, domain.NewError(domain.CodeAuthLoginInvalidToken))
		return
	}

	user, err := h.userRepo.FindByEmail(ctx, email)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	if h.jwtSecret == "" {
		c.Error(domain.NewError(domain.CodeSystemInternal))
		responder.SendError(c, domain.NewError(domain.CodeSystemInternal))
		return
	}

	token, err := security.GenerateJWT([]byte(h.jwtSecret), user.Id, user.Email, user.WalletAddress, string(user.Role))
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send(c, domain.CodeAuthLoginSuccess, AuthGoogleResponse{
		Token: token,
		Role:  string(user.Role),
	})
}

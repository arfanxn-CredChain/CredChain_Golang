package auth

import (
	"context"
	"fmt"

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

// NewHandler creates a new AuthHandler
func NewHandler(p AuthHandlerParams) *Handler {
	return &Handler{userRepo: p.UserRepo, jwtSecret: p.Config.JWTSecret}
}

// AuthGoogleRequest represents the incoming JSON payload
type AuthGoogleRequest struct {
	IDToken string `json:"id_token"`
}

// Validate performs structural validation
func (r AuthGoogleRequest) Validate() error {
	return validationozzo.ValidateStruct(&r,
		validationozzo.Field(&r.IDToken, validationozzo.Required.Error("validation_required")),
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
		c.Error(fmt.Errorf("HandleGoogleLogin: bind failed: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeSystemValidation)
		return
	}

	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	// 1. Verify Google token signature and extract email
	ctx := context.Background()
	payload, err := idtoken.Validate(ctx, req.IDToken, "")
	if err != nil {
		c.Error(fmt.Errorf("HandleGoogleLogin: validation failed: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeAuthLoginInvalidToken)
		return
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		c.Error(fmt.Errorf("HandleGoogleLogin: missing email")) //nolint:errcheck
		responder.SendError(c, domain.CodeAuthLoginInvalidToken)
		return
	}

	// 2. Lookup user via UserRepository
	user, err := h.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		c.Error(fmt.Errorf("HandleGoogleLogin: not found: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeAuthLoginForbidden)
		return
	}

	// 3. Issue our backend JWT
	if h.jwtSecret == "" {
		c.Error(fmt.Errorf("HandleGoogleLogin: no secret")) //nolint:errcheck
		responder.SendError(c, domain.CodeSystemInternal)
		return
	}

	token, err := security.GenerateJWT([]byte(h.jwtSecret), user.ID, user.Email, user.WalletAddress, string(user.Role))
	if err != nil {
		c.Error(fmt.Errorf("HandleGoogleLogin: JWT gen failed: %w", err)) //nolint:errcheck
		responder.SendError(c, domain.CodeAuthLoginJWTFailed)
		return
	}

	responder.Send(c, domain.CodeAuthLoginSuccess, AuthGoogleResponse{
		Token: token,
		Role:  string(user.Role),
	})
}

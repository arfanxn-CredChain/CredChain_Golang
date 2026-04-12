package credential

import (
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/responder"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Handler struct {
	credSvc *Service
}

type CredHandlerParams struct {
	fx.In
	CredSvc *Service
}

func NewHandler(p CredHandlerParams) *Handler {
	return &Handler{credSvc: p.CredSvc}
}

func (h *Handler) GetCredentials(c *gin.Context) {
	responder.Send(c, domain.CodeCredFetchSuccess, gin.H{"status": "stub"})
}

func (h *Handler) GetCredentialByID(c *gin.Context) {
	responder.Send(c, domain.CodeCredFetchSuccess, gin.H{"status": "stub"})
}

func (h *Handler) IssueCredential(c *gin.Context) {
	responder.Send(c, domain.CodeCredIssueSuccess, gin.H{"status": "stub"})
}

func (h *Handler) RevokeCredential(c *gin.Context) {
	responder.Send(c, domain.CodeCredRevokeSuccess, gin.H{"status": "stub"})
}

func (h *Handler) VerifyHash(c *gin.Context) {
	responder.Send(c, domain.CodeCredVerifySuccess, gin.H{"status": "stub"})
}

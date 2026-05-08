package credential

import (
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/responder"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type CredentialHandler interface {
	GetCredentials(c *gin.Context)
	GetCredentialByID(c *gin.Context)
	IssueCredential(c *gin.Context)
	RevokeCredential(c *gin.Context)
	VerifyHash(c *gin.Context)
}

type credentialHandler struct {
	credSvc CredentialService
}

type CredentialHandlerParams struct {
	fx.In
	CredSvc CredentialService
}

func NewCredentialHandler(p CredentialHandlerParams) CredentialHandler {
	return &credentialHandler{credSvc: p.CredSvc}
}

func (h *credentialHandler) GetCredentials(c *gin.Context) {
	responder.Send(c, domain.CodeCredentialFetchSuccess, gin.H{"status": "stub"})
}
func (h *credentialHandler) GetCredentialByID(c *gin.Context) {
	responder.Send(c, domain.CodeCredentialFetchSuccess, gin.H{"status": "stub"})
}
func (h *credentialHandler) IssueCredential(c *gin.Context) {
	responder.Send(c, domain.CodeCredentialIssueSuccess, gin.H{"status": "stub"})
}
func (h *credentialHandler) RevokeCredential(c *gin.Context) {
	responder.Send(c, domain.CodeCredentialRevokeSuccess, gin.H{"status": "stub"})
}
func (h *credentialHandler) VerifyHash(c *gin.Context) {
	responder.Send(c, domain.CodeCredentialVerifySuccess, gin.H{"status": "stub"})
}

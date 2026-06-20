package credential

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	pyai "CredChain_Golang/infrastructure/ai/pyai"
	queryRequest "CredChain_Golang/infrastructure/http/request/query"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

// ── Interface ─────────────────────────────────────────────────────────────

// CredentialHandler is the HTTP layer for credential operations. Method names
// follow the user feature pattern: Paginate, Find, Issue, Revoke, Verify.
// SelfPaginate / SelfFind serve the Holder dashboard at /api/users/self/credentials.
type CredentialHandler interface {
	Paginate(c *gin.Context)
	Find(c *gin.Context)
	Issue(c *gin.Context)
	Revoke(c *gin.Context)
	Verify(c *gin.Context)
	ReExtract(c *gin.Context)
	SelfPaginate(c *gin.Context)
	SelfFind(c *gin.Context)
	DownloadFile(c *gin.Context)
}

// ── Implementation & constructor ──────────────────────────────────────────

type credentialHandler struct {
	credSvc CredentialService
}

type CredentialHandlerParams struct {
	fx.In
	CredSvc CredentialService
}

// NewCredentialHandler is the exported factory for FX injection.
func NewCredentialHandler(p CredentialHandlerParams) CredentialHandler {
	return &credentialHandler{credSvc: p.CredSvc}
}

// ── Paginate ──────────────────────────────────────────────────────────────

// Paginate returns a paginated credential list with search, filters, sorts,
// and optional holder/issuer/revoker user expansions via the includes query
// parameter (e.g. ?includes=holder,issuer).
func (h *credentialHandler) Paginate(c *gin.Context) {
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
	credentials, total, err := h.credSvc.Paginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(credentials)
	responder.SendPagination(c, domain.CodeCredentialFetchSuccess, out, total)
}

// ── Find ──────────────────────────────────────────────────────────────────

// Find returns a single credential by ID with optional user expansions.
// The includes query parameter controls which relations (holder, issuer,
// revoker) are preloaded by the repository.
func (h *credentialHandler) Find(c *gin.Context) {
	id := c.Param("id")

	var req queryRequest.QueryRequest
	c.ShouldBindQuery(&req)
	query, _ := req.ToDomain()
	if query == nil {
		query = &domainQuery.Query{}
	}

	cred, err := h.credSvc.Find(c.Request.Context(), id, query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse([]domain.Credential{*cred})
	if len(out) == 0 {
		responder.Send(c, domain.CodeCredentialFetchSuccess, gin.H{})
		return
	}
	responder.Send(c, domain.CodeCredentialFetchSuccess, out[0])
}

// ── Self (Holder) ─────────────────────────────────────────────────────────

// SelfPaginate returns a paginated list of credentials owned by the
// authenticated user (holder_user_id == auth_user.id). Same query DSL as
// Paginate; the holder filter is injected by the service.
func (h *credentialHandler) SelfPaginate(c *gin.Context) {
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
	credentials, total, err := h.credSvc.SelfPaginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(credentials)
	responder.SendPagination(c, domain.CodeCredentialFetchSuccess, out, total)
}

// SelfFind returns one credential by ID, scoped to the authenticated user.
// Returns 404 (CodeCredentialFetchNotFound) when the credential does not
// exist OR when it exists but is owned by another user — never leaks which
// IDs exist.
func (h *credentialHandler) SelfFind(c *gin.Context) {
	id := c.Param("id")

	var req queryRequest.QueryRequest
	c.ShouldBindQuery(&req)
	query, _ := req.ToDomain()
	if query == nil {
		query = &domainQuery.Query{}
	}

	cred, err := h.credSvc.SelfFind(c.Request.Context(), id, query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse([]domain.Credential{*cred})
	if len(out) == 0 {
		responder.Send(c, domain.CodeCredentialFetchSuccess, gin.H{})
		return
	}
	responder.Send(c, domain.CodeCredentialFetchSuccess, out[0])
}

// ── Issue ─────────────────────────────────────────────────────────────────

// Issue parses a multipart form into batch credential issue items and
// delegates to the service layer.
//
// Expected form structure (one set per item, zero-indexed):
//
//	items[0][holder_user_id]
//	items[0][name]
//	items[0][meta]            (JSON string, optional)
//	items[0][file]            (binary upload)
//	items[1][...]
func (h *credentialHandler) Issue(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	items, err := buildIssueItems(form)
	if err != nil {
		c.Error(err)
		responder.SendValidationError(c, err)
		return
	}

	req := CredentialIssueRequest{Items: items}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	serviceItems := make([]CredentialIssuance, len(items))
	for i, it := range items {
		fileBytes, mime, filename, err := readUploadedFile(it.File)
		if err != nil {
			c.Error(err)
			responder.SendError(c, err)
			return
		}
		if !allowedMIMETypes[mime] {
			responder.SendError(c, domain.NewError(domain.CodeCredentialIssueValidation,
				domain.WithMetadata("file_mime", mime)))
			return
		}
		if int64(len(fileBytes)) > maxFileBytes {
			responder.SendError(c, domain.NewError(domain.CodeCredentialIssueValidation,
				domain.WithMetadata("file_size", len(fileBytes))))
			return
		}
		serviceItems[i] = CredentialIssuance{
			HolderUserID: it.HolderUserID,
			Name:         it.Name,
			Meta:         it.Meta,
			Filename:     filename,
			MIMEType:     mime,
			FileBytes:    fileBytes,
		}
	}

	created, fieldErrs, err := h.credSvc.Issue(c.Request.Context(), serviceItems)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := make([]*response.Credential, len(created))
	successCount := 0
	for i, cred := range created {
		if cred.ID == "" {
			continue
		}
		dto := response.FromDomainCredential(cred)
		out[i] = &dto
		successCount++
	}
	code := domain.CodeCredentialIssueSuccess
	if successCount == 0 {
		code = domain.CodeCredentialIssueFailed
	}
	responder.SendPartial(c, code, out, fieldErrs)
}

// ── Revoke ────────────────────────────────────────────────────────────────

// Revoke batch-revokes credentials by ID (JSON body).
func (h *credentialHandler) Revoke(c *gin.Context) {
	var req CredentialRevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	revoked, err := h.credSvc.Revoke(c.Request.Context(), req.Ids...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(revoked)
	responder.Send(c, domain.CodeCredentialRevokeSuccess, out)
}

// ── Verify ────────────────────────────────────────────────────────────────

// Verify accepts a multipart file upload and returns a similarity verdict
// against all known credentials (cache → exact hash → fuzzy pipeline).
func (h *credentialHandler) Verify(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		responder.SendError(c, domain.NewError(domain.CodeCredentialVerifyValidation,
			domain.WithError(err)))
		return
	}
	fileBytes, mime, filename, err := readUploadedFile(fileHeader)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if !allowedMIMETypes[mime] {
		responder.SendError(c, domain.NewError(domain.CodeCredentialVerifyValidation,
			domain.WithMetadata("file_mime", mime)))
		return
	}
	if int64(len(fileBytes)) > maxFileBytes {
		responder.SendError(c, domain.NewError(domain.CodeCredentialVerifyValidation,
			domain.WithMetadata("file_size", len(fileBytes))))
		return
	}
	code, cred, score, percent, err := h.credSvc.Verify(c.Request.Context(), pyai.ExtractFile{
		Filename: filename,
		MIMEType: mime,
		Data:     fileBytes,
	})
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	desc := responder.ResolveMessage(c, code)
	out := response.CredentialVerify{
		VerdictCode:       code,
		SimilarityScore:   score,
		SimilarityPercent: percent,
		Description:       desc,
	}
	if cred != nil {
		dto := response.FromDomainCredential(*cred)
		out.Credential = &dto
	}
	responder.Send(c, code, out)
}

// ── ReExtract ─────────────────────────────────────────────────────────────

// ReExtract resets failed credential extractions to pending and enqueues new
// extract jobs via the River worker.
func (h *credentialHandler) ReExtract(c *gin.Context) {
	var req CredentialReExtractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	updated, err := h.credSvc.ReExtract(c.Request.Context(), req.Ids...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCredentialReExtractSuccess, mapCredentialsToResponse(updated))
}

// ── DownloadFile ──────────────────────────────────────────────────────────

// DownloadFile returns a single credential file with correct Content-Type for
// browser preview. Uses domain.CodeCredentialFileDownloadSuccess (200) on
// success. Authorization via policy (holder owns OR Issuer+).
func (h *credentialHandler) DownloadFile(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		responder.SendError(c, domain.NewError(domain.CodeCredentialFileDownloadNotFound))
		return
	}
	data, filename, mimeType, err := h.credSvc.DownloadFile(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	c.Header("Content-Type", mimeType)
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filepath.Base(filename)))
	c.Data(200, mimeType, data)
}

// ── Helpers ───────────────────────────────────────────────────────────────

// mapCredentialsToResponse converts domain credentials to response DTOs.
// Preloaded user relations (Holder, Issuer, Revoker) are mapped directly
// from the domain entity — no separate user lookup needed.
func mapCredentialsToResponse(credentials []domain.Credential) []response.Credential {
	out := make([]response.Credential, len(credentials))
	for i, c := range credentials {
		out[i] = response.FromDomainCredential(c)
	}
	return out
}

// readUploadedFile reads a multipart upload into memory and returns
// (bytes, MIME type, filename, error). MIME is taken from the multipart
// header; falls back to extension-based detection.
func readUploadedFile(fh *multipart.FileHeader) ([]byte, string, string, error) {
	src, err := fh.Open()
	if err != nil {
		return nil, "", "", err
	}
	defer src.Close()
	buf, err := io.ReadAll(src)
	if err != nil {
		return nil, "", "", err
	}
	mimeType := fh.Header.Get("Content-Type")
	if mimeType == "" {
		if t := mime.TypeByExtension(strings.ToLower(filepath.Ext(fh.Filename))); t != "" {
			mimeType = t
		} else {
			mimeType = "application/octet-stream"
		}
	}
	return buf, mimeType, fh.Filename, nil
}

// ── Multipart parsing ─────────────────────────────────────────────────────

// buildIssueItems extracts CredentialIssueInput slices from a parsed multipart
// form. Keys follow the pattern:
//
//	items[N][holder_user_id] = "ulid"
//	items[N][name] = "Bachelor's Degree"
//	items[N][meta] = `{"institution":"UI"}`
//	items[N][file] = <binary>
func buildIssueItems(form *multipart.Form) ([]CredentialIssueInput, error) {
	values := form.Value
	files := form.File

	idxSet := make(map[int]bool)
	for k := range values {
		if idx, ok := parseItemIndex(k); ok {
			idxSet[idx] = true
		}
	}
	for k := range files {
		if idx, ok := parseItemIndex(k); ok {
			idxSet[idx] = true
		}
	}
	if len(idxSet) == 0 {
		return nil, nil
	}

	maxIdx := lo.Max(lo.Keys(idxSet))

	items := make([]CredentialIssueInput, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		key := "items[" + strconv.Itoa(i) + "][holder_user_id]"
		if v, ok := values[key]; ok && len(v) > 0 {
			items[i].HolderUserID = v[0]
		}
		key = "items[" + strconv.Itoa(i) + "][name]"
		if v, ok := values[key]; ok && len(v) > 0 {
			items[i].Name = v[0]
		}
		key = "items[" + strconv.Itoa(i) + "][meta]"
		if v, ok := values[key]; ok && len(v) > 0 && v[0] != "" {
			var m map[string]any
			if err := json.Unmarshal([]byte(v[0]), &m); err == nil {
				items[i].Meta = m
			}
		}
		key = "items[" + strconv.Itoa(i) + "][file]"
		if fh, ok := files[key]; ok && len(fh) > 0 {
			items[i].File = fh[0]
		}
	}

	out := lo.Filter(items, func(it CredentialIssueInput, _ int) bool {
		return it.HolderUserID != "" || it.Name != "" || it.File != nil
	})
	return out, nil
}

func parseItemIndex(key string) (int, bool) {
	if !strings.HasPrefix(key, "items[") {
		return 0, false
	}
	closeBracket := strings.Index(key, "]")
	if closeBracket == -1 {
		return 0, false
	}
	idxStr := key[6:closeBracket]
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return 0, false
	}
	return idx, true
}

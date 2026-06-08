package credential

import (
	"mime/multipart"

	"CredChain_Golang/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var allowedMIMETypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"image/tiff":      true,
}

const maxFileBytes = 10 * 1024 * 1024 // 10 MB

// CredentialIssueInput is one item in a batch issue request.
type CredentialIssueInput struct {
	HolderUserID string         `form:"holder_user_id"`
	Name         string         `form:"name"`
	Meta         map[string]any `form:"meta"`
	File         *multipart.FileHeader
}

func (n CredentialIssueInput) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.HolderUserID, validation.Required),
		validation.Field(&n.Name, validation.Required, validation.Length(1, 256)),
	)
}

func (n CredentialIssueInput) ToDomain() domain.Credential {
	return domain.Credential{
		HolderUserID: n.HolderUserID,
		Name:         n.Name,
		Meta:         n.Meta,
	}
}

// CredentialIssueRequest is the parsed multipart batch issue request.
// Gin does not support nested multipart structs, so the handler builds this
// manually from c.MultipartForm().
type CredentialIssueRequest struct {
	Items []CredentialIssueInput
}

func (r CredentialIssueRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Items,
			validation.Required,
			validation.Length(1, 100),
			validation.Each(validation.By(func(v any) error {
				return v.(CredentialIssueInput).Validate()
			})),
		),
	)
}

func (r CredentialIssueRequest) ToDomain() []domain.Credential {
	out := make([]domain.Credential, len(r.Items))
	for i, item := range r.Items {
		out[i] = item.ToDomain()
	}
	return out
}

// CredentialRevokeRequest is the JSON body for POST /api/credentials/batch/revoke.
type CredentialRevokeRequest struct {
	Ids []string `json:"ids"`
}

func (r CredentialRevokeRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Ids,
			validation.Required,
			validation.Length(1, 100),
		),
	)
}

// CredentialReExtractRequest is the JSON body for POST /api/credentials/batch/reextract.
type CredentialReExtractRequest struct {
	Ids []string `json:"ids"`
}

func (r CredentialReExtractRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Ids, validation.Required, validation.Length(1, 100)),
	)
}

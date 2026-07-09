package responder

import (
	"encoding/json"
	"errors"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/tests/gintest"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/stretchr/testify/assert"
)

type respEnvelope struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    map[string]any      `json:"data"`
	Errors  map[string][]string `json:"errors"`
}

func TestSend_EnvelopeAndStatus(t *testing.T) {
	bundle := gintest.LoadTestI18nBundle(t)
	c, w := gintest.NewContext(t, gintest.WithI18nBundle(bundle))

	Send(c, domain.CodeSystemSuccess, map[string]any{"hello": "world"})

	assert.Equal(t, 200, w.Code)
	var got respEnvelope
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, domain.CodeSystemSuccess, got.Code)
	assert.NotEmpty(t, got.Message)
}

func TestSendError_DomainError_LocalizedMessage(t *testing.T) {
	bundle := gintest.LoadTestI18nBundle(t)
	c, w := gintest.NewContext(t, gintest.WithI18nBundle(bundle), gintest.WithAcceptLanguage("en"))

	err := domain.NewError(domain.CodeUserStoreEmailDuplicateInBatch,
		domain.WithMetadata("emails", []string{"a@x.com", "b@x.com"}))
	SendError(c, err)

	assert.Equal(t, 400, w.Code)
	var got respEnvelope
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, domain.CodeUserStoreEmailDuplicateInBatch, got.Code)
	assert.Contains(t, got.Message, "a@x.com")
}

func TestSendError_GenericError_FallsThroughToInternal(t *testing.T) {
	bundle := gintest.LoadTestI18nBundle(t)
	c, w := gintest.NewContext(t, gintest.WithI18nBundle(bundle))

	SendError(c, errors.New("plain old error"))

	assert.Equal(t, 500, w.Code)
	var got respEnvelope
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, domain.CodeSystemInternal, got.Code)
}

func TestSendValidationError(t *testing.T) {
	bundle := gintest.LoadTestI18nBundle(t)
	c, w := gintest.NewContext(t, gintest.WithI18nBundle(bundle), gintest.WithAcceptLanguage("en"))

	type body struct{ Email string }
	b := body{Email: "not-an-email"}
	verr := validation.ValidateStruct(&b,
		validation.Field(&b.Email, validation.Required, is.Email),
	)
	assert.Error(t, verr)

	SendValidationError(c, verr)

	assert.Equal(t, 400, w.Code)
	var got respEnvelope
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, domain.CodeSystemValidation, got.Code)
	assert.NotEmpty(t, got.Errors)
}

func TestSendValidationError_NonOzzoError(t *testing.T) {
	bundle := gintest.LoadTestI18nBundle(t)
	c, w := gintest.NewContext(t, gintest.WithI18nBundle(bundle))

	SendValidationError(c, errors.New("not an ozzo error"))
	assert.Equal(t, 500, w.Code)
}

func TestSendPagination_Envelope(t *testing.T) {
	bundle := gintest.LoadTestI18nBundle(t)
	c, w := gintest.NewContext(t,
		gintest.WithI18nBundle(bundle),
		gintest.WithPath("/api/users"),
		gintest.WithQueryString("page=1&limit=10"),
	)

	SendPagination(c, domain.CodeUserFetchSuccess, []map[string]any{{"id": "1"}}, 1)

	assert.Equal(t, 200, w.Code)
	var got map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(1), data["total"])
	assert.NotNil(t, data["items"])
}

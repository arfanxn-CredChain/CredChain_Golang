package responder

import (
	"fmt"
	"strings"

	"CredChain_Golang/domain"
	appI18n "CredChain_Golang/infrastructure/i18n"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Send writes a unified success or informational response.
func Send(c *gin.Context, code int, data any) {
	localizer := appI18n.GetLocalizer(c)
	msgKey := getMessageKey(c, code)
	message := localize(c, localizer, msgKey, nil)

	c.JSON(HttpCodeFromCode(code), domain.Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// SendError writes a unified error response and aborts the request.
func SendError(c *gin.Context, code int) {
	localizer := appI18n.GetLocalizer(c)
	msgKey := getMessageKey(c, code)
	message := localize(c, localizer, msgKey, nil)

	c.JSON(HttpCodeFromCode(code), domain.Response{
		Code:    code,
		Message: message,
	})
	c.Abort()
}

// SendValidationError handles ozzo-validation.Errors specifically.
func SendValidationError(c *gin.Context, err error) {
	localizer := appI18n.GetLocalizer(c)

	vErrs, ok := err.(validation.Errors)
	if !ok {
		c.Error(fmt.Errorf("responder.SendValidationError: received non-validation error: %w", err)) //nolint:errcheck
		SendError(c, domain.CodeSystemInternal)
		return
	}

	errors := make(map[string][]string)
	for path, fieldErr := range vErrs {
		parts := strings.Split(path, ".")
		leaf := parts[len(parts)-1]

		label := leaf
		if translated := localize(c, localizer, "labels."+leaf, nil); translated != "" {
			label = translated
		}

		msgID := fieldErr.Error()
		msg := localize(c, localizer, msgID, map[string]any{"field": label})
		if msg == "" {
			msg = msgID
		}
		msg = strings.ReplaceAll(msg, "{field}", label)

		errors[path] = append(errors[path], msg)
	}

	msgKey := getMessageKey(c, domain.CodeSystemValidation)
	message := localize(c, localizer, msgKey, nil)

	c.JSON(HttpCodeFromCode(domain.CodeSystemValidation), domain.Response{
		Code:    domain.CodeSystemValidation,
		Message: message,
		Errors:  errors,
	})
	c.Abort()
}

func getMessageKey(c *gin.Context, code int) string {
	msgKey, keyOk := domain.MessageKeys[code]
	if !keyOk {
		c.Error(fmt.Errorf("responder: unregistered code %d", code)) //nolint:errcheck
		return fmt.Sprintf("unregistered_code_%d", code)
	}
	return msgKey
}

func localize(c *gin.Context, localizer *i18n.Localizer, msgID string, data map[string]any) string {
	if msgID == "" {
		return ""
	}
	if localizer == nil {
		c.Error(fmt.Errorf("responder.localize: i18n localizer is nil (msgID=%q)", msgID)) //nolint:errcheck
		return msgID
	}

	text, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: data,
	})
	if err != nil {
		c.Error(fmt.Errorf("responder.localize: i18n miss msgID=%q: %w", msgID, err)) //nolint:errcheck
		return msgID
	}
	return text
}

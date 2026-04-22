package responder

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"CredChain_Golang/domain"
	appI18n "CredChain_Golang/infrastructure/i18n"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Send writes a unified success or informational response.
func Send[T any](c *gin.Context, code int, data T) {
	localizer := appI18n.GetLocalizer(c)
	msgKey := getMessageKey(c, code)
	message := localize(c, localizer, msgKey, nil)

	c.JSON(HttpCodeFromCode(code), domain.Response[T]{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// SendError writes a unified error response and aborts the request.
func SendError(c *gin.Context, err error) {
	localizer := appI18n.GetLocalizer(c)

	var appErr *domain.Error
	if errors.As(err, &appErr) {
		msgKey := getMessageKey(c, appErr.Code)
		message := localize(c, localizer, msgKey, appErr.Metadata)
		c.JSON(HttpCodeFromCode(appErr.Code), domain.Response[any]{
			Code:    appErr.Code,
			Message: message,
		})
		c.Abort()
		return
	}

	msgKey := getMessageKey(c, domain.CodeSystemInternal)
	message := localize(c, localizer, msgKey, nil)
	c.JSON(http.StatusInternalServerError, domain.Response[any]{
		Code:    domain.CodeSystemInternal,
		Message: message,
	})
	c.Abort()
}

// SendValidationError handles ozzo-validation.Errors specifically.
func SendValidationError(c *gin.Context, err error) {
	localizer := appI18n.GetLocalizer(c)

	vErrs, ok := err.(validation.Errors)
	if !ok {
		c.Error(err)
		SendError(c, domain.NewError(domain.CodeSystemInternal))
		return
	}

	fieldErrors := make(map[string][]string)
	for path, fieldErr := range vErrs {
		parts := strings.Split(path, ".")
		leaf := parts[len(parts)-1]

		label := leaf
		if translated := localize(c, localizer, "labels."+leaf, nil); translated != "" {
			label = translated
		}

		// Extract ozzo validation error details
		if vErr, ok := fieldErr.(validation.Error); ok {
			code := vErr.Code()
			params := vErr.Params()

			// Add field to params for i18n template
			if params == nil {
				params = make(map[string]interface{})
			}
			params["field"] = label

			// Use ozzo code directly as i18n key
			message := localize(c, localizer, code, params)
			if message == "" {
				message = vErr.Message()
			}
			fieldErrors[path] = append(fieldErrors[path], message)
		} else {
			// Fallback for non-ozzo errors
			fieldErrors[path] = append(fieldErrors[path], fieldErr.Error())
		}
	}

	msgKey := getMessageKey(c, domain.CodeSystemValidation)
	message := localize(c, localizer, msgKey, nil)

	c.JSON(HttpCodeFromCode(domain.CodeSystemValidation), domain.Response[any]{
		Code:    domain.CodeSystemValidation,
		Message: message,
		Errors:  fieldErrors,
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

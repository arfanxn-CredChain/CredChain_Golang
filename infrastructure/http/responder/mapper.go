package responder

import (
	"CredChain_Golang/domain"
	"net/http"
)

// HttpCodes maps every domain status code to its exact HTTP status code.
var HttpCodes = map[int]int{
	domain.CodeSystemSuccess:    http.StatusOK,
	domain.CodeSystemValidation: http.StatusBadRequest,
	domain.CodeSystemInternal:   http.StatusInternalServerError,

	domain.CodeAuthLoginSuccess:      http.StatusOK,
	domain.CodeAuthLoginUnauthorized: http.StatusUnauthorized,
	domain.CodeAuthLoginInvalidToken: http.StatusUnauthorized,
	domain.CodeAuthLoginForbidden:    http.StatusForbidden,
	domain.CodeAuthLoginJWTFailed:    http.StatusInternalServerError,

	domain.CodeUserFetchSuccess:      http.StatusOK,
	domain.CodeUserFetchNotFound:     http.StatusNotFound,
	domain.CodeUserCreateSuccess:     http.StatusCreated,
	domain.CodeUserCreateFailed:      http.StatusInternalServerError,
	domain.CodeUserProfileSuccess:    http.StatusOK,
	domain.CodeUserProfileFailed:     http.StatusInternalServerError,
	domain.CodeUserEmailSuccess:      http.StatusOK,
	domain.CodeUserEmailConflict:     http.StatusConflict,
	domain.CodeUserRoleSuccess:                      http.StatusOK,
	domain.CodeUserRoleFailed:                       http.StatusInternalServerError,
	domain.CodeUserRoleAdminUpdatePeerForbidden:     http.StatusForbidden,
	domain.CodeUserRoleSignerAdminRequiredForbidden: http.StatusForbidden,
	domain.CodeUserCredentialsFetchSuccess: http.StatusOK,
	domain.CodeUserCredentialsFetchFailed:  http.StatusInternalServerError,

	domain.CodeCredentialFetchSuccess:  http.StatusOK,
	domain.CodeCredentialFetchNotFound: http.StatusNotFound,
	domain.CodeCredentialIssueSuccess:  http.StatusCreated,
	domain.CodeCredentialIssueFailed:   http.StatusInternalServerError,
	domain.CodeCredentialRevokeSuccess: http.StatusOK,
	domain.CodeCredentialRevokeFailed:  http.StatusInternalServerError,
	domain.CodeCredentialVerifySuccess: http.StatusOK,
	domain.CodeCredentialVerifyFailed:  http.StatusUnprocessableEntity,
}

// HttpCodeFromCode looks up the HTTP status for a given domain status code.
func HttpCodeFromCode(code int) int {
	if httpCode, ok := HttpCodes[code]; ok {
		return httpCode
	}
	return http.StatusInternalServerError
}

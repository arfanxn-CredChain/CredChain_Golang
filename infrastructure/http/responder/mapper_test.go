package responder

import (
	"net/http"
	"testing"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

// allDomainCodes is the canonical list of every code constant in domain/codes.go.
// Adding a new code requires adding it here AND to both maps. This catches drift.
var allDomainCodes = []int{
	// System
	domain.CodeSystemSuccess,
	domain.CodeSystemValidation,
	domain.CodeSystemInternal,
	// Auth
	domain.CodeAuthSuccess,
	domain.CodeAuthUnauthorized,
	domain.CodeAuthInvalidToken,
	domain.CodeAuthForbidden,
	domain.CodeAuthRateLimitExceeded,
	domain.CodeAuthJWTFailed,
	// Auth Google Login
	domain.CodeAuthGoogleLoginSuccess,
	domain.CodeAuthGoogleLoginInvalidToken,
	domain.CodeAuthGoogleLoginUserNotFound,
	domain.CodeAuthGoogleLoginJWTFailed,
	// Auth Refresh
	domain.CodeAuthRefreshSuccess,
	domain.CodeAuthRefreshInvalidToken,
	domain.CodeAuthRefreshTokenExpired,
	domain.CodeAuthRefreshTokenRevoked,
	domain.CodeAuthRefreshUserNotFound,
	domain.CodeAuthRefreshJWTFailed,
	// Auth Logout
	domain.CodeAuthLogoutSuccess,
	// User
	domain.CodeUserFetchSuccess,
	domain.CodeUserFetchNotFound,
	domain.CodeUserStoreSuccess,
	domain.CodeUserStoreEmailDuplicateInBatch,
	domain.CodeUserStoreEmailDuplicateInDatabase,
	domain.CodeUserStoreWalletGenerationFailed,
	domain.CodeUserStoreBlockchainSyncFailed,
	domain.CodeUserStoreSuperAdminForbidden,
	domain.CodeUserStoreAdminCreateAdminForbidden,
	domain.CodeUserProfileSuccess,
	domain.CodeUserProfileFailed,
	domain.CodeUserEmailSuccess,
	domain.CodeUserEmailConflict,
	domain.CodeUserRoleSuccess,
	domain.CodeUserRoleFailed,
	domain.CodeUserRoleAdminUpdatePeerForbidden,
	domain.CodeUserRoleSignerAdminRequiredForbidden,
	domain.CodeUserRoleSameRoleUpdateForbidden,
	domain.CodeUserRoleSuperAdminBatchForbidden,
	domain.CodeUserCredentialsFetchSuccess,
	domain.CodeUserCredentialsFetchFailed,
	domain.CodeUserBatchDeleteSuccess,
	domain.CodeUserBatchDeleteFailed,
	domain.CodeUserDeleteAdminForbidden,
	// Credential
	domain.CodeCredentialFetchSuccess,
	domain.CodeCredentialFetchNotFound,
	domain.CodeCredentialIssueSuccess,
	domain.CodeCredentialIssueFailed,
	domain.CodeCredentialRevokeSuccess,
	domain.CodeCredentialRevokeFailed,
	domain.CodeCredentialVerifySuccess,
	domain.CodeCredentialVerifyFailed,
}

func TestHttpCodeFromCode_KnownCodes(t *testing.T) {
	tests := map[int]int{
		domain.CodeSystemSuccess:         http.StatusOK,
		domain.CodeAuthUnauthorized:      http.StatusUnauthorized,
		domain.CodeAuthForbidden:         http.StatusForbidden,
		domain.CodeUserFetchNotFound:     http.StatusNotFound,
		domain.CodeUserStoreSuccess:      http.StatusCreated,
		domain.CodeAuthRateLimitExceeded: http.StatusTooManyRequests,
	}
	for code, want := range tests {
		assert.Equal(t, want, HttpCodeFromCode(code), "code %d", code)
	}
}

func TestHttpCodeFromCode_UnknownReturns500(t *testing.T) {
	assert.Equal(t, http.StatusInternalServerError, HttpCodeFromCode(999999))
}

func TestRegistry_EveryCodeHasMessageKey(t *testing.T) {
	for _, code := range allDomainCodes {
		_, ok := CodeToMessageKey[code]
		assert.True(t, ok, "code %d missing from CodeToMessageKey", code)
	}
}

func TestRegistry_EveryCodeHasHttpStatus(t *testing.T) {
	for _, code := range allDomainCodes {
		_, ok := HttpCodes[code]
		assert.True(t, ok, "code %d missing from HttpCodes", code)
	}
}

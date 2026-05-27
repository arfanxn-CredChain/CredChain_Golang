package responder

import (
	"CredChain_Golang/domain"
	"net/http"
)

// CodeToMessageKey maps every status code to its i18n message key.
var CodeToMessageKey = map[int]string{
	// System codes
	domain.CodeSystemSuccess:    "success_generic",
	domain.CodeSystemValidation: "error_invalid_payload",
	domain.CodeSystemInternal:   "error_internal",

	// Auth codes
	domain.CodeAuthSuccess:                    "success_login",
	domain.CodeAuthUnauthorized:               "error_unauthorized",
	domain.CodeAuthInvalidToken:               "error_invalid_token",
	domain.CodeAuthForbidden:                  "error_unauthorized_email",
	domain.CodeAuthJWTFailed:                  "error_token_issue_failed",
	domain.CodeAuthRateLimitExceeded:          "error_rate_limit_exceeded",

	// Auth Google Login codes
	domain.CodeAuthGoogleLoginSuccess:         "success_login",
	domain.CodeAuthGoogleLoginInvalidToken:    "error_google_token_invalid",
	domain.CodeAuthGoogleLoginUserNotFound:    "error_unauthorized_email",
	domain.CodeAuthGoogleLoginJWTFailed:       "error_token_issue_failed",

	// Auth Refresh codes
	domain.CodeAuthRefreshSuccess:       "success_login",
	domain.CodeAuthRefreshInvalidToken:  "error_invalid_token",
	domain.CodeAuthRefreshTokenExpired:  "error_refresh_token_expired",
	domain.CodeAuthRefreshTokenRevoked:  "error_refresh_token_revoked",
	domain.CodeAuthRefreshUserNotFound:  "error_user_not_found",
	domain.CodeAuthRefreshJWTFailed:     "error_token_issue_failed",

	// Auth Logout codes
	domain.CodeAuthLogoutSuccess:              "success_logout",

	// User codes
	domain.CodeUserFetchSuccess:                    "success_user_fetched",
	domain.CodeUserFetchNotFound:                   "error_user_not_found",
	domain.CodeUserStoreSuccess:                    "success_users_created",
	domain.CodeUserStoreEmailDuplicateInBatch:      "error_email_duplicate_in_batch",
	domain.CodeUserStoreEmailDuplicateInDatabase:   "error_email_duplicate_in_database",
	domain.CodeUserStoreWalletGenerationFailed:     "error_store_wallet_generation_failed",
	domain.CodeUserStoreBlockchainSyncFailed:       "error_store_blockchain_sync_failed",
	domain.CodeUserStoreSuperAdminForbidden:        "error_store_super_admin_forbidden",
	domain.CodeUserStoreAdminCreateAdminForbidden:  "error_store_admin_create_admin_forbidden",
	domain.CodeUserProfileSuccess:                  "success_profile_updated",
	domain.CodeUserProfileFailed:                   "error_update_profile_failed",
	domain.CodeUserEmailSuccess:                    "success_email_updated",
	domain.CodeUserEmailConflict:                   "error_update_email_failed",
	domain.CodeUserRoleSuccess:                     "success_role_updated",
	domain.CodeUserRoleFailed:                      "error_update_role_failed",
	domain.CodeUserRoleAdminUpdatePeerForbidden:    "error_admin_update_peer_role_forbidden",
	domain.CodeUserRoleSignerAdminRequiredForbidden: "error_signer_admin_required_forbidden",
	domain.CodeUserRoleSameRoleUpdateForbidden:     "error_same_role_update_forbidden",
	domain.CodeUserRoleSuperAdminBatchForbidden:    "error_super_admin_batch_forbidden",
	domain.CodeUserCredentialsFetchSuccess:         "success_credentials_fetched",
	domain.CodeUserCredentialsFetchFailed:          "error_fetch_credentials_failed",
	domain.CodeUserBatchDeleteSuccess:              "success_credentials_deleted",
	domain.CodeUserBatchDeleteFailed:               "error_credentials_delete_failed",
	domain.CodeUserDeleteAdminForbidden:            "error_user_delete_admin_forbidden",

	// Credential codes
	domain.CodeCredentialFetchSuccess:  "success_credential_fetched",
	domain.CodeCredentialFetchNotFound: "error_credential_not_found",
	domain.CodeCredentialIssueSuccess:  "success_credential_issued",
	domain.CodeCredentialIssueFailed:   "error_credential_issue_failed",
	domain.CodeCredentialRevokeSuccess: "success_credential_revoked",
	domain.CodeCredentialRevokeFailed:  "error_credential_revoke_failed",
	domain.CodeCredentialVerifySuccess: "success_credential_verified",
	domain.CodeCredentialVerifyFailed:  "error_credential_verify_failed",
}

// HttpCodes maps every domain status code to its exact HTTP status code.
var HttpCodes = map[int]int{
	domain.CodeSystemSuccess:    http.StatusOK,
	domain.CodeSystemValidation: http.StatusBadRequest,
	domain.CodeSystemInternal:   http.StatusInternalServerError,

	domain.CodeAuthSuccess:                    http.StatusOK,
	domain.CodeAuthUnauthorized:               http.StatusUnauthorized,
	domain.CodeAuthInvalidToken:               http.StatusUnauthorized,
	domain.CodeAuthForbidden:                  http.StatusForbidden,
	domain.CodeAuthJWTFailed:                  http.StatusInternalServerError,
	domain.CodeAuthRateLimitExceeded:          http.StatusTooManyRequests,

	domain.CodeAuthGoogleLoginSuccess:         http.StatusOK,
	domain.CodeAuthGoogleLoginInvalidToken:    http.StatusUnauthorized,
	domain.CodeAuthGoogleLoginUserNotFound:    http.StatusForbidden,
	domain.CodeAuthGoogleLoginJWTFailed:       http.StatusInternalServerError,

	domain.CodeAuthRefreshSuccess:       http.StatusOK,
	domain.CodeAuthRefreshInvalidToken:  http.StatusUnauthorized,
	domain.CodeAuthRefreshTokenExpired:  http.StatusUnauthorized,
	domain.CodeAuthRefreshTokenRevoked:  http.StatusUnauthorized,
	domain.CodeAuthRefreshUserNotFound:  http.StatusNotFound,
	domain.CodeAuthRefreshJWTFailed:     http.StatusInternalServerError,

	domain.CodeAuthLogoutSuccess:              http.StatusOK,

	domain.CodeUserFetchSuccess:      http.StatusOK,
	domain.CodeUserFetchNotFound:     http.StatusNotFound,
	domain.CodeUserStoreSuccess:      http.StatusCreated,
	domain.CodeUserStoreEmailDuplicateInBatch:      http.StatusBadRequest,
	domain.CodeUserStoreEmailDuplicateInDatabase:   http.StatusBadRequest,
	domain.CodeUserStoreWalletGenerationFailed:     http.StatusInternalServerError,
	domain.CodeUserStoreBlockchainSyncFailed:       http.StatusInternalServerError,
	domain.CodeUserStoreSuperAdminForbidden:        http.StatusForbidden,
	domain.CodeUserStoreAdminCreateAdminForbidden:  http.StatusForbidden,
	domain.CodeUserProfileSuccess:    http.StatusOK,
	domain.CodeUserProfileFailed:     http.StatusInternalServerError,
	domain.CodeUserEmailSuccess:      http.StatusOK,
	domain.CodeUserEmailConflict:     http.StatusConflict,
	domain.CodeUserRoleSuccess:                      http.StatusOK,
	domain.CodeUserRoleFailed:                       http.StatusInternalServerError,
	domain.CodeUserRoleAdminUpdatePeerForbidden:     http.StatusForbidden,
	domain.CodeUserRoleSignerAdminRequiredForbidden: http.StatusForbidden,
	domain.CodeUserRoleSameRoleUpdateForbidden:      http.StatusForbidden,
	domain.CodeUserRoleSuperAdminBatchForbidden:     http.StatusForbidden,
	domain.CodeUserCredentialsFetchSuccess: http.StatusOK,
	domain.CodeUserCredentialsFetchFailed:  http.StatusInternalServerError,
	domain.CodeUserBatchDeleteSuccess: http.StatusOK,
	domain.CodeUserBatchDeleteFailed:  http.StatusInternalServerError,
	domain.CodeUserDeleteAdminForbidden: http.StatusForbidden,

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

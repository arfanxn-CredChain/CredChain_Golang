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
	domain.CodeAuthUnauthorized:      "error_unauthorized",
	domain.CodeAuthInvalidToken:      "error_invalid_token",
	domain.CodeAuthForbidden:         "error_unauthorized_email",
	domain.CodeAuthRateLimitExceeded: "error_rate_limit_exceeded",

	// Auth Google Login codes
	domain.CodeAuthGoogleLoginSuccess:      "success_login",
	domain.CodeAuthGoogleLoginInvalidToken: "error_google_token_invalid",
	domain.CodeAuthGoogleLoginUserNotFound: "error_unauthorized_email",
	domain.CodeAuthGoogleLoginJWTFailed:    "error_token_issue_failed",

	// Auth Refresh codes
	domain.CodeAuthRefreshSuccess:      "success_login",
	domain.CodeAuthRefreshInvalidToken: "error_invalid_token",
	domain.CodeAuthRefreshTokenExpired: "error_refresh_token_expired",
	domain.CodeAuthRefreshTokenRevoked: "error_refresh_token_revoked",
	domain.CodeAuthRefreshUserNotFound: "error_user_not_found",
	domain.CodeAuthRefreshJWTFailed:    "error_token_issue_failed",

	// Auth Logout codes
	domain.CodeAuthLogoutSuccess: "success_logout",

	// User codes
	domain.CodeUserFetchSuccess:                     "success_user_fetched",
	domain.CodeUserFetchNotFound:                    "error_user_not_found",
	domain.CodeUserStoreSuccess:                     "success_users_created",
	domain.CodeUserStoreEmailDuplicateInBatch:       "error_email_duplicate_in_batch",
	domain.CodeUserStoreEmailDuplicateInDatabase:    "error_email_duplicate_in_database",
	domain.CodeUserStoreWalletGenerationFailed:      "error_store_wallet_generation_failed",
	domain.CodeUserStoreBlockchainSyncFailed:        "error_store_blockchain_sync_failed",
	domain.CodeUserStoreSuperAdminForbidden:         "error_store_super_admin_forbidden",
	domain.CodeUserStoreAdminCreateAdminForbidden:   "error_store_admin_create_admin_forbidden",
	domain.CodeUserProfileSuccess:                   "success_profile_updated",
	domain.CodeUserEmailSuccess:                     "success_email_updated",
	domain.CodeUserEmailConflict:                    "error_update_email_failed",
	domain.CodeUserEmailMismatchedIdToken:           "error_email_mismatched_id_token",
	domain.CodeUserEmailInvalidIdToken:              "error_email_invalid_id_token",
	domain.CodeUserRoleSuccess:                      "success_role_updated",
	domain.CodeUserRoleAdminUpdatePeerForbidden:     "error_admin_update_peer_role_forbidden",
	domain.CodeUserRoleSignerAdminRequiredForbidden: "error_signer_admin_required_forbidden",
	domain.CodeUserRoleSameRoleUpdateForbidden:      "error_same_role_update_forbidden",
	domain.CodeUserRoleSuperAdminBatchForbidden:     "error_super_admin_batch_forbidden",
	domain.CodeUserRoleBlockchainSyncFailed:         "error_role_blockchain_sync_failed",
	domain.CodeUserRoleSelfTargetForbidden:          "error_role_self_target_forbidden",
	domain.CodeUserBatchDeleteSuccess:               "success_credentials_deleted",
	domain.CodeUserDeleteAdminForbidden:             "error_user_delete_admin_forbidden",
	domain.CodeUserDeleteBlockchainSyncFailed:       "error_delete_blockchain_sync_failed",
	domain.CodeUserDeleteSelfTargetForbidden:        "error_delete_self_target_forbidden",
	domain.CodeUserUpdateSuccess:                    "success_users_updated",
	domain.CodeUserUpdateNotFound:                   "error_users_update_not_found",
	domain.CodeUserUpdatePeerAdminForbidden:         "error_users_update_peer_admin_forbidden",
	domain.CodeUserUpdateSuperAdminForbidden:        "error_users_update_super_admin_forbidden",
	domain.CodeUserUpdateSelfForbidden:              "error_users_update_self_forbidden",
	domain.CodeUserUpdateBlockchainSyncFailed:       "error_users_update_blockchain_sync_failed",

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

	domain.CodeAuthUnauthorized:      http.StatusUnauthorized,
	domain.CodeAuthInvalidToken:      http.StatusUnauthorized,
	domain.CodeAuthForbidden:         http.StatusForbidden,
	domain.CodeAuthRateLimitExceeded: http.StatusTooManyRequests,

	domain.CodeAuthGoogleLoginSuccess:      http.StatusOK,
	domain.CodeAuthGoogleLoginInvalidToken: http.StatusUnauthorized,
	domain.CodeAuthGoogleLoginUserNotFound: http.StatusForbidden,
	domain.CodeAuthGoogleLoginJWTFailed:    http.StatusInternalServerError,

	domain.CodeAuthRefreshSuccess:      http.StatusOK,
	domain.CodeAuthRefreshInvalidToken: http.StatusUnauthorized,
	domain.CodeAuthRefreshTokenExpired: http.StatusUnauthorized,
	domain.CodeAuthRefreshTokenRevoked: http.StatusUnauthorized,
	domain.CodeAuthRefreshUserNotFound: http.StatusNotFound,
	domain.CodeAuthRefreshJWTFailed:    http.StatusInternalServerError,

	domain.CodeAuthLogoutSuccess: http.StatusOK,

	domain.CodeUserFetchSuccess:                     http.StatusOK,
	domain.CodeUserFetchNotFound:                    http.StatusNotFound,
	domain.CodeUserStoreSuccess:                     http.StatusCreated,
	domain.CodeUserStoreEmailDuplicateInBatch:       http.StatusBadRequest,
	domain.CodeUserStoreEmailDuplicateInDatabase:    http.StatusBadRequest,
	domain.CodeUserStoreWalletGenerationFailed:      http.StatusInternalServerError,
	domain.CodeUserStoreBlockchainSyncFailed:        http.StatusInternalServerError,
	domain.CodeUserStoreSuperAdminForbidden:         http.StatusForbidden,
	domain.CodeUserStoreAdminCreateAdminForbidden:   http.StatusForbidden,
	domain.CodeUserProfileSuccess:                   http.StatusOK,
	domain.CodeUserEmailSuccess:                     http.StatusOK,
	domain.CodeUserEmailConflict:                    http.StatusConflict,
	domain.CodeUserEmailMismatchedIdToken:           http.StatusUnprocessableEntity,
	domain.CodeUserEmailInvalidIdToken:              http.StatusUnauthorized,
	domain.CodeUserRoleSuccess:                      http.StatusOK,
	domain.CodeUserRoleAdminUpdatePeerForbidden:     http.StatusForbidden,
	domain.CodeUserRoleSignerAdminRequiredForbidden: http.StatusForbidden,
	domain.CodeUserRoleSameRoleUpdateForbidden:      http.StatusForbidden,
	domain.CodeUserRoleSuperAdminBatchForbidden:     http.StatusForbidden,
	domain.CodeUserRoleBlockchainSyncFailed:         http.StatusInternalServerError,
	domain.CodeUserRoleSelfTargetForbidden:          http.StatusForbidden,
	domain.CodeUserBatchDeleteSuccess:               http.StatusOK,
	domain.CodeUserDeleteAdminForbidden:             http.StatusForbidden,
	domain.CodeUserDeleteBlockchainSyncFailed:       http.StatusInternalServerError,
	domain.CodeUserDeleteSelfTargetForbidden:        http.StatusForbidden,
	domain.CodeUserUpdateSuccess:                    http.StatusOK,
	domain.CodeUserUpdateNotFound:                   http.StatusNotFound,
	domain.CodeUserUpdatePeerAdminForbidden:         http.StatusForbidden,
	domain.CodeUserUpdateSuperAdminForbidden:        http.StatusForbidden,
	domain.CodeUserUpdateSelfForbidden:              http.StatusForbidden,
	domain.CodeUserUpdateBlockchainSyncFailed:       http.StatusInternalServerError,

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

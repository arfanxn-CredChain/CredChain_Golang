package domain

// Response is the unified response envelope for every API endpoint.
type Response[T any] struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    T                   `json:"data,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func (r *Response[T]) Error() string {
	return r.Message
}

// ---- 6-Digit AABBCC Status Code Registry ----
//
// AA = Category:  10 System | 20 Auth | 30 User | 40 Credential
// BB = Feature:   01, 02, 03 ... (resets per category)
// CC = Status:    00-19 Success | 20-39 Reserved | 40-99 Error

const (
	// ── System (10) ──────────────────────────────────────────────────────────
	CodeSystemSuccess    = 100000
	CodeSystemValidation = 100040
	CodeSystemInternal   = 100050

	// ── Auth (20) ────────────────────────────────────────────────────────────
	CodeAuthLoginSuccess      = 200100
	CodeAuthLoginUnauthorized = 200140
	CodeAuthLoginInvalidToken = 200141
	CodeAuthLoginForbidden    = 200142
	CodeAuthLoginJWTFailed    = 200150

	// ── User (30) ────────────────────────────────────────────────────────────
	CodeUserFetchSuccess                     = 300100
	CodeUserFetchNotFound                    = 300140
	CodeUserCreateSuccess                    = 300200
	CodeUserCreateFailed                     = 300240
	CodeUserProfileSuccess                   = 300300
	CodeUserProfileFailed                    = 300340
	CodeUserEmailSuccess                     = 300400
	CodeUserEmailConflict                    = 300440
	CodeUserRoleSuccess                      = 300500
	CodeUserRoleFailed                       = 300540
	CodeUserRoleAdminUpdatePeerForbidden     = 300541
	CodeUserRoleSignerAdminRequiredForbidden = 300542
	CodeUserRoleSameRoleUpdateForbidden      = 300543
	CodeUserRoleSuperAdminBatchForbidden     = 300544
	CodeUserCredentialsFetchSuccess          = 300600
	CodeUserCredentialsFetchFailed           = 300640
	CodeUserBatchDeleteSuccess               = 300700
	CodeUserBatchDeleteFailed                = 300740
	CodeUserDeleteAdminForbidden             = 300741

	// ── Credential (40) ──────────────────────────────────────────────────────
	CodeCredentialFetchSuccess  = 400100
	CodeCredentialFetchNotFound = 400140
	CodeCredentialIssueSuccess  = 400200
	CodeCredentialIssueFailed   = 400240
	CodeCredentialRevokeSuccess = 400300
	CodeCredentialRevokeFailed  = 400340
	CodeCredentialVerifySuccess = 400400
	CodeCredentialVerifyFailed  = 400440
)

// MessageKeys maps every status code to its i18n message key.
var MessageKeys = map[int]string{
	CodeSystemSuccess:    "success_generic",
	CodeSystemValidation: "error_invalid_payload",
	CodeSystemInternal:   "error_internal",

	CodeAuthLoginSuccess:      "success_login",
	CodeAuthLoginUnauthorized: "error_unauthorized",
	CodeAuthLoginInvalidToken: "error_invalid_token",
	CodeAuthLoginForbidden:    "error_unauthorized_email",
	CodeAuthLoginJWTFailed:    "error_token_issue_failed",

	CodeUserFetchSuccess:                     "success_user_fetched",
	CodeUserFetchNotFound:                    "error_user_not_found",
	CodeUserCreateSuccess:                    "success_users_created",
	CodeUserCreateFailed:                     "error_create_users_failed",
	CodeUserProfileSuccess:                   "success_profile_updated",
	CodeUserProfileFailed:                    "error_update_profile_failed",
	CodeUserEmailSuccess:                     "success_email_updated",
	CodeUserEmailConflict:                    "error_update_email_failed",
	CodeUserRoleSuccess:                      "success_role_updated",
	CodeUserRoleFailed:                       "error_update_role_failed",
	CodeUserRoleAdminUpdatePeerForbidden:     "error_admin_update_peer_role_forbidden",
	CodeUserRoleSignerAdminRequiredForbidden: "error_signer_admin_required_forbidden",
	CodeUserRoleSameRoleUpdateForbidden:      "error_same_role_update_forbidden",
	CodeUserRoleSuperAdminBatchForbidden:     "error_super_admin_batch_forbidden",
	CodeUserCredentialsFetchSuccess:          "success_credentials_fetched",
	CodeUserCredentialsFetchFailed:           "error_fetch_credentials_failed",
	CodeUserBatchDeleteSuccess:               "success_credentials_deleted",
	CodeUserBatchDeleteFailed:                "error_credentials_delete_failed",
	CodeUserDeleteAdminForbidden:             "error_user_delete_admin_forbidden",

	CodeCredentialFetchSuccess:   "success_credential_fetched",
	CodeCredentialFetchNotFound:  "error_credential_not_found",
	CodeCredentialIssueSuccess:   "success_credential_issued",
	CodeCredentialIssueFailed:    "error_credential_issue_failed",
	CodeCredentialRevokeSuccess:  "success_credential_revoked",
	CodeCredentialRevokeFailed:   "error_credential_revoke_failed",
	CodeCredentialVerifySuccess:  "success_credential_verified",
	CodeCredentialVerifyFailed:   "error_credential_verify_failed",
}

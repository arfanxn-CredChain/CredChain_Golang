package domain

// Response is the unified response envelope for every API endpoint.
type Response struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    any                 `json:"data,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func (r *Response) Error() string {
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
	CodeUserFetchSuccess    = 300100
	CodeUserFetchNotFound   = 300140
	CodeUserCreateSuccess   = 300200
	CodeUserCreateFailed    = 300240
	CodeUserProfileSuccess  = 300300
	CodeUserProfileFailed   = 300340
	CodeUserEmailSuccess    = 300400
	CodeUserEmailConflict   = 300440
	CodeUserRoleSuccess                      = 300500
	CodeUserRoleFailed                       = 300540
	CodeUserRoleAdminUpdatePeerForbidden     = 300541
	CodeUserRoleSignerAdminRequiredForbidden = 300542
	CodeUserCredsFetchSuccess = 300600
	CodeUserCredsFetchFailed  = 300640

	// ── Credential (40) ──────────────────────────────────────────────────────
	CodeCredFetchSuccess  = 400100
	CodeCredFetchNotFound = 400140
	CodeCredIssueSuccess  = 400200
	CodeCredIssueFailed   = 400240
	CodeCredRevokeSuccess = 400300
	CodeCredRevokeFailed  = 400340
	CodeCredVerifySuccess = 400400
	CodeCredVerifyFailed  = 400440
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

	CodeUserFetchSuccess:      "success_user_fetched",
	CodeUserFetchNotFound:     "error_user_not_found",
	CodeUserCreateSuccess:     "success_users_created",
	CodeUserCreateFailed:      "error_create_users_failed",
	CodeUserProfileSuccess:    "success_profile_updated",
	CodeUserProfileFailed:     "error_update_profile_failed",
	CodeUserEmailSuccess:      "success_email_updated",
	CodeUserEmailConflict:     "error_update_email_failed",
	CodeUserRoleSuccess:                      "success_role_updated",
	CodeUserRoleFailed:                       "error_update_role_failed",
	CodeUserRoleAdminUpdatePeerForbidden:     "error_admin_update_peer_role_forbidden",
	CodeUserRoleSignerAdminRequiredForbidden: "error_signer_admin_required_forbidden",
	CodeUserCredsFetchSuccess: "success_creds_fetched",
	CodeUserCredsFetchFailed:  "error_fetch_creds_failed",

	CodeCredFetchSuccess:  "success_cred_fetched",
	CodeCredFetchNotFound: "error_cred_not_found",
	CodeCredIssueSuccess:  "success_cred_issued",
	CodeCredIssueFailed:   "error_cred_issue_failed",
	CodeCredRevokeSuccess: "success_cred_revoked",
	CodeCredRevokeFailed:  "error_cred_revoke_failed",
	CodeCredVerifySuccess: "success_cred_verified",
	CodeCredVerifyFailed:  "error_cred_verify_failed",
}

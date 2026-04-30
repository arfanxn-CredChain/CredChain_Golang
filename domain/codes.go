package domain

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
	CodeUserStoreSuccess                     = 300200
	CodeUserStoreEmailDuplicateInBatch       = 300241
	CodeUserStoreEmailDuplicateInDatabase    = 300242
	CodeUserStoreWalletGenerationFailed      = 300243
	CodeUserStoreBlockchainSyncFailed        = 300244
	CodeUserStoreSuperAdminForbidden         = 300245
	CodeUserStoreAdminCreateAdminForbidden   = 300246
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

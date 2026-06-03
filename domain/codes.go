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
	CodeAuthUnauthorized      = 200140
	CodeAuthInvalidToken      = 200141
	CodeAuthForbidden         = 200142
	CodeAuthRateLimitExceeded = 200143

	// ── Auth Google Login (20) ───────────────────────────────────────────────
	CodeAuthGoogleLoginSuccess      = 200200
	CodeAuthGoogleLoginInvalidToken = 200241
	CodeAuthGoogleLoginUserNotFound = 200242
	CodeAuthGoogleLoginJWTFailed    = 200250

	// ── Auth Refresh (20) ──────────────────────────────────────────────
	CodeAuthRefreshSuccess      = 200300
	CodeAuthRefreshInvalidToken = 200340
	CodeAuthRefreshTokenExpired = 200341
	CodeAuthRefreshTokenRevoked = 200342
	CodeAuthRefreshUserNotFound = 200343
	CodeAuthRefreshJWTFailed    = 200350

	// ── Auth Logout (20) ───────────────────────────────────────────────
	CodeAuthLogoutSuccess = 200400

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
	CodeUserEmailSuccess                     = 300400
	CodeUserEmailConflict                    = 300440
	CodeUserEmailMismatchedIdToken           = 300441
	CodeUserEmailInvalidIdToken              = 300442
	CodeUserRoleSuccess                      = 300500
	CodeUserRoleAdminUpdatePeerForbidden     = 300541
	CodeUserRoleSignerAdminRequiredForbidden = 300542
	CodeUserRoleSameRoleUpdateForbidden      = 300543
	CodeUserRoleSuperAdminBatchForbidden     = 300544
	CodeUserRoleBlockchainSyncFailed         = 300545
	CodeUserRoleSelfTargetForbidden          = 300546
	CodeUserRoleTrashedForbidden             = 300547
	CodeUserBatchDeleteSuccess               = 300700
	CodeUserDeleteAdminForbidden             = 300741
	CodeUserDeleteBlockchainSyncFailed       = 300742
	CodeUserDeleteSelfTargetForbidden        = 300743
	CodeUserUpdateSuccess                    = 300800
	CodeUserUpdateNotFound                   = 300841
	CodeUserUpdatePeerAdminForbidden         = 300842
	CodeUserUpdateSuperAdminForbidden        = 300843
	CodeUserUpdateSelfForbidden              = 300844
	CodeUserUpdateBlockchainSyncFailed       = 300845
	CodeUserUpdateTrashedForbidden           = 300846

	CodeUserTransferSuperAdminSuccess              = 300600
	CodeUserTransferSuperAdminSelfTargetForbidden  = 300641
	CodeUserTransferSuperAdminTargetNotFound       = 300642
	CodeUserTransferSuperAdminTrashedForbidden     = 300643
	CodeUserTransferSuperAdminBlockchainSyncFailed = 300645

	// ── Credential (40) ──────────────────────────────────────────────────────
	CodeCredentialFetchSuccess    = 400100
	CodeCredentialFetchNotFound   = 400140
	CodeCredentialFetchValidation = 400141

	CodeCredentialIssueSuccess              = 400200
	CodeCredentialIssueFailed               = 400240
	CodeCredentialIssueValidation           = 400241
	CodeCredentialIssueDuplicateFileHash    = 400242
	CodeCredentialIssueHolderNotFound       = 400243
	CodeCredentialIssueBlockchainSyncFailed = 400244
	CodeCredentialIssueStorageFailed        = 400245
	CodeCredentialIssueHashFailed           = 400246

	CodeCredentialRevokeSuccess              = 400300
	CodeCredentialRevokeFailed               = 400340
	CodeCredentialRevokeNotFound             = 400341
	CodeCredentialRevokeAlreadyRevoked       = 400342
	CodeCredentialRevokeBlockchainSyncFailed = 400343

	CodeCredentialVerifySuccess            = 400400
	CodeCredentialVerifyFailed             = 400440
	CodeCredentialVerifyValidation         = 400441
	CodeCredentialVerifyExtractNotReady    = 400442
	CodeCredentialVerifyExtractFailed      = 400443
	CodeCredentialVerifyAiServiceFailed    = 400444
	CodeCredentialVerifyCredentialNotFound = 400445
)

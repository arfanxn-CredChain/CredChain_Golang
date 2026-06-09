# CredChain Role System

## Role Definitions

Five roles form strict hierarchy (0=lowest, 4=highest):

| Rank | Go Constant | Go String | Solidity Name | Solidity Value | Postgres ENUM | Description |
|------|-------------|-----------|---------------|----------------|---------------|-------------|
| 0 | `RoleNone` | `"none"` | `None` | 0 | *(not persisted)* | Chain-only revocation target |
| 1 | `RoleHolder` | `"holder"` | `Holder` | 1 | `holder` | Basic credential holder |
| 2 | `RoleIssuer` | `"issuer"` | `Issuer` | 2 | `issuer` | Can issue/revoke credentials |
| 3 | `RoleAdmin` | `"admin"` | `Admin` | 3 | `admin` | User management (limited) |
| 4 | `RoleSuperAdmin` | `"super_admin"` | `SuperAdmin` | 4 | `super_admin` | Full system control |

**Sources:**
- Go: `domain/user.go:11-19` (constants), `domain/user.go:45-60` (`ToUint8`), `domain/user.go:63-78` (`RoleFromUint8`)
- Solidity: `CredentialAuthority.sol:19-25` (enum)
- Postgres: `infrastructure/database/migrations/000001_initial_schema.up.sql:1-6`

`RoleNone` is never persisted in the database — it exists only in Go domain code for on-chain revocation calls.

---

## Role Hierarchy

**Go** (`domain/user.go:22-37`):
```go
func (r Role) Rank() int {
    switch r {
    case RoleSuperAdmin: return 4
    case RoleAdmin:      return 3
    case RoleIssuer:     return 2
    case RoleHolder:     return 1
    case RoleNone:       return 0
    default:             return 0
    }
}
```

**Solidity** (`CredentialAuthority.sol:118-120`):
```solidity
function hasRoleOrAbove(address user, Role minimumRole) public view returns (bool) {
    return userToRole[user] >= minimumRole;
}
```

Both leverage Solidity enum natural ordering, with Go mirroring via `Rank()`.

---

## API Route Authorization

**Source:** `infrastructure/http/router.go:68-106`

Middleware chain: `ErrorLoggerMiddleware` → `I18nMiddleware` → `ApiRateLimitMiddleware` → `AuthMiddleware` → `RoleMiddleware` (if applicable)

| Route | Method | Auth | Min Role (on-chain) |
|-------|--------|------|---------------------|
| `/api/health` | GET | None | None |
| `/api/auth/google` | POST | None | None |
| `/api/auth/refresh` | POST | None | None |
| `/api/auth/logout` | POST | Authenticated | Any |
| `/api/users/self` | GET | Authenticated | Any |
| `/api/users/self/profile` | PUT | Authenticated | Any |
| `/api/users/self/email` | PUT | Authenticated | Any |
| `/api/users/self/transfer-super-admin` | POST | Authenticated | SuperAdmin |
| `/api/users/self/credentials` | GET | Authenticated | Any — lists own credentials (handler `SelfPaginate`) |
| `/api/users/self/credentials/:id` | GET | Authenticated | Any — fetch own credential; 404 if not owned (handler `SelfFind`) |
| `/api/users` | GET | Authenticated | **Issuer+** (read-only) |
| `/api/users/:id` | GET | Authenticated | **Issuer+** (read-only) |
| `/api/users/batch` | POST | Authenticated | Admin+ |
| `/api/users/batch` | PUT | Authenticated | Admin+ |
| `/api/users/batch/role` | PUT | Authenticated | Admin+ |
| `/api/users/batch` | DELETE | Authenticated | Admin+ |
| `/api/credentials` | GET | Authenticated | Issuer+ |
| `/api/credentials/:id` | GET | Authenticated | Issuer+ |
| `/api/credentials/batch/issue` | POST | Authenticated | Issuer+ |
| `/api/credentials/batch/revoke` | POST | Authenticated | Issuer+ |
| `/api/credentials/batch/reextract` | POST | Authenticated | Issuer+ |
| `/api/credentials/verify` | POST | **None (public)** | None — used by external verifiers (HR, employers) |

**Role middlewares** (`infrastructure/http/middleware/auth.go:94-146`):
- `AdminRoleMiddleware` → checks `HasRoleOrAbove(ctx, wallet, RoleAdmin)` on-chain
- `IssuerRoleMiddleware` → checks `HasRoleOrAbove(ctx, wallet, RoleIssuer)` on-chain
- `SuperAdminRoleMiddleware` → checks `HasRoleOrAbove(ctx, wallet, RoleSuperAdmin)` on-chain

All check the **on-chain** role from CredentialAuthority contract, not the DB role.

---

## Policy Layer Rules

### User Policy (`feature/user/user_policy.go`)

#### Store (Admin+)

| Rule | Condition | Code |
|------|-----------|------|
| Cannot create SuperAdmin | RoleSuperAdmin in payload | `CodeUserStoreSuperAdminForbidden` (300245) |
| Admin cannot create Admin | Signer=Admin, target role=Admin | `CodeUserStoreAdminCreateAdminForbidden` (300246) |

Allowed combos: Admin creates Holder/Issuer; SuperAdmin creates anything except SuperAdmin.

#### UpdatePreFetch (put batch)

| Rule | Condition | Code |
|------|-----------|------|
| Signer must be Admin+ | Below Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |
| Cannot update self via batch | Signer ID in target list **AND signer is not SuperAdmin** | `CodeUserUpdateSelfForbidden` (300844) |

> **SuperAdmin self-edit carve-out:** SuperAdmin may include their own ID in the batch payload (Admin still blocked). This lets SuperAdmin manage their own profile fields. See email guard in UpdatePostFetch.

#### UpdatePostFetch

| Rule | Condition | Code |
|------|-----------|------|
| Cannot change own email via batch | Target ID == signer ID AND payload email non-nil | `CodeUserUpdateSelfEmailForbidden` (300845) |
| Cannot update trashed user | Target has non-nil DeletedAt | `CodeUserUpdateTrashedForbidden` (300846) |
| Cannot update SuperAdmin | Target role is SuperAdmin | `CodeUserUpdateSuperAdminForbidden` (300843) |
| Admin cannot update Admin | Signer=Admin, target role=Admin+ | `CodeUserUpdatePeerAdminForbidden` (300842) |
| Cannot assign SuperAdmin | Payload includes RoleSuperAdmin | `CodeUserRoleSuperAdminBatchForbidden` (300544) |
| Admin cannot promote to Admin | Signer=Admin, payload role=Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |

> **Self-email guard:** Even SuperAdmin (the only role allowed to self-edit via batch) cannot change their own email via batch — they must use `PUT /api/users/self/email` (which requires a fresh Google ID token). This prevents a self-update from setting an inaccessible email and locking the account out.

#### UpdateRolePreFetch

| Rule | Condition | Code |
|------|-----------|------|
| Signer must be Admin+ | Below Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |
| Cannot assign SuperAdmin | Target role is SuperAdmin | `CodeUserRoleSuperAdminBatchForbidden` (300544) |
| Cannot target self | Signer ID in target list | `CodeUserRoleSelfTargetForbidden` (300546) |

#### UpdateRolePostFetch

| Rule | Condition | Code |
|------|-----------|------|
| Target must exist | Not found in DB | generic not-found |
| Cannot update trashed target | Target has non-nil DeletedAt | `CodeUserRoleTrashedForbidden` (300547) |
| Cannot update to same role | Target role == payload role | `CodeUserRoleSameRoleUpdateForbidden` (300543) |
| Admin cannot update Admin target | Signer=Admin, target=Admin+ | `CodeUserRoleAdminUpdatePeerForbidden` (300541) |
| Admin cannot promote to Admin | Signer=Admin, payload=Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |

#### DeletePreFetch

| Rule | Condition | Code |
|------|-----------|------|
| Signer must be Admin+ | Below Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |
| Cannot self-delete | Signer ID in target list | `CodeUserDeleteSelfTargetForbidden` (300743) |

#### DeletePostFetch

| Rule | Condition | Code |
|------|-----------|------|
| Admin cannot delete Admin/SuperAdmin | Signer=Admin, target=Admin+ | `CodeUserDeleteAdminForbidden` (300741) |

#### TransferSuperAdminPreFetch

| Rule | Condition | Code |
|------|-----------|------|
| Cannot transfer to self | Target ID == signer ID | `CodeUserTransferSuperAdminSelfTargetForbidden` (300641) |

### Credential Policy (`feature/credential/credential_policy.go`)

| Method | Rule | Code |
|--------|------|------|
| `IssuePreFetch` | Signer must be Issuer+ (DB role) | `CodeAuthForbidden` (200142) |
| `IssuePostFetch` | No-op | — |
| `RevokePreFetch` | Signer must be Issuer+ (DB role) | `CodeAuthForbidden` (200142) |
| `RevokePostFetch` | No-op (future: revoker = original issuer or rank above) | — |
| `VerifyPreFetch` | **No-op (public endpoint)** — verifier (HR, employer) needs no auth | — |
| `ReExtractPreFetch` | Signer must be Issuer+ (DB role) | `CodeAuthForbidden` (200142) |

Credential policy checks use **DB-stored role rank** (`signerIsIssuerOrAbove` at line 78-80), not on-chain.

---

## Solidity Contract Access Control

| Function | Min Role | Contract | Location |
|----------|----------|----------|----------|
| `initialize()` | deployer only (`_requireDeployer`) | All | All contracts |
| `batchUpdateUserRoleWithSignature(params)` | Signer Admin+ | CredentialAuthority | `CredentialAuthority.sol:140-161` |
| `transferSuperAdminWithSignature(params)` | Signer SuperAdmin | CredentialAuthority | `CredentialAuthority.sol:174-192` |
| `batchIssueCredentialsWithSignature(params)` | Issuer Issuer+ | CredentialRegistry | `CredentialRegistry.sol:134-166` |
| `batchRevokeCredentialsWithSignature(params)` | Revoker Issuer+ | CredentialRegistry | `CredentialRegistry.sol:178-201` |

Solidity `_enforceUserRoleUpdateHierarchy` (`CredentialAuthority.sol:246-260`):
- Signer below Admin → revert `RoleBelowAdminError`
- Admin cannot modify Admin+ targets → revert `AdminUpdatePeerAdminRoleError`
- Admin cannot assign Admin+ roles → revert `RoleBelowAdminError`
- Any SuperAdmin assignment → revert `SuperAdminRoleNotUpdatableError`
- Same-role update → revert `SameRoleUpdateError`

---

## Per-Role Capability Matrix

```
Capability                      None   Holder  Issuer  Admin   SuperAdmin
──                             ────   ──────  ──────  ─────   ──────────
Google Login                    —      ✓       ✓       ✓       ✓
Refresh Token                   —      ✓       ✓       ✓       ✓
Logout                          —      ✓       ✓       ✓       ✓
View own profile                —      ✓       ✓       ✓       ✓
Update own phone                —      ✓       ✓       ✓       ✓
Update own email                —      ✓       ✓       ✓       ✓
List own credentials             —      ✓       ✓       ✓       ✓
Find own credential              —      ✓       ✓       ✓       ✓
──
Transfer SuperAdmin             —      —       —       —       ✓
──
List users (read-only)          —      —       ✓       ✓       ✓
Find user by ID (read-only)     —      —       ✓       ✓       ✓
Create users (batch)            —      —       —       ✓¹      ✓²
Update users (batch)            —      —       —       ✓³      ✓⁶
Update user roles               —      —       —       ✓⁴      ✓²
Delete users                    —      —       —       ✓⁵      ✓
──
List credentials                —      —       ✓       ✓       ✓
Find credential                 —      —       ✓       ✓       ✓
Issue credentials               —      —       ✓       ✓       ✓
Revoke credentials              —      —       ✓       ✓       ✓
Re-extract credential           —      —       ✓       ✓       ✓
Verify credential (public)      ✓      ✓       ✓       ✓       ✓
```

**Key:**
- ✓ = Allowed
- — = Denied

**Notes:**
1. Admin can create Holder, Issuer only; **cannot** create Admin or SuperAdmin
2. SuperAdmin cannot create/assign SuperAdmin via batch — only via Transfer SuperAdmin flow
3. Admin can update Holder, Issuer; **cannot** update Admin, SuperAdmin, self (via batch), or trashed users; **cannot** promote to Admin
4. Admin can update Holder/Issuer among Holder/Issuer roles; **cannot** target self, Admin peers, SuperAdmin; **cannot** assign Admin role
5. Admin can delete Holder, Issuer; **cannot** delete Admin, SuperAdmin, or self
6. **SuperAdmin can update self via batch (profile fields only — name, number, phone, birth_date, gender, meta, role).** Email cannot be changed via batch even by SuperAdmin — must use `PUT /api/users/self/email` (Google reauth required) to prevent locking out with an inaccessible email.

---

## Denied Operations Quick Reference

| Operation | Who Can't Do It | Why / Error Code |
|-----------|----------------|------------------|
| Create SuperAdmin via API | Everyone (incl. SuperAdmin) | `CodeUserStoreSuperAdminForbidden` (300245) — CLI only |
| Assign SuperAdmin via batch | Everyone | `CodeUserRoleSuperAdminBatchForbidden` (300544) |
| Admin creates another Admin | Admin signer | `CodeUserStoreAdminCreateAdminForbidden` (300246) |
| Admin updates another Admin | Admin signer | `CodeUserUpdatePeerAdminForbidden` (300842) |
| Admin promotes to Admin | Admin signer | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |
| Admin deletes Admin/SuperAdmin | Admin signer | `CodeUserDeleteAdminForbidden` (300741) |
| Update self via batch | Admin (SuperAdmin allowed) | `CodeUserUpdateSelfForbidden` (300844) |
| Change own email via batch | Anyone (incl. SuperAdmin) | `CodeUserUpdateSelfEmailForbidden` (300845) — use `/users/self/email` |
| Self-delete | Anyone | `CodeUserDeleteSelfTargetForbidden` (300743) |
| Self-target role update | Anyone | `CodeUserRoleSelfTargetForbidden` (300546) |
| Transfer SuperAdmin to self | SuperAdmin | `CodeUserTransferSuperAdminSelfTargetForbidden` (300641) |
| Update trashed user | Admin+ | `CodeUserUpdateTrashedForbidden` (300846) |
| Update trashed user's role | Admin+ | `CodeUserRoleTrashedForbidden` (300547) |
| Update SuperAdmin's profile | Admin+ (SuperAdmin can self-edit; cannot change own email via batch) | `CodeUserUpdateSuperAdminForbidden` (300843) |
| Same-role update | Admin+ | `CodeUserRoleSameRoleUpdateForbidden` (300543) |
| Below-Admin accesses Admin+ endpoint | Holder, Issuer | `CodeUserRoleSignerAdminRequiredForbidden` (300542) / `CodeAuthForbidden` (200142) |

---

## Special Flows

### SuperAdmin Creation

**Only via CLI (`make init-super-admin`).** Source: `cmd/init_super_admin.go`

Pre-conditions:
1. Wallet must have `SuperAdmin` role on-chain in CredentialAuthority contract
2. No existing (live) SuperAdmin in database (`userRepo.FindByRole(ctx, RoleSuperAdmin)`, filtering out soft-deleted)

### Transfer SuperAdmin

**Only endpoint that changes SuperAdmin role.** Source: `feature/user/user_service.go:437-488`

Endpoint: `POST /api/users/self/transfer-super-admin`

Flow:
1. SuperAdmin calls endpoint targeting another user
2. Policy blocks self-transfer (`CodeUserTransferSuperAdminSelfTargetForbidden`)
3. On-chain: `authorityService.TransferSuperAdmin()` — atomic swap (signer→Admin, target→SuperAdmin)
4. DB: both users' roles updated
5. Refresh tokens revoked for both users

### On-Chain Role Sync

**`syncBlockchainRoles`** (`feature/user/user_service.go:152-159`) called by all mutation paths:
- `Store` → `storeUsersAndSyncBlockchainRoles`
- `UpdateRole` → `updateRoleAndSyncBlockchainRoles`
- `Update` (when role changes) → `syncBlockchainRoles`
- `Delete` → `deleteUserAndSyncBlockchain` (sends `RoleNone`, the chain-only revocation target)

Each packs users into `UserRoleUpdation` structs, signs with auth user's wallet, calls `batchUpdateUserRoleWithSignature` on CredentialAuthority, waits for receipt via `bind.WaitMined`, and verifies receipt status. Chain failure rolls back the DB transaction.

### RoleNone (On-Chain Revocation Only)

`RoleNone` (`domain/user.go:14`, value `"none"` → Solidity `0`) is **never persisted** in Postgres (the `role` ENUM excludes `none`). It exists solely as the Solidity revocation target used by `Delete` to revoke the user's on-chain role.

---

## Architecture: Dual Role Sources

Roles are stored in two places, kept in sync:

| Check Location | Role Source | Used By |
|---------------|-------------|---------|
| API middleware (`middleware/auth.go`) | **On-chain** (Authority contract `userToRole`) | Route-level access control |
| Policy layer (`feature/*/policy.go`) | **Database** (Postgres `role` column) | Business rule enforcement |
| Credential policy (`credential_policy.go`) | **Database** | Store-level checks |

The middleware checks the on-chain role (via `authorityService.HasRoleOrAbove`), while policy layers check the DB-stored role rank. The `syncBlockchainRoles` utility keeps both sources in sync on every mutation. If diverged (e.g., due to a failed chain tx that DB rolled back), the middleware enforces the stricter (on-chain) role.

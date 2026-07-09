# CredChain Credential System

> **Related docs:** For the role hierarchy and general authorization rules, see [ROLE.md](ROLE.md).

## Entity Definitions

### `domain.Credential` (`domain/credential.go:37-58`)

| Field | DB Type | Purpose |
|-------|---------|---------|
| `ID` | `CHAR(26)` PK | ULID primary key |
| `HolderUserID` | `CHAR(26)` FK → `users.id` | Credential owner |
| `IssuerUserID` | `CHAR(26)` FK → `users.id` | Who issued it |
| `RevokerUserID` | `CHAR(26)` FK → `users.id`, nullable | Who revoked it |
| `Name` | `VARCHAR(256)` | Human-readable label (e.g. "Bachelor's Degree") |
| `Meta` | `JSONB` | Arbitrary metadata (institution, grade, etc.) |
| `TokenID` | `VARCHAR(256)`, unique | On-chain ERC-721 token ID (decimal string) |
| `FileHash` | `CHAR(66)` | `0x`-prefixed keccak256 of raw file bytes |
| `FileURI` | `TEXT`, nullable | Storage path (e.g. `local:///uploads/...`) |
| `ExtractStatus` | `credential_extract_status` ENUM | Async extraction job state |
| `ExtractError` | `TEXT`, nullable | Error message if extraction failed |
| `ExtractedAt` | `TIMESTAMP`, nullable | When extraction completed |
| `IssuedAt` | `TIMESTAMP` | When credential was created (default `CURRENT_TIMESTAMP`) |
| `RevokedAt` | `TIMESTAMP`, nullable | When credential was revoked |

**Sources:** Go `domain/credential.go`, Postgres migration `000001_initial_schema.up.sql:39-57`, GORM model `model/credential.go:20-43`.

Embeds `Holder`, `Issuer`, `Revoker` (`*domain.User`, `gorm:"-" json:"-"`) populated by GORM Preload when query.Includes contains the corresponding key. Never serialized to JSON — response DTO maps them explicitly.

### `response.Credential` (`response/credential.go:15-33`)

Mirrors domain entity minus embeddings. `Holder`, `Issuer`, `Revoker` are `*response.User` with `omitempty` for null-safe JSON output. Factory: `response.FromDomainCredential(c domain.Credential)`.

### `response.CredentialVerify` (`response/credential.go:74-80`)

| Field | Type | Purpose |
|-------|------|---------|
| `VerdictCode` | `int` | 6-digit domain code (400401-400412) |
| `SimilarityScore` | `*float64` | Fuzzy match score (0-1), non-nil only on fuzzy path |
| `SimilarityPercent` | `*string` | Human-readable percentage, non-nil only on fuzzy path |
| `Description` | `string` | Localized verdict description resolved via the request's i18n localizer |
| `Credential` | `*Credential` | Matched credential, if any |

### `CredentialIssuance` (service-layer input, `credential_service.go:51-58`)

| Field | Type | Source |
|-------|------|--------|
| `HolderUserID` | `string` | Multipart form field |
| `Name` | `string` | Multipart form field |
| `Meta` | `map[string]any` | Multipart form field (JSON string) |
| `Filename` | `string` | Uploaded file name |
| `MIMEType` | `string` | Content-Type header (validated against allowlist) |
| `FileBytes` | `[]byte` | Uploaded file contents (max 10 MB) |

### `CredentialIssueInput` / `CredentialIssueRequest` (`credential_request.go:21-68`)

Validator-enforced constraints: `HolderUserID` required, `Name` 1-256 chars, batch size 1-100 items. Gin does not support nested multipart structs; handler parses manually via `buildIssueItems()`.

### `CredentialRevokeRequest` / `CredentialReExtractRequest` (`credential_request.go:71-93`)

Plain JSON `{"ids": [...]}`, 1-100 items per batch.

---

## Extract Status Lifecycle

Three-value `ExtractStatus` ENUM (`credential_extract_status` in Postgres):

```
pending ──→ succeeded
  │
  └──→ failed ──→ pending (via ReExtract)
```

| Status | Go Constant | Meaning |
|--------|-------------|---------|
| `pending` | `ExtractStatusPending` | Awaiting OCR by Python River worker |
| `succeeded` | `ExtractStatusSucceeded` | Text/IDs/embedding extracted and stored in Mongo |
| `failed` | `ExtractStatusFailed` | Extraction failed; retryable via ReExtract |

**Sources:** `domain/credential.go:16-22`, migration `000001_initial_schema.up.sql:33-37`.

New credentials are created with `extract_status=pending`. On-chain issuance is synchronous (keccak256 computed immediately), but extraction (text, IDs, embedding — needed by verify's fuzzy path) requires a slow Python OCR+EmbeddingGemma round-trip via River async worker.

**ReExtract flow** (`credential_service.go:628-677`):
1. Validates all targets exist, are `failed`, and have `file_uri`
2. Resets to `pending`, clears `extract_error` via CASE batch UPDATE
3. Enqueues River jobs
4. If enqueue fails, **compensates**: stamps credential back to `failed` with `"reenqueue failed"` error, preserving previous error if present

---

## On-Chain Registration

### Token ID Derivation

**Token ID derivation** (`chain/registry_service.go:254-258`):

packed := append(issuer.Bytes(), common.LeftPadBytes(nonce.Bytes(), 32)...)
packed = append(packed, holder.Bytes()...)
packed = append(packed, []byte(hash)...)
return new(big.Int).SetBytes(crypto.Keccak256(packed))

Token ID = `uint256(keccak256(issuer || zeroPadLeft(nonce, 32) || holder || hash))`

This matches the Solidity side in `CredentialRegistry.sol:160-169`.

### On-Chain Storage

**Mapping** on CredentialRegistry:

```solidity
enum CredentialStatus { None, Issued, Revoked }

mapping(bytes32 => CredentialStatus) public credentialHashToStatus;

function getCredentialHashStatuses(
    bytes32[] calldata hashes
) external view returns (CredentialStatus[] memory);
```

- `None (0)`: No credential issued for this hash
- `Issued (1)`: Active credential on-chain
- `Revoked (2)`: Credential was revoked

**Sources:** `CredentialRegistry.sol`.

### Issue Flow (On-Chain)

RegistryService.IssueCredentials (planned):
1. Fetches nonce from CredentialRegistry for signer
2. Packs: `issuer || pad32(nonce) || (holder || hash || uri)[]`
3. Signs EIP-191 digest with signer's encrypted private key
4. Calls `batchIssueCredentialsWithSignature(params)` via relayer
5. Contract mints ERC-721 tokens and sets `credentialHashToStatus[hash] = Issued`
6. Returns token IDs (derived from issuer+nonce+holder+hash)

### Revoke Flow (On-Chain)

RegistryService.RevokeCredentials:
1. Fetches nonce from CredentialRegistry for signer
2. Packs: `revoker || pad32(nonce) || pad32(tokenId)[]`
3. Signs EIP-191 digest
4. Calls `batchRevokeCredentialsWithSignature(params)` via relayer
5. Contract sets `credentialHashToStatus[hash] = Revoked`
6. Token remains soulbound (no transfer/burn allowed — `_update()` reverts)

### FindCredentialByHash (Exact-Hash Verify Path — Planned)

Uses Postgres bridge:
1. `FindByFileHashes` queries Postgres for matching `file_hash` rows
2. Gets `token_id`s from matched rows
3. Calls `getCredentialsByIds` on-chain with those token IDs
4. Returns on-chain credential data for cross-reference

**Sources:** `chain/registry_service.go:98-108` (current impl, uses direct `tokenIdFromHash`), `chain/registry_service.go:186-244` (revoke), `chain/registry_service.go:110-184` (issue).

---

## Credential Status Enum (DB-level)

Derived from `RevokedAt`:

| Status | Condition | Meaning |
|--------|-----------|---------|
| Active | `revoked_at IS NULL` | Credential is live and verifiable |
| Revoked | `revoked_at IS NOT NULL` | Credential was revoked by an Issuer+ |

No separate DB status column — revocation is timestamp-driven. The on-chain `CredentialStatus` enum (None/Issued/Revoked) is separate and reflects the chain state rather than DB state.

---

## API Routes

**Source:** `infrastructure/http/router.go:79-109`

| Route | Method | Auth | Handler | Notes |
|---|---|---|---|---|
| `/api/credentials/verify` | POST | None (public) | `Verify` | Rate-limited by global ApiRateLimitMiddleware |
| `/api/credentials` | GET | Issuer+ (on-chain) | `Paginate` | Search, filters, sorts, includes (holder/issuer/revoker) |
| `/api/credentials/:id` | GET | Issuer+ (on-chain) | `Find` | Single credential with optional Preload |
| `/api/credentials/:id/file` | GET | Authenticated (no role gate) | `DownloadFile` | Download decrypted credential file; authorization via policy (holder OR Issuer+) |
| `/api/credentials/batch/issue` | POST | Issuer+ (on-chain) | `Issue` | Multipart form, 1-100 items |
| `/api/credentials/batch/revoke` | POST | Issuer+ (on-chain) | `Revoke` | JSON body `{"ids": [...]}` |
| `/api/credentials/batch/reextract` | POST | Issuer+ (on-chain) | `ReExtract` | JSON body `{"ids": [...]}` |
| `/api/users/self/credentials` | GET | Authenticated | `SelfPaginate` | Scoped to `holder_user_id == auth_user.id` |
| `/api/users/self/credentials/:id` | GET | Authenticated | `SelfFind` | 404 if not owned (no ID leak) |
| `/api/overview` | GET | Authenticated (no role gate) | `Get` | Role-conditional dashboard: credential_counts + recents (Holder: own, Issuer+: system-wide). Optional `?limit=N` controls recent items per category (default 5). |

**Route middleware chain:** `ErrorLoggerMiddleware` → `I18nMiddleware` → `ApiRateLimitMiddleware` → `AuthMiddleware` → `IssuerRoleMiddleware` (for credential management routes).

Credential policy checks use **DB-stored role rank**, not on-chain.

---

## Policy Rules

### Credential Policy Rules

The credential policy interface lives at `feature/credential/credential_policy.go:16-20`. It defines three methods:

| Method | Line | Purpose |
|---|---|---|
| `IssuePostFetch` | 30 | Validates credentials after fetch for issue; no-op (role enforcement is via `IssuerRoleMiddleware`, on-chain) |
| `RevokePostFetch` | 34 | Validates credentials after fetch for revoke; no-op (role enforcement is via `IssuerRoleMiddleware`, on-chain) |
| `DownloadFilePreFetch` | 38 | Enforces that only issuers and above can download credential files |

Role enforcement for issue/revoke/reextract is done at the **route level** by `IssuerRoleMiddleware` (on-chain check), not by credential policy. The route-level guard is applied in `router.go:112-116`.

There is no `IssuePreFetch`, `RevokePreFetch`, `VerifyPreFetch`, or `ReExtractPreFetch` method. The verify route is public (no auth middleware — `router.go:88`).

---

## Duplicate Hash Rules

### Global File Hash Uniqueness

| Scenario | Rule |
|---|---|
| Same hash, any holder, active | **Blocked** — `CredentialStatus.Issued` on-chain, DB unique index |
| Same hash, was revoked | **Allowed** — re-issue after revocation (any holder) |

**Enforcement flow:**
1. Go computes file hash
2. Go calls `getCredentialHashStatuses([hash])` on-chain
3. If status is `Issued` → return duplicate error (`CodeCredentialIssueDuplicateFileHash` 400242)
4. If status is `None` or `Revoked` → proceed

**Sources:** Design discussion; `credential_service.go:216-221` (current Go-side check that will be removed), `credential_service.go:239-241` (current per-batch claimedHash that will be kept for same-batch dedup).

---

## Batch Flows

### Issue (`credential_service.go:187-342`)

Architecture: sync chain, async embeddings.

1. Role enforcement via `IssuerRoleMiddleware` (route-level, on-chain check) — no policy gate
2. `issueValidate` — pre-computed holder lookup + on-chain `GetCredentialHashStatuses` batch → holder existence + duplicate file hash checks
3. `issuePrepareCredentials` — encrypt files, persist to storage, build domain entities with `extract_status=pending`
4. `issueCommit` (within UoW): `Store` → check `file_uri` invariant → `syncBlockchainIssue` → `Update` token IDs → enqueue River extraction jobs
5. Chain failure rolls back DB transaction; orphan files cleaned up via `issueCleanupOrphanFiles`

**All-or-nothing:** Any per-item failure (validation, chain sync, storage, hash computation) aborts the entire batch inside the UoW — no partial results are returned. The handler returns a single error code via `responder.SendError`.

**File cleanup:** `cleanupOrphanFiles` deletes stored files on validation/chain failure — best-effort (log-and-continue).

**Invariant:** Every stored credential with `file_uri == nil` triggers `CodeCredentialIssueStorageFailed` (400245) BEFORE on-chain mint — prevents orphaned NFT.

### Revoke (`credential_service.go:362-427`)

1. Role enforcement via `IssuerRoleMiddleware` (route-level, on-chain check) — no policy gate
2. UoW: `FindByIds` targets → validate all found → check none already revoked → `RevokePostFetch` → CASE batch UPDATE (revoked_at, revoker_user_id) → `syncBlockchainRevoke` with decimal token IDs
3. Already-revoked check: `CodeCredentialRevokeAlreadyRevoked` (400342)
4. Missing targets: `CodeCredentialRevokeNotFound` (400341)
5. Token IDs with non-nil value collected for on-chain sync

### ReExtract (`credential_service.go:628-677`)

1. Role enforcement via `IssuerRoleMiddleware` (route-level, on-chain check) — no policy gate
2. UoW: `FindByIds` targets → validate all exist + are failed + have file_uri → CASE batch UPDATE (extract_status=pending, extract_error="") → enqueue River jobs
3. On enqueue failure: **compensate** — stamp back to failed with `"reenqueue failed"` error

---

## Verify Pipeline (`credential_service.go:433-539`)

Three-stage pipeline: **Cache → Exact Hash → Fuzzy**.

### Stage 1: Cache Lookup
- Compute `uploadedHash = "0x" + hex.EncodeToString(keccak256(file.Data))`
- Query MongoDB `credential_verifications` by `uploaded_file_hash`
- On cache hit: re-check holder/issuer deleted status **live** (users table) for party-disabled override (400410-400412)
- Return cached verdict + matched credential (fetched fresh by ID)
- Cache TTL: configurable via `AI_VERIFICATION_CACHE_TTL_HOURS` (default 24h)

### Stage 2: Exact Hash Path
- `FindByFileHashes([uploadedHash])` in Postgres
- **Planned:** Uses Postgres bridge → gets token_ids → calls `getCredentialsByIds` on-chain
- Cross-reference with on-chain via `registryService.FindCredentialByHash`
- Determines base verdict:
  - Hash on-chain, not revoked → `Authentic` (400401)
  - Hash on-chain, DB revoked → `Revoked` (400402)
  - Hash NOT on-chain → `IntegrityWarning` (400403, HTTP 409)
- Applies party-disabled override for Authentic only
- Caches result, returns

### Stage 3: Fuzzy Path
- Call Python AI `/extract` (IDs only — text+embedding already stored)
- If no IDs extracted → `NoIdentifiers` (400408), cache
- `FindRankedByIds` in Mongo: aggregation pipeline ranks extractions by ID intersection count (cap 10)
- If no matches → `NoMatch` (400409), cache
- `verifyPickBestMatch`: ties broken by revocation status (prefer non-revoked) → newer `IssuedAt`
- Call Python AI `/verify` with best match's embedding
- Map AI verdict string → domain code:
  - `"tampered"` → `Tampered` (400404)
  - `"suspicious"` → `Suspicious` (400405)
  - `"low_similarity"` → `LowSimilarity` (400406)
  - default → `NotSimilar` (400407)
- Apply party-disabled override for Authentic only
- Cache result with similarity score/percent, return

### Party-Disabled Override

Applied to `Authentic` verdicts only. Stronger verdicts (Revoked, Tampered, IntegrityWarning) persist unchanged.

| Condition | Override Code |
|-----------|---------------|
| Holder soft-deleted, Issuer live | `HolderDisabled` (400410) |
| Issuer soft-deleted, Holder live | `IssuerDisabled` (400411) |
| Both soft-deleted | `PartyDisabled` (400412) |

Re-checked **live on every call** including cache hits — cache stores only credential-level verdict; holder/issuer status is always fresh.

### Verdict HTTP Status Codes

| Verdict | HTTP Status |
|---------|-------------|
| All verdicts except IntegrityWarning | 200 |
| `IntegrityWarning` (400403) | 409 |

---

## Storage Architecture

### PostgreSQL (GORM) — `credentials` table

Primary store for credential metadata, file hash, token ID, status flags. Full migration at `000001_initial_schema.up.sql:39-63`.

```sql
CREATE INDEX idx_credentials_holder_user_id ON credentials(holder_user_id);
CREATE INDEX idx_credentials_issuer_user_id ON credentials(issuer_user_id);
CREATE INDEX idx_credentials_revoked_at     ON credentials(revoked_at);
CREATE INDEX idx_credentials_extract_status ON credentials(extract_status);
CREATE INDEX idx_credentials_file_hash      ON credentials(file_hash);
```

**Repository methods** (`gorm_credential_repository.go`):
- `Get`: pagination with search (name, meta TEXT, holder name/email/number/phone), filters (name/issued_at/revoked_at/holder_user_id), sorts (name/issued_at/revoked_at + holder_name/email/number/phone), Preload (holder/issuer/revoker)
- `Find`: single row by ID with optional Preload
- `FindByIds`: batch lookup, single IN-clause
- `FindByHolderId`: scoped to one holder
- `FindByFileHashes`: batch dup-check, single IN-clause
- `Store`: batch insert with ULID generation
- `Update`: CASE-based batch UPDATE (single SQL statement regardless of batch size)

### MongoDB — `credential_extractions`

Lightweight collection: `credential_id` (unique), `file_hash`, text, `ids[]`, `embedding[]` (768 floats), `created_at`, `updated_at`.

Used by fuzzy verify path only. Repository: `domain.CredentialExtractionRepository` (`credential_extraction.go:32-42`).

Searchable by `ids.value` via aggregation pipeline (`FindRankedByIds`).

### MongoDB — `credential_verifications`

TTL-bounded verify result cache. Keyed by `uploaded_file_hash`. Fields: `verdict_code`, `matched_credential_id`, `similarity_score`, `similarity_percent`, `created_at`.

TTL enforced via `created_at` MongoDB TTL index (default 24h, configurable via `AI_VERIFICATION_CACHE_TTL_HOURS`).

Repository: `domain.CredentialVerificationRepository` (`credential_verification.go:22-28`).

### Local File Storage (IPFS-compatible interface)

Persisted via `storage.Storage`. Storage base path configured by `STORAGE_PATH` (default `"uploads"`). Credential files stored under `{STORAGE_PATH}/{CREDENTIAL_FILE_STORAGE_PATH}/{filename}` where `CREDENTIAL_FILE_STORAGE_PATH` defaults to `"credentials"`. The DB `file_uri` field stores only the filename — subdirectory is reconstructed at read time.

Files are encrypted at rest with AES-256-GCM using `FILE_ENCRYPTION_KEY`. The file hash (keccak256) is computed from the **original plaintext** before encryption, so on-chain fingerprints always represent the original document.

**File URI format:** `ULID.ext` (e.g. `01JQNXYZ...pdf`). Full on-disk path: `uploads/credentials/01JQNXYZ...pdf`.

Extendable to IPFS via `storage.Storage` interface.

---

## Error Codes

### Credential Fetch (40-01)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400100 | `CodeCredentialFetchSuccess` | 200 | Credential(s) fetched |
| 400140 | `CodeCredentialFetchNotFound` | 404 | Credential not found (or not owned, self-find) |
| 400141 | `CodeCredentialFetchValidation` | 400 | Invalid query parameters |

### Credential Issue (40-02)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400200 | `CodeCredentialIssueSuccess` | 200 | All credentials issued |
| 400241 | `CodeCredentialIssueValidation` | 400 | Validation error (invalid MIME, file too large) |
| 400242 | `CodeCredentialIssueDuplicateFileHash` | 409 | Duplicate file hash (holder already has active credential) |
| 400243 | `CodeCredentialIssueHolderNotFound` | 400 | Target holder user not found |
| 400244 | `CodeCredentialIssueBlockchainSyncFailed` | 500 | On-chain mint failed |
| 400245 | `CodeCredentialIssueStorageFailed` | 500 | File storage failed (or empty path, or missing file_uri) |
| 400246 | `CodeCredentialIssueHashFailed` | 500 | Hash computation failed |

### Credential Revoke (40-03)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400300 | `CodeCredentialRevokeSuccess` | 200 | Credentials revoked |
| 400340 | `CodeCredentialRevokeFailed` | 500 | General revoke failure |
| 400341 | `CodeCredentialRevokeNotFound` | 404 | One or more credential IDs not found |
| 400342 | `CodeCredentialRevokeAlreadyRevoked` | 409 | One or more credentials already revoked |
| 400343 | `CodeCredentialRevokeBlockchainSyncFailed` | 500 | On-chain revocation failed |

### Credential Verify (40-04)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400400 | `CodeCredentialVerifySuccess` | 200 | Verify succeeded (unused — verdict codes used instead) |
| 400440 | `CodeCredentialVerifyFailed` | 500 | General verify failure |
| 400441 | `CodeCredentialVerifyValidation` | 400 | Invalid input (bad file, wrong MIME, too large) |
| 400442 | `CodeCredentialVerifyExtractNotReady` | 503 | Extraction not yet complete |
| 400443 | `CodeCredentialVerifyExtractFailed` | 500 | Extraction previously failed |
| 400444 | `CodeCredentialVerifyAiServiceFailed` | 502 | Python AI service unreachable/errored |
| 400445 | `CodeCredentialVerifyCredentialNotFound` | 404 | Matched credential not found in DB |

#### Verdict Codes (400401-400412)

| Code | Constant | HTTP | Stage | Meaning |
|------|----------|------|-------|---------|
| 400401 | `CodeCredentialVerifyAuthentic` | 200 | Exact/Fuzzy | Credential matched and on-chain |
| 400402 | `CodeCredentialVerifyRevoked` | 200 | Exact | Hash matched but revoked on-chain |
| 400403 | `CodeCredentialVerifyIntegrityWarning` | 409 | Exact | Hash in DB but NOT on-chain |
| 400404 | `CodeCredentialVerifyTampered` | 200 | Fuzzy | AI detected tampering |
| 400405 | `CodeCredentialVerifySuspicious` | 200 | Fuzzy | AI flagged as suspicious |
| 400406 | `CodeCredentialVerifyLowSimilarity` | 200 | Fuzzy | Similar but below threshold |
| 400407 | `CodeCredentialVerifyNotSimilar` | 200 | Fuzzy | No fuzzy match |
| 400408 | `CodeCredentialVerifyNoIdentifiers` | 200 | Fuzzy | AI could not extract any identifiers |
| 400409 | `CodeCredentialVerifyNoMatch` | 200 | Fuzzy | Identifiers extracted but no database match |
| 400410 | `CodeCredentialVerifyHolderDisabled` | 200 | Override | Authentic but holder soft-deleted |
| 400411 | `CodeCredentialVerifyIssuerDisabled` | 200 | Override | Authentic but issuer soft-deleted |
| 400412 | `CodeCredentialVerifyPartyDisabled` | 200 | Override | Authentic but both parties soft-deleted |

Verdict codes (400401-400412) deliberately avoid CC 01-12 for other credential codes — these are success outcomes, not errors.

### Credential Re-Extract (40-05)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400500 | `CodeCredentialReExtractSuccess` | 200 | Re-extraction queued |
| 400540 | `CodeCredentialReExtractNotFound` | 404 | One or more credential IDs not found |
| 400541 | `CodeCredentialReExtractNotEligible` | 409 | One or more credentials not in failed state (or missing file_uri) |

### Credential File Download (40-06)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400600 | `CodeCredentialFileDownloadSuccess` | 200 | File downloaded |
| 400640 | `CodeCredentialFileDownloadNotFound` | 404 | Credential not found |
| 400641 | `CodeCredentialFileDownloadForbidden` | 403 | Not authorized (not holder, not Issuer+) |
| 400642 | `CodeCredentialFileDownloadDecryptionFailed` | 500 | File decryption error |
| 400643 | `CodeCredentialFileDownloadNoFile` | 404 | Credential has no stored file |

---

## Allowed MIME Types

Defined in `credential_request.go:10-16`:

```go
var allowedMIMETypes = map[string]bool{
    "application/pdf": true,
    "image/jpeg":      true,
    "image/png":       true,
    "image/webp":      true,
    "image/tiff":      true,
}
```

Max file size: 10 MB (`maxFileBytes = 10 * 1024 * 1024`). Validation enforced at handler level for both issue and verify.

---

## Architecture: Async Extraction Model

```
Issue (sync chain, async embeddings)
  │
  ├── 1. Compute hash (Go, sync)
  ├── 2. Store file (Go, sync)
  ├── 3. DB INSERT (Go, sync)
  ├── 4. On-chain mint (Go → Registry, sync)
  ├── 5. DB UPDATE token_id (Go, sync)
  └── 6. Enqueue River job (Go → River/PG, async)
         │
         └── River worker (async)
              ├── 1. Call Python /extract (text, IDs, embedding)
              ├── 2. Store in Mongo credential_extractions
              └── 3. DB UPDATE extract_status = succeeded/failed
```

River jobs live in Postgres (`river_jobs` table) but use a separate `pgx` connection pool from GORM's (`database/sql` + `pgx`). They cannot share the GORM transaction. Mitigation: credentials stay in `pending` and ReExtract recovers failures.

---

## Cross-Repo Integration

- **`../CredChain_Solidity/CredentialRegistry.sol`** — ERC-721 soulbound token, `credentialHashToStatus` mapping, CredentialStatus enum. Used by `chain.RegistryService` via abigen bindings.
- **`../CredChain_Python/`** — AI service called via HTTP for `/extract` and `/verify` endpoints. Response envelope matches Go's `{code, message, data, errors}`. Python owns error category `50`.
- **`../CredChain_React/`** — sole HTTP consumer. Mirrors `domain.Code*` verdict constants in `@shared/api/codes.ts`. Locale keys mirrored in `src/shared/i18n/`.

**Response code format:** 6-digit `AABBCC`. `40` credential category shared across all repos. Python AI errors propagate with original `50xxxx` code.

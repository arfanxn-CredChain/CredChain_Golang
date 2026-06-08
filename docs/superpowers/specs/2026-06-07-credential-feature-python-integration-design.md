# Credential Feature Update — Go ↔ Python AI Integration

**Date:** 2026-06-07
**Status:** Approved design (pre-implementation)
**Scope:** `CredChain_Golang` credential feature — `/batch/issue`, `/batch/revoke`, `/verify`, plus a new `/batch/reextract` admin endpoint. Introduces MongoDB-backed extraction storage and a verification cache.

---

## Overview

The Go credential feature is out of sync with the current `CredChain_Python` AI service contract, and the verification flow must be redesigned. This work has three thrusts:

1. **Wire-contract repair** — the Go `pyai` client no longer matches the Python service (auth header, field names, request/response shapes, error codes). These break issue/verify at runtime today.
2. **Re-home extraction data to MongoDB** — move the heavy, searchable extraction payload (`text`, `ids`, `embedding`, `file_hash`) out of the Postgres `credentials.embeddings` column into a new `credential_extractions` Mongo collection. Postgres keeps only lifecycle fields (`extract_status`, `extract_error`, `extracted_at`).
3. **New verification flow** — `/verify` becomes a multi-stage pipeline: cache lookup → exact-hash match (with on-chain cross-check) → fuzzy ID-based match → similarity verdict. All verdicts are cached in a TTL-bounded `credential_verifications` Mongo collection.

A new `/batch/reextract` endpoint lets admins re-enqueue extraction for credentials whose extract jobs permanently failed.

**Non-goals (explicitly out of scope):**
- Credential pagination filter/sort/include/search improvements (separate future plan).
- Notification feature (future; partial-success issue is designed to enable it later).
- Env-gated Mongo integration tests (mocks only for now, matching repo convention).
- Editing abigen-generated bindings or Solidity contracts.

---

## Section 1: Wire-Contract Repair (Mandatory)

The current `infrastructure/ai/pyai/client.go` is broken against the live Python service. These fixes are required before any feature work.

| Issue | Current Go | Current Python | Fix |
|---|---|---|---|
| Auth | No `X-API-Key` header | Requires `X-API-Key` on all POST | Add header from config |
| `/extract` response | Reads `raw_text`, `embeddings` | Returns `text`, `embedding`, `ids` | Update struct tags; add `ids` |
| `/verify` request | Sends `metadata` field with `[{"stored_embeddings":[...]}]` | Expects `embeddings` field with `[[...]]` | Rename field; send flat array-of-arrays |
| `/verify` response | Reads `description` object | Returns `descriptions` map | Rename field |
| All-failed code | Checks `500140` | Python returns `500150` (`GEMINI_FAILED`) | Update check |

### Config

- New field `AIServiceAPIKey *string` in `config.Config` (pointer type, per convention).
- Env var `AI_SERVICE_API_KEY` added to `.env`, `.env.docker`, `.env.example`.
- No fatal check (mirrors `GEMINI_API_KEY` treatment); if empty, the header is omitted and Python auth must be disabled.

### `pyai` client changes (`infrastructure/ai/pyai/client.go`)

- `NewPythonAIClient` reads `cfg.AIServiceAPIKey`; when non-nil/non-empty, every request sets header `X-API-Key`.
- `extractData` struct tags: `text` (was `raw_text`), `embedding` (was `embeddings`), plus new `ids []ExtractedID` where `ExtractedID{ Type string json:"type"; Value string json:"value" }`.
- `PythonExtractResult` gains `IDs []ExtractedID` and `Text string` (rename from `RawText`).
- `/verify` request: send form field `embeddings` containing a JSON array-of-arrays (one stored embedding per file). For the single-file verify path, that is `[[...]]`.
- `verifyData` struct: read `descriptions map[string]string` (was `description` object). `VerifyDescription` becomes a `map[string]string` (`{en, id}`) or is dropped in favor of the verdict-code response — see Section 5 (descriptions no longer surfaced; verdict message comes from i18n).
- All-failed detection: check Python code `500150` (extract) instead of `500140`.
- Stale comments referencing LaBSE updated to EmbeddingGemma.

### Stale-comment cleanup

- Update doc comments in `domain/credential.go`, `pyai/client.go`, and the worker that reference "LaBSE" → "EmbeddingGemma", and "embeddings column" → "Mongo extraction doc".

---

## Section 2: MongoDB Infrastructure (New)

MongoDB is wired in config (`MongoURI`) and the driver is installed (`go.mongodb.org/mongo-driver/v2`), but **no Go code uses it yet**. This section introduces the first Mongo-backed repositories, following DDD layering (domain interface → infrastructure impl → FX provider).

### New collections

**1. `credential_extractions`** — the extraction payload moved out of Postgres.

```
{
  _id,                         // Mongo ObjectID
  credential_id: string,       // ULID — links to Postgres credentials.id
  file_hash: string,           // keccak256, indexed
  text: string,                // extracted document text (Python "text")
  ids: [{ type, value }],      // extracted identifiers
  embedding: [float64],        // EmbeddingGemma vector
  created_at, updated_at
}
```

Indexes: `credential_id` (unique), `file_hash`, `ids.value` (multikey, for the id-search aggregation).

**2. `credential_verifications`** — the verify result cache.

```
{
  _id,
  uploaded_file_hash: string,      // keccak256 of the uploaded file, unique
  verdict_code: int,               // domain code (e.g. 400201)
  matched_credential_id: *string,  // ULID of matched credential (nil for no-match/no-ids)
  similarity_score: *float64,
  similarity_percent: *string,
  created_at: time                 // TTL index → auto-expire
}
```

Indexes: `uploaded_file_hash` (unique), `created_at` (TTL, `expireAfterSeconds`).

**TTL index explained:** MongoDB runs a background thread that auto-deletes documents once `now - created_at > expireAfterSeconds`. This is how the verify cache self-heals: a cached `no_match` (or any verdict) is evicted after the TTL window, so the next upload re-runs fresh — picking up credentials issued after the original verification. No cleanup cron needed. Mongo TTL granularity is **seconds**, so the Go code converts the human-friendly hours env var to seconds (`hours * 3600`) before building the index spec.

### New files

- `infrastructure/database/mongo/client.go` — `*mongo.Client` + `*mongo.Database` FX providers (reads `MongoURI`, plus a new `MONGO_DATABASE` env var for the DB name).
- `domain/credential_extraction.go` — `CredentialExtraction` entity + `CredentialExtractionRepository` interface + `ExtractedID` value type.
- `domain/credential_verification.go` — `CredentialVerification` entity + `CredentialVerificationRepository` interface.
- `feature/credential/mongo_credential_extraction_repository.go` — impl (unexported struct, exported factory).
- `feature/credential/mongo_credential_verification_repository.go` — impl.

### Repository contracts

`CredentialExtractionRepository`:
- `Store(ctx, extraction) error` — upsert by `credential_id`.
- `FindByCredentialId(ctx, credentialID) (*CredentialExtraction, error)`.
- `FindRankedByIds(ctx, values []string, limit int) ([]CredentialExtraction, error)` — **single aggregation pipeline**: `$match` on `ids.value $in values` → `$addFields matchCount = $size($setIntersection(...))` → `$sort` by `matchCount` desc → `$limit`. Returns full docs (including `embedding`) so the verify flow needs no second round-trip for the matched embedding. **No per-id queries.**

`CredentialVerificationRepository`:
- `FindByUploadedFileHash(ctx, hash) (*CredentialVerification, error)`.
- `Store(ctx, verification) error` — upsert by `uploaded_file_hash`.

Repositories return raw errors; the service layer translates to domain codes (mirrors GORM repo convention).

### Config

- `MongoDatabase *string` — env `MONGO_DATABASE` (default e.g. `credchain`).
- `AIVerificationCacheTTLHours *int` — env `AI_VERIFICATION_CACHE_TTL_HOURS` (default 24).

### Mongo migration (new CLI command + Makefile targets)

golang-migrate is Postgres-only here, so Mongo gets its own idempotent migration mechanism mirroring the existing `cmd/migrate.go` pattern:

- New `cmd/migrate_mongo.go` — Cobra command `migrate-mongo` with inline `migrateMongoUp` / `migrateMongoDown` private helpers (naming convention: command-name prefix).
- **Up:** create collections + indexes idempotently. `credential_extractions` (`credential_id` unique, `file_hash`, `ids.value` multikey); `credential_verifications` (`uploaded_file_hash` unique, `created_at` TTL with `expireAfterSeconds = AI_VERIFICATION_CACHE_TTL_HOURS * 3600`).
- **Down:** drop both collections (+ their indexes).
- Makefile targets `make migrate-mongo` and `make migrate-mongo-down` (both honor `ENV=.env.docker`).
- `docker-fresh` and setup docs updated to run `migrate-mongo` alongside `migrate-up`.

---

## Section 3: `/batch/issue` Redesign (Partial Success)

The key change: **partial success** — a batch upload may partially succeed, and the response mirrors Python's envelope with per-item granularity. The chain write itself remains all-or-nothing for whatever survived validation.

### Pipeline

Per-item pre-chain:
1. Hash file (`keccak256`) → `file_hash`
2. Validate MIME + size (handler layer, already exists).
3. Save file to storage → `file_uri`
4. Validate holder exists, no duplicate active `file_hash`

For items that pass validation:
5. `IssuePostFetch` policy
6. INSERT Postgres rows (`extract_status=pending`), URI field = credential ULID on-chain
7. Chain issue via `RegistryService.IssueCredentials` → token IDs
8. UPDATE Postgres `token_id`
9. Enqueue one `credential_extract_jobs` row per credential

### Response shape

The response array aligns positionally with the input. Failed items = `null`; success = credential DTO. Errors keyed by `credentials.<index>`.

```
{
  "code": 400200,
  "message": "2/3 credentials issued",
  "data": [null, {credential_2}, {credential_3}],
  "errors": {"credentials.0": ["holder not found"]}
}
```

- **Top-level code**: success if ≥1 issued, error if all failed.
- **Errors**: `credentials.<index>` notation (not `items`).
- **Partial-success scope**: Pre-chain validation failures (bad holder, dup hash, storage) drop individual items; survivors proceed. The chain tx is atomic — if it reverts, all survivors fail together. This is a blockchain constraint, not a design choice.

### Postgres schema change

Directly edit `infrastructure/database/migrations/000001_initial_schema.up.sql`:
- Remove line `embeddings JSONB,` (column `embeddings`) — it moves to Mongo.
- Keep: `extract_status`, `extract_error`, `extracted_at`, `file_hash`.

**No new migration file** — just update the existing `up.sql` and keep `down.sql` in sync (add the column back). The existing `extract_status` ENUM values (`pending`, `succeeded`, `failed`) are unchanged.

### Worker change

`infrastructure/jobs/credential_extract_worker.go`:
- Call Python `/extract` → get `{text, ids, embedding}`.
- Write a **`credential_extractions` Mongo doc** (`credential_id`, `file_hash`, `text`, `ids`, `embedding`).
- Update **Postgres** lifecycle only: `extract_status=succeeded`, `extracted_at` (drop the `embeddings` column update).
- On permanent failure: `extract_status=failed` + `extract_error` (Postgres), no Mongo doc, admin can re-extract later (Section 6).

### Existing domain codes (unchanged)

Issue-related codes that remain as-is:
- `CodeCredentialIssueSuccess` (400200)
- `CodeCredentialIssueFailed` (400240)
- `CodeCredentialIssueValidation` (400241)
- `CodeCredentialIssueDuplicateFileHash` (400242)
- `CodeCredentialIssueHolderNotFound` (400243)
- `CodeCredentialIssueBlockchainSyncFailed` (400244)

New re-extract codes are defined in Section 6.


---

## Section 4: `/batch/revoke` Redesign (All-or-Nothing)

Accepts 1–100 Postgres `credentials.id` (ULIDs, not extracted IDs). Marks them revoked + by whom, syncs the chain. **Revocation is permanent — no un-revoke.**

The existing `Revoke` implementation (`credential_service.go`) is already nearly correct. Changes:

1. **Mongo + storage untouched on revoke** — revoke only mutates the `credentials` table (`revoked_at`, `revoker_user_id`). The `credential_extractions` Mongo doc and the stored file are deliberately preserved. Documented as an explicit invariant.
2. **No-unrevoke** stays via the existing `CodeCredentialRevokeAlreadyRevoked` guard.
3. **All-or-nothing** (confirmed): if any submitted ID is not found or already revoked, the whole batch fails loudly. Revocation is a high-consequence security action — failing on a bad list is safer than silent partial completion. The chain tx is atomic regardless.

Existing single-UoW flow retained: fetch targets → validate exist → validate not-already-revoked → `RevokePostFetch` policy → batch `Update` (CASE-based, no N+1) → chain `RevokeCredentials`. Chain failure rolls back the DB revoke.

---

## Section 5: `/verify` Redesign (Core)

Accepts **one file**. Returns `{code, message, data?}` where the code IS the verdict (category `40`, feature `02`). Auth: Issuer+.

### Flow

```
1. VerifyPreFetch policy (Issuer+)
2. Hash uploaded file → uploaded_file_hash

3. CACHE LOOKUP — Mongo credential_verifications by uploaded_file_hash
   HIT → re-join Postgres by matched_credential_id (live revoked status)
         return cached verdict + score + credential. DONE. (no Python, no Gemini)

4. EXACT-HASH PATH — Postgres FindByFileHashes(uploaded_file_hash)
   MATCH found:
     a. Chain cross-check: keccak256(hash) → registry.FindCredential(id)
        - chain mismatch/missing → verified_integrity_warning (HTTP 409)
     b. revoked_at != nil → verified_revoked (credential + revoked info)
     c. else → verified_authentic (credential)
     → CACHE this verdict, return. (no Python)

5. FUZZY PATH (no exact hash match):
   a. Python /extract-ids (uploaded file) → ids[]
      - ids empty → no_identifiers. CACHE. return.
   b. Mongo FindRankedByIds(ids, limit) → best match by intersection count
      - zero matches → no_match. CACHE. return.
      - tie-break: prefer non-revoked, then most-recent issued_at
   c. Best match = credential B (full doc incl. embedding from the same aggregation).
   d. Python /verify (uploaded file + B's stored embedding) → similarity + verdict
   e. Map similarity → verdict code (tampered/suspicious/low_similarity/not_similar)
   f. CACHE verdict (uploaded_file_hash, code, B.id, score, percent). return.
```

### Caching policy

**All verdicts are cached** (keyed by `uploaded_file_hash`), bounded by the TTL index. This bounds Gemini cost (repeated unmatched uploads within the TTL hit the cache) while the TTL handles staleness (after expiry, the file re-runs fresh and can pick up newly-issued credentials). On a cache hit, the matched credential is re-fetched from Postgres so `revoked_at` reflects live state even if the cache predates a revocation.

### Tie-breaker

When two extractions match the same number of IDs: prefer non-revoked, then most-recent `issued_at`. The aggregation returns the top-ranked candidates (e.g. all sharing the max match count); the service then resolves the tie by issuing **one** `credentialRepo.FindByIds` IN-query for the tied ULIDs to read their `revoked_at` / `issued_at`, and picks the winner in memory. This is a single batched query — not a per-candidate loop.

### Chain cross-check

On exact-hash match, compute `id = keccak256(hash)` and call `RegistryBinding.FindCredential(id)` (read method). If the on-chain record is missing or its `hash` does not match, return `verified_integrity_warning` (HTTP 409) — signals Postgres/chain divergence. Requires adding `FindCredential` to the `chain.RegistryBinding` interface in `infrastructure/chain/bindings.go` (the abigen `*contracts.Registry` already satisfies it structurally) and a wrapper method on `RegistryService`.

### N+1 guarantees

- Cache lookup: 1 Mongo query.
- Exact path: 1 Postgres query + 1 RPC (chain check) + 1 cache write.
- Fuzzy path: 1 Python `/extract-ids` + 1 Mongo aggregation (returns B's full doc incl. embedding — no second fetch) + 1 Python `/verify` + 1 cache write.
- **No per-id or per-credential loops anywhere.**

### Verdict code taxonomy (new domain codes, category 40, feature 02)

Verify is credential feature `04`; existing verify codes occupy `400400` (success), `400440`–`400445` (errors). Verdict codes use the `4004xx` success sub-range:

| Outcome | Constant | Code | HTTP | data |
|---|---|---|---|---|
| `verified_authentic` | `CodeCredentialVerifyAuthentic` | 400401 | 200 | credential |
| `verified_revoked` | `CodeCredentialVerifyRevoked` | 400402 | 200 | credential + revoked info |
| `verified_integrity_warning` | `CodeCredentialVerifyIntegrityWarning` | 400403 | 409 | credential |
| `tampered` | `CodeCredentialVerifyTampered` | 400404 | 200 | credential B + score |
| `suspicious` | `CodeCredentialVerifySuspicious` | 400405 | 200 | credential B + score |
| `low_similarity` | `CodeCredentialVerifyLowSimilarity` | 400406 | 200 | credential B + score |
| `not_similar` | `CodeCredentialVerifyNotSimilar` | 400407 | 200 | credential B + score |
| `no_identifiers` | `CodeCredentialVerifyNoIdentifiers` | 400408 | 200 | none |
| `no_match` | `CodeCredentialVerifyNoMatch` | 400409 | 200 | none |

The pre-existing generic `CodeCredentialVerifySuccess` (400400) and the `extract-not-ready`/`extract-failed` codes (400442/400443) are retained in `codes.go` but no longer emitted by the verify handler.

Existing verify error codes (validation `CodeCredentialVerifyValidation`, AI-service-failure `CodeCredentialVerifyAiServiceFailed`, not-found) are retained. The prior `extract-not-ready` / `extract-failed` gating codes are no longer used by verify (verify no longer reads the credential's own extraction status — it works off the uploaded file).

### Response DTO

`response.CredentialVerify` reworked: carries optional matched `Credential`, optional `SimilarityScore`, optional `SimilarityPercent`. The verdict text comes from the response code via i18n (no bilingual description block). `data` is omitted entirely for `no_identifiers` / `no_match`.

---

## Section 6: Admin Re-Extract Endpoint + Supporting Wiring

### New endpoint

`POST /api/credentials/batch/reextract` (Issuer+), handler `ReExtract`.

- Accepts 1–100 credential ULIDs (JSON body, mirrors the revoke request shape).
- Per credential: validate it exists, `extract_status == failed`, `file_uri != nil`.
- Reset `extract_status=pending`, clear `extract_error`, enqueue a fresh `credential_extract_jobs` row.
- **All-or-nothing** within one UoW (consistent with revoke).
- Codes: `CodeCredentialReExtractSuccess` (400300), `CodeCredentialReExtractNotFound` (400340), `CodeCredentialReExtractNotFailed` (400341 — only failed credentials may be re-extracted).

### Request DTO

`CredentialReExtractRequest { Ids []string }` with Ozzo validation: 1–100 IDs, each a non-empty ULID. Lives in `feature/credential/credential_request.go` with a `Validate()` method + request test.

### FX wiring (`cmd/server.go`)

New providers:
- `mongo.NewClient` / `mongo.NewDatabase` (Mongo client + database).
- `NewCredentialExtractionRepository`, `NewCredentialVerificationRepository`.
- Updated `pyai.NewPythonAIClient` (now reads API key from config — no signature change if it already takes `*config.Config`).

The credential service constructor (`CredentialServiceParams`) gains the two Mongo repositories. The worker (`CredentialExtractWorkerParams`) gains the extraction repository.

### Domain codes checklist

Each new code requires updating, in lockstep (tests enforce this):
- `domain/codes.go` — constant.
- `infrastructure/http/responder/mapper.go` — `CodeToMessageKey` + `HttpCodes`.
- `locales/en.json` + `locales/id.json` — message key.
- `infrastructure/http/responder/mapper_test.go` — `allDomainCodes` list.

New codes total: 9 verify verdicts (400201–400209) + 3 re-extract (400300, 400340, 400341).

### Routes (`infrastructure/http/router.go`)

| Method | Path | Auth | Handler |
|---|---|---|---|
| POST | `/api/credentials/batch/issue` | Issuer+ | `Issue` (partial success) |
| POST | `/api/credentials/batch/revoke` | Issuer+ | `Revoke` (all-or-nothing) |
| POST | `/api/credentials/batch/reextract` | Issuer+ | `ReExtract` (new) |
| POST | `/api/credentials/verify` | Issuer+ | `Verify` (redesigned) |

---

## Section 7: Testing & Verification

Per AGENTS.md conventions (white-box, in-package, `stretchr/testify`).

### Mongo testing approach: mocks only

SQLite cannot emulate Mongo, and the repo has no integration tests against external services. Therefore:
- New mocks in `infrastructure/testutil/mocks/`: `MockCredentialExtractionRepository`, `MockCredentialVerificationRepository`.
- Service-layer unit tests run against these mocks (same pattern as `MockAuthorityService`).
- Aggregation pipeline correctness (`FindRankedByIds`) is verified manually via Postman against real Mongo — **not** in the automated suite.
- No env-gated integration tests in this scope.

### Test coverage to add

- **Verify service** — one test per path: cache hit; exact authentic; exact revoked; exact integrity-warning (chain mismatch); fuzzy no-identifiers; fuzzy no-match; fuzzy each similarity verdict (tampered/suspicious/low/not-similar); tie-break (non-revoked preferred, then newest).
- **Issue service** — partial success (mix of valid + bad-holder + dup-hash), all-failed, chain-failure rollback (via `PropagatingUnitOfWork`).
- **Revoke service** — all-or-nothing failure on bad/already-revoked ID; happy path; chain-failure rollback.
- **Re-extract service** — happy path; not-found; not-failed (status guard).
- **Request DTOs** — validation tests for `CredentialReExtractRequest` (and any reshaped issue/verify DTOs).
- **pyai client** — extract response parsing (new field names + ids), verify request/response shapes, API-key header presence.
- **Locale + mapper drift** — `locale_keys_test.go` and `mapper_test.go` stay green with the new codes.

### Verification command

```
go test ./... && go vet ./... && gofmt -l .
```
(`gofmt -l .` must produce zero output.)

---

## Section 8: AGENTS.md Update (end of implementation)

At the end of the plan, update `CredChain_Golang/AGENTS.md` to reflect:
- MongoDB now actively used: `credential_extractions` + `credential_verifications` collections, `migrate-mongo` / `migrate-mongo-down` commands and Makefile targets.
- New env vars: `AI_SERVICE_API_KEY`, `MONGO_DATABASE`, `AI_VERIFICATION_CACHE_TTL_HOURS`.
- New route `POST /api/credentials/batch/reextract`.
- Verify verdict-code taxonomy (400201–400209) and re-extract codes.
- `credentials.embeddings` column removed (extraction data lives in Mongo).
- Issue is partial-success; revoke and re-extract are all-or-nothing.
- **New strict NO-N+1 rule** (explicit): all batch repository operations (Postgres and Mongo) MUST use a single query/aggregation regardless of batch size. No per-row/per-id loops issuing queries. Postgres batch updates use CASE statements; Mongo id-search uses one aggregation pipeline; relation joins use single `IN`-clause / `$in` queries. Reviewer must reject any code path that issues queries inside a loop over input items.

---

## Open Items / Future Plans (out of scope)

- Credential pagination filter/sort/include/search overhaul — separate spec.
- Notification feature consuming partial-success issue results — future.
- Env-gated Mongo integration tests for the aggregation pipeline — future.

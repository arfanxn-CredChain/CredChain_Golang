# Credential Seeding Design Spec

## Overview

Add `CredentialSeeder` and `CredentialExtractionSeeder` to CredChain_Golang. Seeders generate realistic Indonesian diploma credentials with programmatic file generation, store them across Postgres, on-chain, and MongoDB — all deterministic. Supports three MIME types: JPEG, PNG, and PDF.

## Seed Flow

1. **`make seed`** runs: `user` → `credential` → `credential_extraction` (Registry order)
2. **`make seed-chain`** runs: user chain registration → credential chain minting (extended from existing)

---

## CredentialSeeder

**Query live holders** (non-trashed, Holder+ role) and **issuers** (non-trashed, Issuer+ role) from DB. Assign credentials with ~4:1 holder-to-issuer ratio. ~20% of credentials revoked after storage.

### Per credential:
1. Pick holder and issuer deterministically
2. Select credential type (cycle through 10 Indonesian degree names)
3. Select MIME type (cycle through JPEG, PNG, PDF)
4. Build diploma content (holder info, issuer as dean, fixed rektor)
5. Render diploma in selected format:
   - **PNG**: Go `image` package + `golang.org/x/image/font`
   - **JPEG**: Render as PNG, then convert via `golang.org/x/image/jpeg`
   - **PDF**: Use `github.com/go-pdf/fpdf` to embed rendered image as background + overlay selectable text
6. Compute `keccak256` of raw bytes → `file_hash`
7. AES-256-GCM encrypt → save to `uploads/credentials/<ulid>.<ext>`
8. INSERT to Postgres (`extract_status=succeeded` — skip River job)
9. Store extraction directly to MongoDB (no River, no Python call)

### Diploma Layout (based on Indonesian Ijazah image):
- Top: University emblem + name
- "Memberikan Kepada" subtitle
- Holder name (large), birth date, birth place
- "IJAZAH" label, credential name, program studi
- Legal text: "Dengan segala hak dan kewajiban yang berhubungan dengan sebutan akademik ini."
- Bottom: Rektor (fixed) + Dekan (issuer) signatures with NIPs, university seal

### File Generation:
- Pure Go packages (`image`, `image/jpeg`, `golang.org/x/image/font`, `github.com/go-pdf/fpdf`)
- Embed emblem PNG and seal PNG as Go byte constants (generated once, committed as fixtures)
- MIME types cycle: JPEG → PNG → PDF → JPEG → ...

### Credential Names (cycled in order):
| # | Name | Short |
|---|---|---|
| 1 | Sarjana Komputer | S.Kom |
| 2 | Sarjana Akuntansi | S.Ak |
| 3 | Sarjana Hukum | S.H. |
| 4 | Sarjana Kedokteran | S.Ked. |
| 5 | Sarjana Teknik | S.T. |
| 6 | Sarjana Ekonomi | S.E. |
| 7 | Sarjana Psikologi | S.Psi. |
| 8 | Sarjana Pendidikan | S.Pd. |
| 9 | Sarjana Ilmu Komunikasi | S.I.Kom. |
| 10 | Sarjana Administrasi Bisnis | S.A.B. |

### Fallbacks for Nullable User Fields:
| Column | Fallback |
|---|---|
| `BirthDate` | `hashToSeed(holder.Id)` → deterministic date 1970-2005 |
| `Gender` | `hashToSeed(holder.Id + "gender")` → Male/Female/Other |
| `PhoneNumber` | Generated Indonesian number (DB only) |
| `Number` | NIM if Holder, NIP if Issuer (already in user seeder) |

### Fixed Institution Head:
- Name: "Prof. Dr. Ahmad Wijaya, M.Sc"
- NIP: "196803152008123456"

---

## CredentialExtractionSeeder

**Runs after CredentialSeeder.** Iterates all seeded credentials.

### Per credential:
1. Re-derive the same diploma content used during rendering
2. Build `text` — full diploma text in Indonesian
3. Build `ids[]` — 12 extraction IDs visible on diploma:
   - `credential_name`, `credential_short`, `program_studi`, `holder_name`, `birth_date`, `birth_place`, `university_head_name`, `university_head_nip`, `dean_name`, `dean_nip`, `issued_date`, `institution`
4. Generate deterministic `embedding[]` — SHA-256 of text → 768 floats
5. Upsert to MongoDB `credential_extractions`

Phone number is NOT in diploma → NOT in extraction IDs.

---

## No River Job — Direct Storage

Since we generate the file ourselves, we know exactly what content is on it. Flow:
1. Generate diploma file (JPEG/PNG/PDF) — we know all content
2. Build extraction `text` and `ids[]` from the same data used to render the file
3. Generate deterministic `embedding[]` (SHA-256 → 768 floats)
4. Save file to disk, INSERT credential to Postgres with `extract_status=succeeded`
5. Store extraction directly to MongoDB via `CredentialExtractionRepository.Store()`

No River job, no Python call, no async. Much simpler and faster.

---

## Chain Minting

Extend `cmd/seed_chain.go`:
1. Read all credentials with `token_id IS NULL` from DB
2. Derive issuer wallets from mnemonic
3. Batch-mint via `CredentialRegistry.IssueCredentials` (chunks of ≤100, respecting `MAX_BATCH_CREDENTIAL`)
4. Single batch UPDATE: `credentialRepo.Update(ctx, updates...)` — one CASE SQL, no N+1

---

## Determinism

- `hashToSeed("credchain-seed")` → seeds `math/rand` and `gofakeit`
- All values derived deterministically from seed + DB state
- Same `make seed` run = identical output every time

---

## Naming

- Variable names in English: `universityHead`, `dean`, `credentialName`
- Extraction ID keys: `dean_name`, `dean_nip`, `university_head_name`, `university_head_nip`
- Helper prefix: `seed` (existing convention)
- Function rename: `seedHashSeed` → `hashToSeed` (user_seeder.go too)

---

## Config

Pass `*config.Config` directly to seeder constructors (not individual fields).

---

## Files to Create/Modify

| Action | File | Purpose |
|---|---|---|
| Create | `infrastructure/database/seeder/credential_seeder.go` | CredentialSeeder — diploma rendering + Postgres insert |
| Create | `infrastructure/database/seeder/credential_seeder_test.go` | Tests |
| Create | `infrastructure/database/seeder/credential_extraction_seeder.go` | CredentialExtractionSeeder — MongoDB storage |
| Create | `infrastructure/database/seeder/credential_extraction_seeder_test.go` | Tests |
| Create | `infrastructure/database/seeder/diploma.go` | Diploma rendering logic (JPEG/PNG/PDF generation) |
| Create | `infrastructure/database/seeder/diploma_test.go` | Tests |
| Create | `infrastructure/database/seeder/fixtures/emblem.png` | University emblem (embedded as Go bytes) |
| Create | `infrastructure/database/seeder/fixtures/seal.png` | University seal (embedded as Go bytes) |
| Modify | `cmd/seed.go` | Wire new seeders into Registry |
| Modify | `cmd/seed_chain.go` | Add credential minting after user registration |
| Modify | `infrastructure/database/seeder/user_seeder.go` | Rename `seedHashSeed` → `hashToSeed` |
| Modify | `Makefile` | No change needed (existing targets work) |

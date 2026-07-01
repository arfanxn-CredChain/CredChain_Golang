# Scale Seeder to 150 Users — Holder-Heavy, Chunked Chain Registration

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scale the user seeder from 15 to 150 total users with an 80/20 Holder-to-Issuer tilt and 10 soft-deleted users. Chunk `seed-chain` registration to respect the Solidity contract's `MAX_BATCH_ROLE = 100` limit.

**Architecture:** The `UserSeeder.seedBuildUsers` array grows from 15 to 150 slots. The random-user loop expands from 10 to 145 iterations. The role distribution pool shifts from 60/40 (3 Holder + 2 Issuer) to 80/20 (4 Holder + 1 Issuer). Soft-deletes scale from 5 to 10 (Anna Sorokin at index 4 + 9 random users at indices 10–18). The `seed-chain` CLI command is updated to split filtered users into ≤100-user chunks and submit each as a separate on-chain tx, re-fetching the SuperAdmin signer nonce per chunk. `AuthorityService.UpdateUserRole` is left unchanged — it faithfully passes whatever the caller sends to the contract (which enforces its own ≤100 limit with a clear `MaxBatchExceededError` revert).

**Tech Stack:** Go 1.25.1, GORM, go-ethereum, testify, Cobra, Uber FX

## Global Constraints

- Go module path is `CredChain_Golang` (underscore)
- No feature branches — commit directly to master
- Canonical verification before push: `go test ./... && go vet ./... && gofmt -l .`
- Solidity `CredentialAuthority.MAX_BATCH_ROLE = 100` — on-chain role update batches must not exceed 100 per tx
- Wallet keys derived via BIP44 from Hardhat mnemonic (indices 1–150)
- All users get NIP (18-digit, Issuer+) or NIM (`2209XXXX`, Holder) numbers
- Seeder output must be deterministic (same seed produces same users)
- Soft-deleted users: `deleted_at` set, DB role preserved, on-chain role skipped on fresh deploy
- Predefined users (indices 0–4) remain unchanged in identity and role
- Tests use in-memory SQLite (no real Postgres dependency)
- Do NOT chunk in `authority_service.go` — it runs inside DB transactions in feature services; a partially-committed chain tx mid-chunk would leave DB and chain out of sync with no undo path. Chunking belongs in the bootstrap `seed-chain` CLI where no DB transaction wraps the calls.

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `infrastructure/database/seeder/user_seeder.go` | Modify | Scale to 150 users, 80/20 role pool, 10 soft-deletes |
| `infrastructure/database/seeder/user_seeder_test.go` | Modify | Update test name and count assertions |
| `cmd/seed_chain.go` | Modify | Chunk filtered users into ≤100-user batches per on-chain tx |
| `AGENTS.md` | Modify | Update two doc references (15→150) |

---

### Task 1: Scale UserSeeder to 150 Users

**Files:**
- Modify: `infrastructure/database/seeder/user_seeder.go:54`
- Modify: `infrastructure/database/seeder/user_seeder.go:97`
- Modify: `infrastructure/database/seeder/user_seeder.go:290-293`
- Modify: `infrastructure/database/seeder/user_seeder.go:40-47`

**Interfaces:**
- Consumes: `domain.UserRepository.Store`, `domain.UserRepository.Delete`, `cryptoInfra.DeriveKeyFromMnemonic`, `cryptoInfra.Encrypt`
- Produces: `UserSeeder.Seed(ctx)` — same interface, just scaled

- [ ] **Step 1: Expand users array from 15 to 150**

In `infrastructure/database/seeder/user_seeder.go`, line 54:

```go
users := make([]domain.User, 15)
```

Change to:

```go
users := make([]domain.User, 150)
```

- [ ] **Step 2: Expand random-user loop from 10 to 145**

In `infrastructure/database/seeder/user_seeder.go`, line 97:

```go
for i := range 10 {
```

Change to:

```go
for i := range 145 {
```

- [ ] **Step 3: Tilt role pool from 60/40 to 80/20 (Holder/Issuer)**

In `infrastructure/database/seeder/user_seeder.go`, lines 290-293:

```go
var seedUserRoles = []domain.Role{
	domain.RoleHolder, domain.RoleHolder, domain.RoleHolder,
	domain.RoleIssuer, domain.RoleIssuer,
}
```

Change to:

```go
var seedUserRoles = []domain.Role{
	domain.RoleHolder, domain.RoleHolder, domain.RoleHolder, domain.RoleHolder,
	domain.RoleIssuer,
}
```

- [ ] **Step 4: Scale soft-deletes from 5 to 10 (Anna + 9 random at indices 10–18)**

In `infrastructure/database/seeder/user_seeder.go`, lines 40-47:

```go
deleteIDs := make([]string, 0, 5)
deleteIDs = append(deleteIDs, users[4].Id)
for i := 10; i <= 13; i++ {
	deleteIDs = append(deleteIDs, users[i].Id)
}
```

Change to:

```go
deleteIDs := make([]string, 0, 10)
deleteIDs = append(deleteIDs, users[4].Id)
for i := 10; i <= 18; i++ {
	deleteIDs = append(deleteIDs, users[i].Id)
}
```

- [ ] **Step 5: Run existing tests to verify they still pass with larger array**

```bash
cd CredChain_Golang && go test ./infrastructure/database/seeder/... -v -count=1
```

Expected: `TestUserSeeder_Seeds15Users` FAIL (still expects 15 total and 5 deleted — will fix in Task 2). `TestUserSeeder_DeterministicRandomUsers` should PASS (deterministic output, the test only compares cross-run equality not absolute counts).

- [ ] **Step 6: Commit**

```bash
git add infrastructure/database/seeder/user_seeder.go
git commit -m "feat(seeder): scale users from 15 to 150, 80/20 Holder/Issuer tilt, 10 soft-deletes"
```

---

### Task 2: Update Seeder Tests for 150 Users

**Files:**
- Modify: `infrastructure/database/seeder/user_seeder_test.go:16`
- Modify: `infrastructure/database/seeder/user_seeder_test.go:73`
- Modify: `infrastructure/database/seeder/user_seeder_test.go:83`

**Interfaces:**
- Consumes: Updated `UserSeeder` from Task 1
- Produces: Updated assertions matching 150-user expectations

- [ ] **Step 1: Rename test function**

In `infrastructure/database/seeder/user_seeder_test.go`, line 16:

```go
func TestUserSeeder_Seeds15Users(t *testing.T) {
```

Change to:

```go
func TestUserSeeder_Seeds150Users(t *testing.T) {
```

- [ ] **Step 2: Update total user count assertion**

In `infrastructure/database/seeder/user_seeder_test.go`, line 73:

```go
assert.Equal(t, 15, total)
```

Change to:

```go
assert.Equal(t, 150, total)
```

- [ ] **Step 3: Update soft-deleted count assertion**

In `infrastructure/database/seeder/user_seeder_test.go`, line 83:

```go
assert.Equal(t, 5, deletedCount)
```

Change to:

```go
assert.Equal(t, 10, deletedCount)
```

- [ ] **Step 4: Run seeder tests to verify they pass**

```bash
cd CredChain_Golang && go test ./infrastructure/database/seeder/... -v -count=1
```

Expected: ALL tests PASS. `TestUserSeeder_Seeds150Users` passes with 150 total, 10 deleted. `TestUserSeeder_DeterministicRandomUsers` passes.

- [ ] **Step 5: Commit**

```bash
git add infrastructure/database/seeder/user_seeder_test.go
git commit -m "test(seeder): update assertions for 150 users and 10 soft-deletes"
```

---

### Task 3: Chunk Seed-Chain Registration to Respect MAX_BATCH_ROLE=100

**Files:**
- Modify: `cmd/seed_chain.go:94-145`

**Interfaces:**
- Consumes: `authorityService.UpdateUserRole(ctx, superAdminWallet, users...)`, `userRepo.Get(ctx, nil)`
- Produces: Updated `seedChainRun` that submits users in ≤100-user chunks

**Context:** The Solidity contract `CredentialAuthority` enforces `MAX_BATCH_ROLE = 100` (line 45). `batchUpdateUserRoleWithSignature` reverts with `MaxBatchExceededError` if the array exceeds 100. With 150 seeded users, after filtering (skip 1 SuperAdmin + 10 soft-deleted→RoleNone), ~139 users remain — exceeding the limit. This task wraps the `UpdateUserRole` call in a chunking loop. Nonce is re-fetched per chunk because each successful tx increments it on-chain.

**Design decision:** Chunking lives in `seed_chain.go`, not in `authority_service.go`. `AuthorityService.UpdateUserRole` is called inside DB transactions by feature-service `syncBlockchainRoles` helpers. If a chunked call partially committed chain state mid-stream and then failed, the DB would roll back but the chain would not — a data consistency bug with no recovery path. `seed-chain` is a CLI bootstrap tool with no surrounding DB transaction, so a partial chunk failure (chunk 1 succeeds, chunk 2 reverts) is recoverable: fix the contract state or filter already-registered users, then re-run.

- [ ] **Step 1: Read current seedChainRun to understand the flow**

```bash
head -n 155 CredChain_Golang/cmd/seed_chain.go
```

- [ ] **Step 2: Replace the single UpdateUserRole call with a chunking loop**

In `cmd/seed_chain.go`, find the block at lines 139-146:

```go
	logger.Info("registering users on-chain",
		zap.Int("count", len(usersToRegister)),
		zap.String("signer", superAdminWallet.Address),
	)

	if err := authorityService.UpdateUserRole(ctx, superAdminWallet, usersToRegister...); err != nil {
		return fmt.Errorf("seed-chain: on-chain registration: %w", err)
	}

	logger.Info("seed-chain completed successfully",
		zap.Int("users_registered", len(usersToRegister)),
	)
```

Replace with:

```go
	const maxBatchRole = 100
	registeredCount := 0
	for start := 0; start < len(usersToRegister); start += maxBatchRole {
		end := start + maxBatchRole
		if end > len(usersToRegister) {
			end = len(usersToRegister)
		}
		chunk := usersToRegister[start:end]

		logger.Info("registering users on-chain",
			zap.Int("chunk_size", len(chunk)),
			zap.Int("chunk_start", start),
			zap.Int("total", len(usersToRegister)),
			zap.String("signer", superAdminWallet.Address),
		)

		if err := authorityService.UpdateUserRole(ctx, superAdminWallet, chunk...); err != nil {
			return fmt.Errorf("seed-chain: on-chain registration chunk [%d:%d]: %w", start, end, err)
		}
		registeredCount += len(chunk)
	}

	logger.Info("seed-chain completed successfully",
		zap.Int("users_registered", registeredCount),
	)
```

- [ ] **Step 3: Verify Go compilation**

```bash
cd CredChain_Golang && go build ./cmd/...
```

Expected: Compiles without errors (the `cmd` package has no tests, but compilation verifies no type/syntax issues).

- [ ] **Step 4: Run full test suite (seed-chain has no dedicated tests, but verify no regressions)**

```bash
cd CredChain_Golang && go test ./... -count=1
```

Expected: ALL tests PASS. No tests directly exercise `seedChainRun`, but the compilation check and full suite ensure nothing is broken.

- [ ] **Step 5: Commit**

```bash
git add cmd/seed_chain.go
git commit -m "feat(seed-chain): chunk on-chain role registration to respect MAX_BATCH_ROLE=100"
```

---

### Task 4: Update Documentation References

**Files:**
- Modify: `AGENTS.md:29`
- Modify: `AGENTS.md:434`

- [ ] **Step 1: Update make seed comment in AGENTS.md**

In `AGENTS.md`, line 29:

```
make seed                           # Run database seeders (populate 15 users)
```

Change to:

```
make seed                           # Run database seeders (populate 150 users)
```

- [ ] **Step 2: Update Database Seeder section in AGENTS.md**

In `AGENTS.md`, line 434, the sentence:

```
The `UserSeeder` creates 15 users (5 defined + 10 randomised Indonesian names)
```

Change to:

```
The `UserSeeder` creates 150 users (5 defined + 145 randomised Indonesian names)
```

Also update the soft-delete count in the same paragraph. Find:

```
Five users are soft-deleted.
```

Change to:

```
Ten users are soft-deleted.
```

- [ ] **Step 3: Verify no other stale references**

```bash
cd CredChain_Golang && grep -rn "15 users" --include='*.md' --include='Makefile' .
```

Expected: Only references inside `docs/superpowers/plans/` (old plans — leave those) and the lines being changed in `AGENTS.md`. No remaining production references to "15 users".

- [ ] **Step 4: Commit**

```bash
git add AGENTS.md
git commit -m "docs: update seeder user count references 15→150"
```

---

### Task 5: Full Verification

**Files:** None (read-only checkpoint)

- [ ] **Step 1: Run all seeder tests**

```bash
cd CredChain_Golang && go test ./infrastructure/database/seeder/... -v -count=1
```

Expected: ALL PASS.

- [ ] **Step 2: Run full test suite**

```bash
cd CredChain_Golang && go test ./... -count=1
```

Expected: ALL PASS. No regressions.

- [ ] **Step 3: Run static analysis**

```bash
cd CredChain_Golang && go vet ./... && gofmt -l .
```

Expected: `go vet` produces zero output. `gofmt -l .` produces zero output.

- [ ] **Step 4: Commit (if needed)**

Only if `gofmt` made formatting changes:

```bash
git add -u && git commit -m "chore: apply gofmt formatting"
```

---

## Self-Review Checklist

1. **Spec coverage:**
   - [x] Scale users from 15 to 150 — Task 1
   - [x] Tilt to 80/20 Holder/Issuer — Task 1
   - [x] 10 soft-deleted users — Task 1
   - [x] Update tests — Task 2
   - [x] Handle `MAX_BATCH_ROLE = 100` Solidity constraint — Task 3 (seed-chain chunking)
   - [x] Update documentation — Task 4
   - [x] Full verification — Task 5

2. **Placeholder scan:** No TBD, TODO, or "add appropriate error handling" patterns found. All code blocks are complete.

3. **Type consistency:**
   - `seedChainRun` signature unchanged: `(cfg *config.Config, userRepo domain.UserRepository, authorityService chain.AuthorityService, names []string, logger *zap.Logger) error` ✓
   - `authorityService.UpdateUserRole` NOT modified — same interface as before ✓
   - Chunk loop uses `maxBatchRole = 100`, matching Solidity `MAX_BATCH_ROLE` ✓
   - Nonce re-fetched implicitly: each `UpdateUserRole` call fetches nonce, submits tx, waits mined, nonce increments on-chain → next call gets updated nonce ✓

4. **Architecture rationale recorded:**
   - Chunking in `seed_chain.go` (CLI bootstrap, no DB tx) — safe for partial failure recovery ✓
   - NOT chunking in `authority_service.go` (called inside DB transactions) — prevents partial-commit data inconsistency ✓

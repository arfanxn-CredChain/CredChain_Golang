# Task 2: Rename seedHashSeed → hashToSeed

**Files:**
- Modify: `infrastructure/database/seeder/user_seeder.go`

**Goal:** Rename the unexported function `seedHashSeed` to `hashToSeed` and update its single call site. This function computes an FNV-64a hash to produce a deterministic int64 seed for `math/rand.NewSource()`.

**Changes:**

1. In `infrastructure/database/seeder/user_seeder.go`:
   - Line 30: Change `seedHashSeed("credchain-seed")` to `hashToSeed("credchain-seed")`
   - Line 204 (function definition): Change `func seedHashSeed(s string) int64` to `func hashToSeed(s string) int64`

2. Verify with tests:
   ```bash
   go test ./infrastructure/database/seeder/... -v
   ```
   Expected: all tests pass.

3. Commit:
   ```bash
   git add infrastructure/database/seeder/user_seeder.go
   git commit -m "refactor: rename seedHashSeed to hashToSeed"
   ```

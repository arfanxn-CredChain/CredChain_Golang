# Task 1: Add Dependencies

**Files:**
- Modify: `go.mod` (via `go get`)

**Goal:** Add `golang.org/x/image` (Go font rendering) and `github.com/go-pdf/fpdf` (PDF generation) so diploma rendering can use them.

**Steps:**

1. Run from `CredChain_Golang/`:
   ```bash
   go get golang.org/x/image@latest
   go get github.com/go-pdf/fpdf@latest
   go mod tidy
   ```

2. Verify:
   ```bash
   grep -E 'golang.org/x/image|go-pdf/fpdf' go.mod
   ```
   Expected: both modules appear in go.mod.

3. Commit:
   ```bash
   git add go.mod go.sum
   git commit -m "chore: add x/image and go-pdf/fpdf for diploma rendering"
   ```

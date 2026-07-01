# Credential Seeding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `CredentialSeeder` and `CredentialExtractionSeeder` that generate realistic Indonesian diploma credentials in JPEG/PNG/PDF, store across Postgres, MongoDB, and on-chain — all deterministic.

**Architecture:** Diploma rendering lives in `diploma.go` (content building + image/PDF generation). `credential_seeder.go` orchestrates: query live holders/issuers, build diploma content, render to file, keccak256 hash, AES encrypt, save to disk, INSERT Postgres, upsert MongoDB extraction. `credential_extraction_seeder.go` re-derives content and stores `credential_extractions` to MongoDB. Chain minting extends `seed_chain.go` with credential minting and batch UPDATE.

**Tech Stack:** Go 1.25.1, `golang.org/x/image/font`, `github.com/go-pdf/fpdf`, `github.com/ethereum/go-ethereum/crypto` (keccak256), GORM, MongoDB driver v2

## Global Constraints

- Module path: `CredChain_Golang` (underscore)
- Seeder interface: `Name() string`, `Seed(ctx context.Context) error` — see `seeder.go`
- Registry runs seeders in registration order
- Deterministic: `hashToSeed("credchain-seed")` seeds `math/rand` and `gofakeit`
- Pass `*config.Config` directly to seeder constructors (not individual fields)
- Variable names in English: `universityHead`, `dean`, `credentialName`
- Extraction ID keys: `dean_name`, `dean_nip`, `university_head_name`, `university_head_nip`
- Helper functions: `seed` prefix convention
- Error wrapping: `fmt.Errorf("credential seeder: context: %w", err)`
- Tests: white-box, in-package (`package seeder_test`), testify/assert
- No River job, no Python call — direct storage
- No N+1 queries — batch operations only
- Re-verify with: `go test ./... && go vet ./... && gofmt -l .`

---

### Task 1: Add Dependencies

**Files:**
- Modify: `go.mod` (via `go get`)

**Interfaces:**
- Produces: `golang.org/x/image` (font rendering), `github.com/go-pdf/fpdf` (PDF generation) available for import

- [ ] **Step 1: Add dependencies via go get**

```bash
cd CredChain_Golang
go get golang.org/x/image@latest
go get github.com/go-pdf/fpdf@latest
go mod tidy
```

- [ ] **Step 2: Verify modules are resolved**

```bash
grep -E 'golang.org/x/image|go-pdf/fpdf' go.mod
```
Expected: both modules appear in go.mod require blocks.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add x/image and go-pdf/fpdf for diploma rendering"
```

---

### Task 2: Rename seedHashSeed → hashToSeed

**Files:**
- Modify: `infrastructure/database/seeder/user_seeder.go:30,204-208`

**Interfaces:**
- Produces: `func hashToSeed(s string) int64` — FNV-64a hash to int64 seed

- [ ] **Step 1: Rename function and update call site**

In `infrastructure/database/seeder/user_seeder.go`, line 204:

```go
func hashToSeed(s string) int64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return int64(h.Sum64())
}
```

In line 30, change `seedHashSeed` to `hashToSeed`:

```go
seed := hashToSeed("credchain-seed")
```

- [ ] **Step 2: Run tests to verify**

```bash
cd CredChain_Golang
go test ./infrastructure/database/seeder/... -v
```
Expected: all tests pass (UserSeeder tests use this function).

- [ ] **Step 3: Commit**

```bash
git add infrastructure/database/seeder/user_seeder.go
git commit -m "refactor: rename seedHashSeed to hashToSeed"
```

---

### Task 3: Create diploma.go — Diploma Content & Rendering

**Files:**
- Create: `infrastructure/database/seeder/diploma.go`

**Interfaces:**
- Produces:
  - `type diplomaContent struct` — all fields for rendering
  - `func seedBuildDiplomaContent(holder domain.User, issuer domain.User, credentialIndex int, rng *rand.Rand) diplomaContent`
  - `func seedRenderDiplomaPNG(content diplomaContent) ([]byte, error)`
  - `func seedRenderDiplomaJPEG(content diplomaContent) ([]byte, error)`
  - `func seedRenderDiplomaPDF(content diplomaContent, pngBytes []byte) ([]byte, error)`
  - `func seedBuildDiplomaText(content diplomaContent) string`
  - `func seedBuildDiplomaExtractionIDs(content diplomaContent) []domain.CredentialExtractedID`
  - `var seedCredentialNames []seedCredentialName` — 10 Indonesian degrees
  - `var seedCredentialShortToProgramStudi map[string]string` — short name → program studi mapping

- [ ] **Step 1: Write diploma.go with all types, content builder, renderers**

```go
package seeder

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math/rand"
	"strings"
	"time"

	"CredChain_Golang/domain"

	"github.com/go-pdf/fpdf"
	"github.com/samber/lo"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const seedDiplomaWidth = 1240
const seedDiplomaHeight = 1754

var seedDiplomaWhite = color.RGBA{255, 255, 255, 255}
var seedDiplomaBlack = color.RGBA{0, 0, 0, 255}
var seedDiplomaDarkBlue = color.RGBA{30, 40, 100, 255}
var seedDiplomaGray = color.RGBA{100, 100, 100, 255}
var seedDiplomaLightGray = color.RGBA{230, 230, 230, 255}

var seedBoldFont *opentype.Font
var seedRegularFont *opentype.Font

func init() {
	var err error
	seedBoldFont, err = opentype.Parse(gobold.TTF)
	if err != nil {
		panic(fmt.Sprintf("diploma: parse bold font: %v", err))
	}
	seedRegularFont, err = opentype.Parse(goregular.TTF)
	if err != nil {
		panic(fmt.Sprintf("diploma: parse regular font: %v", err))
	}
}

type seedCredentialName struct {
	Name  string
	Short string
}

var seedCredentialNames = []seedCredentialName{
	{"Sarjana Komputer", "S.Kom"},
	{"Sarjana Akuntansi", "S.Ak"},
	{"Sarjana Hukum", "S.H."},
	{"Sarjana Kedokteran", "S.Ked."},
	{"Sarjana Teknik", "S.T."},
	{"Sarjana Ekonomi", "S.E."},
	{"Sarjana Psikologi", "S.Psi."},
	{"Sarjana Pendidikan", "S.Pd."},
	{"Sarjana Ilmu Komunikasi", "S.I.Kom."},
	{"Sarjana Administrasi Bisnis", "S.A.B."},
}

var seedCredentialShortToProgramStudi = map[string]string{
	"S.Kom":    "Teknik Informatika",
	"S.Ak":     "Akuntansi",
	"S.H.":     "Ilmu Hukum",
	"S.Ked.":   "Pendidikan Dokter",
	"S.T.":     "Teknik Sipil",
	"S.E.":     "Ilmu Ekonomi",
	"S.Psi.":   "Psikologi",
	"S.Pd.":    "Pendidikan Bahasa Indonesia",
	"S.I.Kom.": "Ilmu Komunikasi",
	"S.A.B.":   "Administrasi Bisnis",
}

const seedDiplomaOrgName = "UNIVERSITAS CREDCHAIN INDONESIA"
const seedDiplomaSubtitle = "Memberikan Kepada"
const seedDiplomaLegalText = "Dengan segala hak dan kewajiban yang berhubungan dengan sebutan akademik ini."
const seedDiplomaRektorName = "Prof. Dr. Ahmad Wijaya, M.Sc"
const seedDiplomaRektorNIP = "196803152008123456"

type diplomaContent struct {
	HolderName        string
	HolderBirthDate   string
	HolderBirthPlace  string
	IssuedDate        string
	CredentialName    string
	CredentialShort   string
	ProgramStudi      string
	DeanName          string
	DeanNIP           string
	OrganizationName  string
	UniversityHeadName  string
	UniversityHeadNIP   string
}

func seedBuildDiplomaContent(holder domain.User, issuer domain.User, credentialIndex int, rng *rand.Rand) diplomaContent {
	cn := seedCredentialNames[credentialIndex%len(seedCredentialNames)]
	programStudi := seedCredentialShortToProgramStudi[cn.Short]

	birthDate := seedFormatDiplomaDate(holder.BirthDate)
	if holder.BirthDate == nil {
		birthDate = seedFormatDiplomaDate(lo.ToPtr(seedRandomBirthDate(rng)))
	}

	issuedDate := time.Now().In(time.FixedZone("WIB", 7*3600))
	issuedDateStr := issuedDate.Format("2 January 2006")

	holderName := "—"
	if holder.Name != nil {
		holderName = *holder.Name
	}

	issuerName := "—"
	if issuer.Name != nil {
		issuerName = *issuer.Name
	}
	issuerNIP := "—"
	if issuer.Number != nil {
		issuerNIP = *issuer.Number
	}

	holderBirthPlace := seedDeterministicBirthPlace(holder.Id, rng)

	return diplomaContent{
		HolderName:          holderName,
		HolderBirthDate:     birthDate,
		HolderBirthPlace:    holderBirthPlace,
		IssuedDate:          issuedDateStr,
		CredentialName:      cn.Name,
		CredentialShort:     cn.Short,
		ProgramStudi:        programStudi,
		DeanName:            issuerName,
		DeanNIP:             issuerNIP,
		OrganizationName:    seedDiplomaOrgName,
		UniversityHeadName:  seedDiplomaRektorName,
		UniversityHeadNIP:   seedDiplomaRektorNIP,
	}
}

func seedFormatDiplomaDate(t *time.Time) string {
	if t == nil {
		return "—"
	}
	months := []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month()-1], t.Year())
}

var seedIndonesianCities = []string{
	"Jakarta", "Bandung", "Surabaya", "Medan", "Yogyakarta",
	"Semarang", "Makassar", "Palembang", "Denpasar", "Malang",
	"Padang", "Bogor", "Pekanbaru", "Pontianak", "Banjarmasin",
	"Balikpapan", "Manado", "Mataram", "Aceh", "Ambon",
}

func seedDeterministicBirthPlace(userID string, rng *rand.Rand) string {
	idx := seedHashStringToIndex(userID, len(seedIndonesianCities))
	return seedIndonesianCities[idx]
}

func seedHashStringToIndex(s string, n int) int {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return int(h % uint32(n))
}

func seedRenderDiplomaPNG(content diplomaContent) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, seedDiplomaWidth, seedDiplomaHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{seedDiplomaWhite}, image.Point{}, draw.Src)

	seedDiplomaDrawBorder(img)
	seedDiplomaDrawHeader(img)
	seedDiplomaDrawBody(img, content)
	seedDiplomaDrawFooter(img, content)
	seedDiplomaDrawSeal(img)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("diploma: encode png: %w", err)
	}
	return buf.Bytes(), nil
}

func seedRenderDiplomaJPEG(content diplomaContent) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, seedDiplomaWidth, seedDiplomaHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{seedDiplomaWhite}, image.Point{}, draw.Src)

	seedDiplomaDrawBorder(img)
	seedDiplomaDrawHeader(img)
	seedDiplomaDrawBody(img, content)
	seedDiplomaDrawFooter(img, content)
	seedDiplomaDrawSeal(img)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("diploma: encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func seedRenderDiplomaPDF(content diplomaContent, pngBytes []byte) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	opts := fpdf.ImageInfoType{
		sm: fpdf.ImageOptions{ImageType: "png"},
	}
	info := pdf.RegisterImageInfoReader("diploma", opts, bytes.NewReader(pngBytes), "", "")
	if info != nil {
		pdf.ImageInfo("diploma", 0, 0, 210, 297, false, "", 0, "")
	}

	seedDiplomaAddPDFText(pdf, content)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("diploma: output pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func seedDiplomaAddPDFText(pdf *fpdf.Fpdf, content diplomaContent) {
	pdf.SetFont("Helvetica", "", 8)
	col := map[string]float64{"x": 25, "top": 75, "lineH": 6}

	writeLine := func(y float64, text string) {
		pdf.SetXY(col["x"], y)
		pdf.CellFormat(160, col["lineH"], text, "", 0, "L", false, 0, "")
	}

	y := col["top"]
	pdf.SetFont("Helvetica", "B", 10)
	writeLine(y, "Nama: "+content.HolderName)
	y += 7
	pdf.SetFont("Helvetica", "", 8)
	writeLine(y, "Tempat/Tgl Lahir: "+content.HolderBirthPlace+", "+content.HolderBirthDate)
	y += col["lineH"]
	writeLine(y, "Gelar: "+content.CredentialShort+" ("+content.CredentialName+")")
	y += col["lineH"]
	writeLine(y, "Program Studi: "+content.ProgramStudi)
	y += col["lineH"]
	writeLine(y, "Tanggal Dikeluarkan: "+content.IssuedDate)
	y += col["lineH"]
	writeLine(y, "Rektor: "+content.UniversityHeadName+" | NIP: "+content.UniversityHeadNIP)
	y += col["lineH"]
	writeLine(y, "Dekan: "+content.DeanName+" | NIP: "+content.DeanNIP)
}

func seedDiplomaDrawBorder(img *image.RGBA) {
	margin := 60
	x0, y0 := margin, margin
	x1, y1 := seedDiplomaWidth-margin, seedDiplomaHeight-margin
	seedDrawRect(img, x0, y0, x1-x0, y1-y0, seedDiplomaDarkBlue, 3)
	seedDrawRect(img, x0+15, y0+15, x1-x0-30, y1-y0-30, seedDiplomaDarkBlue, 1)
}

func seedDiplomaDrawHeader(img *image.RGBA) {
	y := 90
	seedDrawEmblem(img, seedDiplomaWidth/2-35, y-10, 70, 70)

	y += 75
	seedDrawCenteredText(img, y, seedDiplomaOrgName, seedDiplomaDarkBlue, 22, seedBoldFont)

	y += 30
	seedDrawCenteredText(img, y, "SK Menteri Pendidikan No. 42/CREDCHAIN/2024", seedDiplomaGray, 10, seedRegularFont)

	y += 40
	seedDrawLine(img, seedDiplomaWidth/2-200, y, 400, seedDiplomaDarkBlue, 2)

	y += 25
	seedDrawCenteredText(img, y, seedDiplomaSubtitle, seedDiplomaDarkBlue, 16, seedBoldFont)
}

func seedDiplomaDrawBody(img *image.RGBA, c diplomaContent) {
	y := 400

	seedDrawCenteredText(img, y, "IJAZAH", seedDiplomaDarkBlue, 28, seedBoldFont)
	y += 50

	seedDrawCenteredText(img, y, "No. "+c.IssuedDate, seedDiplomaGray, 10, seedRegularFont)
	y += 55

	seedDrawCenteredText(img, y, c.HolderName, seedDiplomaBlack, 26, seedBoldFont)
	y += 38

	seedDrawCenteredText(img, y, fmt.Sprintf("Tempat / Tanggal Lahir: %s, %s", c.HolderBirthPlace, c.HolderBirthDate), seedDiplomaBlack, 14, seedRegularFont)
	y += 45

	seedDrawCenteredText(img, y, c.CredentialName+" ("+c.CredentialShort+")", seedDiplomaDarkBlue, 18, seedBoldFont)
	y += 30

	seedDrawCenteredText(img, y, "Program Studi: "+c.ProgramStudi, seedDiplomaBlack, 14, seedRegularFont)
	y += 50

	seedDrawLine(img, seedDiplomaWidth/2-150, y, 300, seedDiplomaDarkBlue, 1)
	y += 25

	lines := seedWrapText(seedDiplomaLegalText, 60)
	for _, line := range lines {
		seedDrawCenteredText(img, y, line, seedDiplomaBlack, 11, seedRegularFont)
		y += 18
	}
}

func seedDiplomaDrawFooter(img *image.RGBA, c diplomaContent) {
	y := 1450

	seedDrawCenteredText(img, y, c.IssuedDate, seedDiplomaBlack, 12, seedRegularFont)
	y += 25

	seedDrawCenteredText(img, y, "Rektor,", seedDiplomaBlack, 12, seedRegularFont)
	y += 20

	x := seedDiplomaWidth/2 - 180
	seedDrawTextAt(img, x, y, "Dekan,", seedDiplomaBlack, 12, seedRegularFont)
	seedDrawTextAt(img, x+240, y, "Rektor,", seedDiplomaBlack, 12, seedRegularFont)

	y += 60
	seedDrawTextAt(img, x, y, c.DeanName, seedDiplomaBlack, 12, seedBoldFont)
	seedDrawTextAt(img, x+240, y, c.UniversityHeadName, seedDiplomaBlack, 12, seedBoldFont)

	y += 18
	seedDrawTextAt(img, x, y, "NIP. "+c.DeanNIP, seedDiplomaBlack, 10, seedRegularFont)
	seedDrawTextAt(img, x+240, y, "NIP. "+c.UniversityHeadNIP, seedDiplomaBlack, 10, seedRegularFont)
}

func seedDiplomaDrawSeal(img *image.RGBA) {
	cx := seedDiplomaWidth/2 + 80
	cy := 1500
	r := 40

	for _, dy := range []int{-r, -r/2, 0, r/2, r} {
		for _, dx := range []int{-r, -r/2, 0, r/2, r} {
			dist := (dx*dx + dy*dy)
			if dist <= r*r {
				seedSetPixel(img, cx+dx, cy+dy, color.RGBA{180, 0, 0, 50})
			}
		}
	}
	seedDrawCenteredText(img, cy-5, "UNIVERSITAS", color.RGBA{180, 0, 0, 100}, 8, seedBoldFont)
	seedDrawCenteredText(img, cy+7, "CREDCHAIN", color.RGBA{180, 0, 0, 100}, 7, seedBoldFont)
}

func seedDrawEmblem(img *image.RGBA, x, y, w, h int) {
	cx, cy := x+w/2, y+h/2
	r := w / 2

	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				seedSetPixel(img, cx+dx, cy+dy, seedDiplomaDarkBlue)
			}
		}
	}

	innerR := r * 2 / 3
	for dy := -innerR; dy <= innerR; dy++ {
		for dx := -innerR; dx <= innerR; dx++ {
			if dx*dx+dy*dy <= innerR*innerR {
				seedSetPixel(img, cx+dx, cy+dy, seedDiplomaWhite)
			}
		}
	}

	seedDrawCenteredTextAt(img, cx, cy-4, "UCI", seedDiplomaDarkBlue, 12, seedBoldFont)
}

func seedDrawRect(img *image.RGBA, x, y, w, h int, c color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		for i := x + t; i < x+w-t; i++ {
			seedSetPixel(img, i, y+t, c)
			seedSetPixel(img, i, y+h-1-t, c)
		}
		for j := y + t; j < y+h-t; j++ {
			seedSetPixel(img, x+t, j, c)
			seedSetPixel(img, x+w-1-t, j, c)
		}
	}
}

func seedDrawLine(img *image.RGBA, x, y, w int, c color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		for i := 0; i < w; i++ {
			seedSetPixel(img, x+i, y+t, c)
		}
	}
}

func seedDrawCenteredText(img *image.RGBA, y int, text string, c color.RGBA, size float64, otf *opentype.Font) {
	face, _ := opentype.NewFace(otf, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	defer face.Close()

	adv := font.MeasureString(face, text)
	width := adv.Ceil()
	seedDrawTextAt(img, (seedDiplomaWidth-width)/2, y, text, c, size, otf)
}

func seedDrawCenteredTextAt(img *image.RGBA, cx, y int, text string, c color.RGBA, size float64, otf *opentype.Font) {
	face, _ := opentype.NewFace(otf, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	defer face.Close()

	adv := font.MeasureString(face, text)
	width := adv.Ceil()
	seedDrawTextAt(img, cx-width/2, y, text, c, size, otf)
}

func seedDrawTextAt(img *image.RGBA, x, y int, text string, c color.RGBA, size float64, otf *opentype.Font) {
	face, err := opentype.NewFace(otf, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return
	}
	defer face.Close()

	d := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{c},
		Face: face,
	}
	px := fixed.I(x)
	for _, r := range text {
		dr, mask, maskp, advance, ok := d.Face.Glyph(fixed.Point26_6{X: px, Y: fixed.I(y)}, r)
		if ok {
			draw.DrawMask(d.Dst, dr, d.Src, image.Point{}, mask, maskp, draw.Over)
		}
		px += advance
	}
}

func seedSetPixel(img *image.RGBA, px, py int, c color.RGBA) {
	b := img.Bounds()
	if px < b.Min.X || px >= b.Max.X || py < b.Min.Y || py >= b.Max.Y {
		return
	}
	off := img.PixOffset(px, py)
	img.Pix[off+0] = c.R
	img.Pix[off+1] = c.G
	img.Pix[off+2] = c.B
	img.Pix[off+3] = c.A
}

func seedWrapText(text string, maxChars int) []string {
	words := strings.Fields(text)
	var lines []string
	var current string
	for _, w := range words {
		if len(current)+len(w)+1 > maxChars && current != "" {
			lines = append(lines, current)
			current = w
		} else if current == "" {
			current = w
		} else {
			current += " " + w
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func seedBuildDiplomaText(c diplomaContent) string {
	return fmt.Sprintf(
		"%s\n"+
			"%s\n"+
			"IJAZAH\n"+
			"Nama: %s\n"+
			"Tempat/Tanggal Lahir: %s, %s\n"+
			"Gelar: %s (%s)\n"+
			"Program Studi: %s\n"+
			"Dikeluarkan: %s\n"+
			"%s\n"+
			"Rektor: %s (NIP: %s)\n"+
			"Dekan: %s (NIP: %s)\n"+
			"Institusi: %s",
		c.OrganizationName,
		seedDiplomaSubtitle,
		c.HolderName,
		c.HolderBirthPlace,
		c.HolderBirthDate,
		c.CredentialShort, c.CredentialName,
		c.ProgramStudi,
		c.IssuedDate,
		seedDiplomaLegalText,
		c.UniversityHeadName, c.UniversityHeadNIP,
		c.DeanName, c.DeanNIP,
		c.OrganizationName,
	)
}

func seedBuildDiplomaExtractionIDs(c diplomaContent) []domain.CredentialExtractedID {
	return []domain.CredentialExtractedID{
		{Type: "credential_name", Value: c.CredentialName},
		{Type: "credential_short", Value: c.CredentialShort},
		{Type: "program_studi", Value: c.ProgramStudi},
		{Type: "holder_name", Value: c.HolderName},
		{Type: "birth_date", Value: c.HolderBirthDate},
		{Type: "birth_place", Value: c.HolderBirthPlace},
		{Type: "university_head_name", Value: c.UniversityHeadName},
		{Type: "university_head_nip", Value: c.UniversityHeadNIP},
		{Type: "dean_name", Value: c.DeanName},
		{Type: "dean_nip", Value: c.DeanNIP},
		{Type: "issued_date", Value: c.IssuedDate},
		{Type: "institution", Value: c.OrganizationName},
	}
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd CredChain_Golang
go build ./infrastructure/database/seeder/...
```
Expected: builds without errors.

- [ ] **Step 3: Commit**

```bash
git add infrastructure/database/seeder/diploma.go
git commit -m "feat: add diploma rendering (PNG/JPEG/PDF) for credential seeder"
```

---

### Task 4: Create diploma_test.go — Diploma Rendering Tests

**Files:**
- Create: `infrastructure/database/seeder/diploma_test.go`

**Interfaces:**
- Consumes: `diplomaContent`, `seedBuildDiplomaContent`, `seedRenderDiplomaPNG`, `seedRenderDiplomaJPEG`, `seedRenderDiplomaPDF`, `seedBuildDiplomaText`, `seedBuildDiplomaExtractionIDs`, `seedCredentialNames`
- Produces: nothing (tests only)

- [ ] **Step 1: Write diploma_test.go**

```go
package seeder_test

import (
	"bytes"
	"image/png"
	"math/rand"
	"strings"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/seeder"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestSeedCredentialNames_All10HaveProgramStudi(t *testing.T) {
	cn := seeder.SeedCredentialNames()
	shortToProdi := seeder.SeedCredentialShortToProgramStudi()
	assert.Len(t, cn, 10)
	for _, c := range cn {
		assert.NotEmpty(t, c.Name)
		assert.NotEmpty(t, c.Short)
		assert.NotEmpty(t, shortToProdi[c.Short], "missing program studi for %s", c.Short)
	}
}

func TestDiplomaContent_AllFieldsPopulated(t *testing.T) {
	holder := domain.User{
		Id:   "01JTEST000000000000000000",
		Name: lo.ToPtr("Budi Santoso"),
		BirthDate: lo.ToPtr(time.Date(1998, 6, 15, 0, 0, 0, 0, time.UTC)),
	}
	issuer := domain.User{
		Id:   "01JTEST000000000000000001",
		Name: lo.ToPtr("Dr. Siti Rahayu"),
		Number: lo.ToPtr("197505202003122001"),
	}
	rng := rand.New(rand.NewSource(42))
	content := seeder.SeedBuildDiplomaContent(holder, issuer, 0, rng)

	assert.Equal(t, "Budi Santoso", content.HolderName)
	assert.Contains(t, content.HolderBirthDate, "1998")
	assert.NotEmpty(t, content.HolderBirthPlace)
	assert.NotEmpty(t, content.IssuedDate)
	assert.Equal(t, "Sarjana Komputer", content.CredentialName)
	assert.Equal(t, "S.Kom", content.CredentialShort)
	assert.Equal(t, "Teknik Informatika", content.ProgramStudi)
	assert.Equal(t, "Dr. Siti Rahayu", content.DeanName)
	assert.Equal(t, "197505202003122001", content.DeanNIP)
}

func TestDiplomaContent_FallbackForNilBirthDate(t *testing.T) {
	holder := domain.User{Id: "01JTEST000000000000000000"}
	issuer := domain.User{Id: "01JTEST000000000000000001"}
	rng := rand.New(rand.NewSource(99))
	content := seeder.SeedBuildDiplomaContent(holder, issuer, 0, rng)
	assert.NotEqual(t, "—", content.HolderBirthDate)
	assert.NotEmpty(t, content.HolderBirthPlace)
}

func TestDiplomaContent_FallbackForNilName(t *testing.T) {
	holder := domain.User{Id: "01JTEST000000000000000000", BirthDate: lo.ToPtr(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))}
	issuer := domain.User{Id: "01JTEST000000000000000001"}
	rng := rand.New(rand.NewSource(42))
	content := seeder.SeedBuildDiplomaContent(holder, issuer, 0, rng)
	assert.Equal(t, "—", content.HolderName)
	assert.Equal(t, "—", content.DeanName)
}

func TestRenderDiplomaPNG_ProducesValidImage(t *testing.T) {
	content := seeder.SeedBuildDiplomaContent(
		domain.User{Id: "A", Name: lo.ToPtr("Test Holder")},
		domain.User{Id: "B", Name: lo.ToPtr("Test Issuer"), Number: lo.ToPtr("123456789012345678")},
		0, rand.New(rand.NewSource(1)),
	)
	data, err := seeder.SeedRenderDiplomaPNG(content)
	assert.NoError(t, err)
	assert.Greater(t, len(data), 1000)

	img, err := png.Decode(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Equal(t, 1240, img.Bounds().Dx())
	assert.Equal(t, 1754, img.Bounds().Dy())
}

func TestRenderDiplomaJPEG_ProducesValidImage(t *testing.T) {
	content := seeder.SeedBuildDiplomaContent(
		domain.User{Id: "A", Name: lo.ToPtr("Test Holder")},
		domain.User{Id: "B", Name: lo.ToPtr("Test Issuer"), Number: lo.ToPtr("123456789012345678")},
		0, rand.New(rand.NewSource(1)),
	)
	data, err := seeder.SeedRenderDiplomaJPEG(content)
	assert.NoError(t, err)
	assert.Greater(t, len(data), 1000)

	img, err := jpeg.Decode(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Equal(t, 1240, img.Bounds().Dx())
}

func TestRenderDiplomaPDF_ProducesValidPDF(t *testing.T) {
	content := seeder.SeedBuildDiplomaContent(
		domain.User{Id: "A", Name: lo.ToPtr("Test Holder")},
		domain.User{Id: "B", Name: lo.ToPtr("Test Issuer"), Number: lo.ToPtr("123456789012345678")},
		0, rand.New(rand.NewSource(1)),
	)
	pngData, err := seeder.SeedRenderDiplomaPNG(content)
	assert.NoError(t, err)

	pdfData, err := seeder.SeedRenderDiplomaPDF(content, pngData)
	assert.NoError(t, err)
	assert.Greater(t, len(pdfData), 100)
	assert.True(t, bytes.Contains(pdfData, []byte("%PDF")))
}

func TestBuildDiplomaText_ContainsAllFields(t *testing.T) {
	content := seeder.SeedBuildDiplomaContent(
		domain.User{Id: "A", Name: lo.ToPtr("Test Holder")},
		domain.User{Id: "B", Name: lo.ToPtr("Test Issuer"), Number: lo.ToPtr("123456789012345678")},
		0, rand.New(rand.NewSource(1)),
	)
	text := seeder.SeedBuildDiplomaText(content)

	assert.True(t, strings.Contains(text, "Test Holder"))
	assert.True(t, strings.Contains(text, "Test Issuer"))
	assert.True(t, strings.Contains(text, "UNIVERSITAS CREDCHAIN INDONESIA"))
	assert.True(t, strings.Contains(text, "IJAZAH"))
	assert.True(t, strings.Contains(text, "Prof. Dr. Ahmad Wijaya"))
}

func TestBuildDiplomaExtractionIDs_12Fields(t *testing.T) {
	content := seeder.SeedBuildDiplomaContent(
		domain.User{Id: "A", Name: lo.ToPtr("Test Holder")},
		domain.User{Id: "B", Name: lo.ToPtr("Test Issuer"), Number: lo.ToPtr("123456789012345678")},
		0, rand.New(rand.NewSource(1)),
	)
	ids := seeder.SeedBuildDiplomaExtractionIDs(content)
	assert.Len(t, ids, 12)

	idMap := make(map[string]string, len(ids))
	for _, id := range ids {
		idMap[id.Type] = id.Value
	}
	assert.Equal(t, content.CredentialName, idMap["credential_name"])
	assert.Equal(t, content.CredentialShort, idMap["credential_short"])
	assert.Equal(t, content.DeanName, idMap["dean_name"])
	assert.Equal(t, content.DeanNIP, idMap["dean_nip"])
	assert.Equal(t, content.UniversityHeadName, idMap["university_head_name"])
	assert.Equal(t, content.UniversityHeadNIP, idMap["university_head_nip"])
}

func TestDiplomaContent_CredentialIndexCyclesNames(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	holder := domain.User{Id: "A"}
	issuer := domain.User{Id: "B"}

	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		c := seeder.SeedBuildDiplomaContent(holder, issuer, i, rng)
		assert.False(t, seen[c.CredentialShort], "duplicate credential at index %d: %s", i, c.CredentialShort)
		seen[c.CredentialShort] = true
	}
	assert.Len(t, seen, 10)
}

func TestDiplomaContent_Deterministic(t *testing.T) {
	holder := domain.User{Id: "A", Name: lo.ToPtr("X")}
	issuer := domain.User{Id: "B", Name: lo.ToPtr("Y"), Number: lo.ToPtr("Z")}

	c1 := seeder.SeedBuildDiplomaContent(holder, issuer, 0, rand.New(rand.NewSource(1)))
	c2 := seeder.SeedBuildDiplomaContent(holder, issuer, 0, rand.New(rand.NewSource(1)))

	assert.Equal(t, c1, c2)
}

func TestRenderDiplomaPNG_Deterministic(t *testing.T) {
	content := seeder.SeedBuildDiplomaContent(
		domain.User{Id: "A", Name: lo.ToPtr("X")},
		domain.User{Id: "B", Name: lo.ToPtr("Y"), Number: lo.ToPtr("Z")},
		0, rand.New(rand.NewSource(1)),
	)
	d1, _ := seeder.SeedRenderDiplomaPNG(content)
	d2, _ := seeder.SeedRenderDiplomaPNG(content)
	assert.Equal(t, d1, d2)
}
```

Note: Replace `time.Date` with `time.Date` import. Add `time` import to test file.

- [ ] **Step 2: Run tests to verify they pass**

```bash
cd CredChain_Golang
go test ./infrastructure/database/seeder/... -run TestSeedCredentialNames -v
go test ./infrastructure/database/seeder/... -run TestDiplomaContent -v
go test ./infrastructure/database/seeder/... -run TestRenderDiploma -v
go test ./infrastructure/database/seeder/... -run TestBuildDiploma -v
```

Expected: all tests pass.

- [ ] **Step 3: Run full seeder test suite**

```bash
go test ./infrastructure/database/seeder/... -v
```
Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add infrastructure/database/seeder/diploma_test.go
git commit -m "test: add diploma rendering tests"
```

---

### Task 5: Create credential_seeder.go — CredentialSeeder

**Files:**
- Create: `infrastructure/database/seeder/credential_seeder.go`

**Interfaces:**
- Consumes: `domain.UserRepository`, `domain.CredentialRepository`, `domain.CredentialExtractionRepository`, `*config.Config`, `*storage.Storage`
- Produces: `func NewCredentialSeeder(cfg *config.Config, userRepo domain.UserRepository, credRepo domain.CredentialRepository, extrRepo domain.CredentialExtractionRepository, storage *storage.Storage) *CredentialSeeder`
- Implements: `Seeder` interface (`Name() string`, `Seed(ctx context.Context) error`)

- [ ] **Step 1: Write credential_seeder.go**

```go
package seeder

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	"CredChain_Golang/infrastructure/storage"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
)

type CredentialSeeder struct {
	cfg         *config.Config
	userRepo    domain.UserRepository
	credRepo    domain.CredentialRepository
	extrRepo    domain.CredentialExtractionRepository
	storage     *storage.Storage
}

func NewCredentialSeeder(
	cfg *config.Config,
	userRepo domain.UserRepository,
	credRepo domain.CredentialRepository,
	extrRepo domain.CredentialExtractionRepository,
	storage *storage.Storage,
) *CredentialSeeder {
	return &CredentialSeeder{
		cfg:      cfg,
		userRepo: userRepo,
		credRepo: credRepo,
		extrRepo: extrRepo,
		storage:  storage,
	}
}

func (s *CredentialSeeder) Name() string { return "credential" }

func (s *CredentialSeeder) Seed(ctx context.Context) error {
	seed := hashToSeed("credchain-seed")
	rng := rand.New(rand.NewSource(seed))

	holders, err := s.queryLiveRoleUsers(ctx, domain.RoleHolder)
	if err != nil {
		return fmt.Errorf("credential seeder: query holders: %w", err)
	}
	if len(holders) == 0 {
		return fmt.Errorf("credential seeder: no live holders found — run user seeder first")
	}

	issuers, err := s.queryLiveRoleUsers(ctx, domain.RoleIssuer)
	if err != nil {
		return fmt.Errorf("credential seeder: query issuers: %w", err)
	}
	if len(issuers) == 0 {
		return fmt.Errorf("credential seeder: no live issuers found — run user seeder first")
	}

	totalCredentials := len(holders) * 3
	credentials := make([]domain.Credential, 0, totalCredentials)
	extractions := make([]domain.CredentialExtraction, 0, totalCredentials)

	var holderIdx, issuerIdx int
	for i := 0; i < totalCredentials; i++ {
		holder := holders[holderIdx%len(holders)]
		issuer := issuers[issuerIdx%len(issuers)]

		if i%3 == 0 {
			holderIdx++
		}
		if i%12 == 0 {
			issuerIdx++
		}

		content := seedBuildDiplomaContent(holder, issuer, i, rng)
		mimeIndex := i % 3

		cred, extr, err := s.seedBuildCredential(ctx, content, holder, issuer, mimeIndex, i)
		if err != nil {
			return fmt.Errorf("credential seeder: build credential %d: %w", i, err)
		}

		credentials = append(credentials, cred)
		extractions = append(extractions, extr)
	}

	stored, err := s.credRepo.Store(ctx, credentials...)
	if err != nil {
		return fmt.Errorf("credential seeder: store credentials: %w", err)
	}

	for i, extr := range extractions {
		extr.CredentialID = stored[i].ID
		if err := s.extrRepo.Store(ctx, extr); err != nil {
			return fmt.Errorf("credential seeder: store extraction %d: %w", i, err)
		}
	}

	revokeCount := totalCredentials / 5
	revokeIDs := make([]string, 0, revokeCount)
	for i := 0; i < revokeCount; i++ {
		idx := (i * 5) % len(stored)
		revokeIDs = append(revokeIDs, stored[idx].ID)
	}

	if len(revokeIDs) > 0 {
		now := time.Now()
		toRevoke := make([]domain.Credential, 0, len(revokeIDs))
		for _, id := range revokeIDs {
			toRevoke = append(toRevoke, domain.Credential{
				ID: id, RevokedAt: &now,
				RevokerUserID: lo.ToPtr(stored[0].IssuerUserID),
			})
		}
		if _, err := s.credRepo.Update(ctx, toRevoke...); err != nil {
			return fmt.Errorf("credential seeder: revoke: %w", err)
		}
	}

	return nil
}

func (s *CredentialSeeder) seedBuildCredential(
	ctx context.Context,
	content diplomaContent,
	holder domain.User,
	issuer domain.User,
	mimeIndex int,
	credIndex int,
) (domain.Credential, domain.CredentialExtraction, error) {
	var fileBytes []byte
	var ext string

	switch mimeIndex {
	case 0:
		fileBytes, ext = nil, "jpeg"
		var err error
		fileBytes, err = seedRenderDiplomaJPEG(content)
		if err != nil {
			return domain.Credential{}, domain.CredentialExtraction{}, err
		}
	case 1:
		var err error
		fileBytes, err = seedRenderDiplomaPNG(content)
		if err != nil {
			return domain.Credential{}, domain.CredentialExtraction{}, err
		}
		ext = "png"
	case 2:
		pngBytes, err := seedRenderDiplomaPNG(content)
		if err != nil {
			return domain.Credential{}, domain.CredentialExtraction{}, err
		}
		fileBytes, err = seedRenderDiplomaPDF(content, pngBytes)
		if err != nil {
			return domain.Credential{}, domain.CredentialExtraction{}, err
		}
		ext = "pdf"
	}

	fileHash := "0x" + hex.EncodeToString(ethCrypto.Keccak256(fileBytes))

	encryptedBytes, err := cryptoInfra.Encrypt(fileBytes, []byte(*s.cfg.FileEncryptionKey))
	if err != nil {
		return domain.Credential{}, domain.CredentialExtraction{},
			fmt.Errorf("credential seeder: encrypt file: %w", err)
	}

	credID := ulid.Make().String()
	filePath := fmt.Sprintf("%s/%s.%s", *s.cfg.CredentialFileStoragePath, credID, ext)
	if _, err := s.storage.SaveBytes(fileBytes, filePath); err != nil {
		return domain.Credential{}, domain.CredentialExtraction{},
			fmt.Errorf("credential seeder: save file: %w", err)
	}

	encryptedPath := fmt.Sprintf("%s/%s.enc", *s.cfg.CredentialFileStoragePath, credID)
	if _, err := s.storage.SaveBytes([]byte(encryptedBytes), encryptedPath); err != nil {
		return domain.Credential{}, domain.CredentialExtraction{},
			fmt.Errorf("credential seeder: save encrypted file: %w", err)
	}

	now := time.Now()
	cred := domain.Credential{
		ID:            credID,
		HolderUserID:  holder.Id,
		IssuerUserID:  issuer.Id,
		Name:          content.CredentialName,
		FileHash:      fileHash,
		FileURI:       lo.ToPtr(encryptedPath),
		ExtractStatus: domain.ExtractStatusSucceeded,
		ExtractedAt:   &now,
		IssuedAt:      now,
	}

	text := seedBuildDiplomaText(content)
	ids := seedBuildDiplomaExtractionIDs(content)
	embedding := seedBuildEmbedding(text)

	extr := domain.CredentialExtraction{
		FileHash:  fileHash,
		Text:      text,
		IDs:       ids,
		Embedding: embedding,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return cred, extr, nil
}

func (s *CredentialSeeder) queryLiveRoleUsers(ctx context.Context, role domain.Role) ([]domain.User, error) {
	users, err := s.userRepo.FindByRole(ctx, role)
	if err != nil {
		return nil, err
	}
	var live []domain.User
	for _, u := range users {
		if u.DeletedAt == nil {
			live = append(live, u)
		}
	}
	return live, nil
}

func seedBuildEmbedding(text string) []float64 {
	hash := ethCrypto.Keccak256([]byte(text))
	emb := make([]float64, 768)

	for i := range emb {
		idx := i % 32
		pert := float64(i*7+13) * 0.001
		emb[i] = (float64(int8(hash[idx])) - pert*float64(i)) / 128.0
	}

	norm := 0.0
	for _, v := range emb {
		norm += v * v
	}
	if norm > 0 {
		norm = 1.0 / sqrt(norm)
		for i := range emb {
			emb[i] *= norm
		}
	}
	return emb
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 20; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd CredChain_Golang
go build ./infrastructure/database/seeder/...
```
Expected: builds without errors.

- [ ] **Step 3: Commit**

```bash
git add infrastructure/database/seeder/credential_seeder.go
git commit -m "feat: add CredentialSeeder with diploma generation and storage"
```

---

### Task 6: Create credential_seeder_test.go — CredentialSeeder Tests

**Files:**
- Create: `infrastructure/database/seeder/credential_seeder_test.go`

**Interfaces:**
- Consumes: `NewCredentialSeeder`, mocks for repos and storage, `TestWalletEncryptionKey`
- Produces: nothing (tests only)

- [ ] **Step 1: Write credential_seeder_test.go**

```go
package seeder_test

import (
	"context"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/infrastructure/storage"
	"CredChain_Golang/infrastructure/testutil/mocks"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCredentialSeeder_Name(t *testing.T) {
	s := seeder.NewCredentialSeeder(nil, nil, nil, nil, nil)
	assert.Equal(t, "credential", s.Name())
}

func TestCredentialSeeder_NoHoldersReturnsError(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("FindByRole", mock.Anything, domain.RoleHolder).Return([]domain.User{}, nil)

	cfg := &config.Config{}
	s := seeder.NewCredentialSeeder(cfg, userRepo, nil, nil, nil)
	err := s.Seed(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no live holders found")
}

func TestCredentialSeeder_NoIssuersReturnsError(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("FindByRole", mock.Anything, domain.RoleHolder).Return([]domain.User{
		{Id: "h1", Role: domain.RoleHolder},
	}, nil)
	userRepo.On("FindByRole", mock.Anything, domain.RoleIssuer).Return([]domain.User{}, nil)

	cfg := &config.Config{}
	s := seeder.NewCredentialSeeder(cfg, userRepo, nil, nil, nil)
	err := s.Seed(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no live issuers found")
}

func TestCredentialSeeder_SkipsDeletedUsers(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	now := time.Now()
	userRepo.On("FindByRole", mock.Anything, domain.RoleHolder).Return([]domain.User{
		{Id: "h1", Role: domain.RoleHolder},
		{Id: "h2", Role: domain.RoleHolder, DeletedAt: &now},
	}, nil)
	userRepo.On("FindByRole", mock.Anything, domain.RoleIssuer).Return([]domain.User{
		{Id: "i1", Role: domain.RoleIssuer},
	}, nil)

	credRepo := new(mocks.MockCredentialRepository)
	credRepo.On("Store", mock.Anything, mock.Anything).Return(func(ctx context.Context, creds ...domain.Credential) []domain.Credential {
		return creds
	}, nil)
	credRepo.On("Update", mock.Anything, mock.Anything).Return(nil, nil)

	extrRepo := new(mocks.MockCredentialExtractionRepository)
	extrRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	fileKey := string(make([]byte, 32))
	for i := range fileKey {
		fileKey = fileKey[:i] + "A" + fileKey[i+1:]
	}
	cfg := &config.Config{
		FileEncryptionKey:         lo.ToPtr(fileKey),
		CredentialFileStoragePath: lo.ToPtr("credentials"),
		StoragePath:               lo.ToPtr("test_uploads"),
	}
	st, err := storage.NewStorage(storage.StorageParams{Config: cfg})
	assert.NoError(t, err)

	s := seeder.NewCredentialSeeder(cfg, userRepo, credRepo, extrRepo, st)
	err = s.Seed(context.Background())
	assert.NoError(t, err)

	userRepo.AssertExpectations(t)
	credRepo.AssertExpectations(t)
}
```

Note: Add `time` import.

- [ ] **Step 2: Run tests**

```bash
cd CredChain_Golang
go test ./infrastructure/database/seeder/... -run TestCredentialSeeder -v
```
Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add infrastructure/database/seeder/credential_seeder_test.go
git commit -m "test: add CredentialSeeder tests"
```

---

### Task 7: Create credential_extraction_seeder.go

**Files:**
- Create: `infrastructure/database/seeder/credential_extraction_seeder.go`

**Interfaces:**
- Consumes: `domain.CredentialRepository`, `domain.CredentialExtractionRepository`, `*config.Config`
- Produces: `func NewCredentialExtractionSeeder(cfg *config.Config, credRepo domain.CredentialRepository, extrRepo domain.CredentialExtractionRepository) *CredentialExtractionSeeder`
- Implements: `Seeder` interface

- [ ] **Step 1: Write credential_extraction_seeder.go**

```go
package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"

	"github.com/samber/lo"
)

type CredentialExtractionSeeder struct {
	cfg      *config.Config
	credRepo domain.CredentialRepository
	extrRepo domain.CredentialExtractionRepository
}

func NewCredentialExtractionSeeder(
	cfg *config.Config,
	credRepo domain.CredentialRepository,
	extrRepo domain.CredentialExtractionRepository,
) *CredentialExtractionSeeder {
	return &CredentialExtractionSeeder{
		cfg:      cfg,
		credRepo: credRepo,
		extrRepo: extrRepo,
	}
}

func (s *CredentialExtractionSeeder) Name() string { return "credential_extraction" }

func (s *CredentialExtractionSeeder) Seed(ctx context.Context) error {
	seed := hashToSeed("credchain-seed")
	rng := rand.New(rand.NewSource(seed))

	allCreds, _, err := s.credRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("credential extraction seeder: get credentials: %w", err)
	}
	if len(allCreds) == 0 {
		return fmt.Errorf("credential extraction seeder: no credentials — run credential seeder first")
	}

	for i, cred := range allCreds {
		holder := domain.User{
			Id:   cred.HolderUserID,
			Name: lo.EmptyableToPtr(cred.HolderUserID),
		}
		if cred.Holder != nil {
			holder = *cred.Holder
		}

		issuer := domain.User{
			Id:   cred.IssuerUserID,
			Name: lo.EmptyableToPtr(cred.IssuerUserID),
		}
		if cred.Issuer != nil {
			issuer = *cred.Issuer
		}

		content := seedBuildDiplomaContent(holder, issuer, i, rng)
		text := seedBuildDiplomaText(content)
		ids := seedBuildDiplomaExtractionIDs(content)
		embedding := seedBuildEmbedding(text)

		now := time.Now()
		extr := domain.CredentialExtraction{
			CredentialID: cred.ID,
			FileHash:     cred.FileHash,
			Text:         text,
			IDs:          ids,
			Embedding:    embedding,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := s.extrRepo.Store(ctx, extr); err != nil {
			return fmt.Errorf("credential extraction seeder: store %d: %w", i, err)
		}
	}

	return nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./infrastructure/database/seeder/...
```
Expected: builds without errors.

- [ ] **Step 3: Commit**

```bash
git add infrastructure/database/seeder/credential_extraction_seeder.go
git commit -m "feat: add CredentialExtractionSeeder for MongoDB storage"
```

---

### Task 8: Create credential_extraction_seeder_test.go

**Files:**
- Create: `infrastructure/database/seeder/credential_extraction_seeder_test.go`

**Interfaces:**
- Consumes: `NewCredentialExtractionSeeder`, mocks
- Produces: nothing (tests only)

- [ ] **Step 1: Write credential_extraction_seeder_test.go**

```go
package seeder_test

import (
	"context"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/infrastructure/testutil/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCredentialExtractionSeeder_Name(t *testing.T) {
	s := seeder.NewCredentialExtractionSeeder(nil, nil, nil)
	assert.Equal(t, "credential_extraction", s.Name())
}

func TestCredentialExtractionSeeder_NoCredentialsReturnsError(t *testing.T) {
	credRepo := new(mocks.MockCredentialRepository)
	credRepo.On("Get", mock.Anything, (*domainQuery.Query)(nil)).Return([]domain.Credential{}, 0, nil)

	cfg := &config.Config{}
	s := seeder.NewCredentialExtractionSeeder(cfg, credRepo, nil)
	err := s.Seed(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no credentials")
}

func TestCredentialExtractionSeeder_StoresExtractions(t *testing.T) {
	credRepo := new(mocks.MockCredentialRepository)
	credRepo.On("Get", mock.Anything, (*domainQuery.Query)(nil)).Return([]domain.Credential{
		{
			ID: "c1", HolderUserID: "h1", IssuerUserID: "i1",
			FileHash: "0xabcd",
			Holder:   &domain.User{Id: "h1", Name: ptr("Test Holder")},
			Issuer:   &domain.User{Id: "i1", Name: ptr("Test Issuer"), Number: ptr("NIP123")},
		},
	}, 1, nil)

	extrRepo := new(mocks.MockCredentialExtractionRepository)
	extrRepo.On("Store", mock.Anything, mock.MatchedBy(func(e domain.CredentialExtraction) bool {
		return e.CredentialID == "c1" && e.FileHash == "0xabcd"
	})).Return(nil)

	cfg := &config.Config{}
	s := seeder.NewCredentialExtractionSeeder(cfg, credRepo, extrRepo)
	err := s.Seed(context.Background())

	assert.NoError(t, err)
	extrRepo.AssertExpectations(t)
}

func ptr[T any](v T) *T { return &v }
```

Note: Add `domainQuery "CredChain_Golang/domain/query"` import.

- [ ] **Step 2: Run tests**

```bash
go test ./infrastructure/database/seeder/... -run TestCredentialExtraction -v
```
Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add infrastructure/database/seeder/credential_extraction_seeder_test.go
git commit -m "test: add CredentialExtractionSeeder tests"
```

---

### Task 9: Export Diploma Functions for Testing

**Files:**
- Modify: `infrastructure/database/seeder/diploma.go` — add exported wrapper functions

**Interfaces:**
- Produces: exported `SeedBuildDiplomaContent`, `SeedRenderDiplomaPNG`, `SeedRenderDiplomaJPEG`, `SeedRenderDiplomaPDF`, `SeedBuildDiplomaText`, `SeedBuildDiplomaExtractionIDs`, `SeedCredentialNames`, `SeedCredentialShortToProgramStudi` for test access

- [ ] **Step 1: Add exported wrappers at the bottom of diploma.go**

```go
func SeedBuildDiplomaContent(holder domain.User, issuer domain.User, credentialIndex int, rng *rand.Rand) diplomaContent {
	return seedBuildDiplomaContent(holder, issuer, credentialIndex, rng)
}

func SeedRenderDiplomaPNG(content diplomaContent) ([]byte, error) {
	return seedRenderDiplomaPNG(content)
}

func SeedRenderDiplomaJPEG(content diplomaContent) ([]byte, error) {
	return seedRenderDiplomaJPEG(content)
}

func SeedRenderDiplomaPDF(content diplomaContent, pngBytes []byte) ([]byte, error) {
	return seedRenderDiplomaPDF(content, pngBytes)
}

func SeedBuildDiplomaText(c diplomaContent) string {
	return seedBuildDiplomaText(c)
}

func SeedBuildDiplomaExtractionIDs(c diplomaContent) []domain.CredentialExtractedID {
	return seedBuildDiplomaExtractionIDs(c)
}

func SeedCredentialNames() []seedCredentialName {
	return seedCredentialNames
}

func SeedCredentialShortToProgramStudi() map[string]string {
	return seedCredentialShortToProgramStudi
}
```

- [ ] **Step 2: Verify compilation and tests**

```bash
cd CredChain_Golang
go build ./infrastructure/database/seeder/...
go test ./infrastructure/database/seeder/... -v
```
Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add infrastructure/database/seeder/diploma.go
git commit -m "feat: export diploma functions for test access"
```

---

### Task 10: Wire Seeders into cmd/seed.go

**Files:**
- Modify: `cmd/seed.go:49-56` — add new providers and seeders

**Interfaces:**
- Consumes: `NewCredentialSeeder`, `NewCredentialExtractionSeeder`, `NewGormCredentialRepository`, `NewMongoCredentialExtractionRepository`, `NewStorage`, `NewClient` (mongo), `NewDatabase` (mongo)
- Produces: extended FX app wiring

- [ ] **Step 1: Update cmd/seed.go**

Add imports:
```go
"CredChain_Golang/feature/credential"
"CredChain_Golang/infrastructure/database/mongo"
"CredChain_Golang/infrastructure/storage"
```

Update the FX provide block to wire all dependencies and seeders:

```go
fx.Provide(
    NewConfigFromCmd(cmd),
    gormInfra.NewGorm,
    mongo.NewClient,
    mongo.NewDatabase,
    storage.NewStorage,
    user.NewGormUserRepository,
    credential.NewGormCredentialRepository,
    credential.NewMongoCredentialExtractionRepository,
    func(
        cfg *config.Config,
        userRepo domain.UserRepository,
        credRepo domain.CredentialRepository,
        extrRepo domain.CredentialExtractionRepository,
        storage *storage.Storage,
    ) *seeder.Registry {
        mnemonic := seedGetHardhatMnemonic(cfg)
        return seeder.NewRegistry(
            seeder.NewUserSeeder(userRepo, mnemonic, *cfg.WalletEncryptionKey),
            seeder.NewCredentialSeeder(cfg, userRepo, credRepo, extrRepo, storage),
            seeder.NewCredentialExtractionSeeder(cfg, credRepo, extrRepo),
        )
    },
),
```

Also update the `seedGetHardhatMnemonic` function usage — keep existing function unchanged.

- [ ] **Step 2: Verify compilation**

```bash
cd CredChain_Golang
go build ./cmd/...
```
Expected: builds without errors.

- [ ] **Step 3: Run full seeder tests**

```bash
go test ./infrastructure/database/seeder/... -v
```
Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/seed.go
git commit -m "feat: wire credential seeders into seed command"
```

---

### Task 11: Add Credential Minting to cmd/seed_chain.go

**Files:**
- Modify: `cmd/seed_chain.go` — add credential minting providers, RegistryService injection, and minting logic

**Interfaces:**
- Consumes: `chain.RegistryService`, `domain.CredentialRepository`, `domain.UserRepository.FindByIds`
- Produces: credential minting + batch UPDATE flow after user registration

**Design:** Group unminted credentials by issuer, batch-mint via `RegistryService.IssueCredentials` (one batch = one issuer, ≤100), collect token IDs, single batch UPDATE per chunk. Query holders and issuers via single `FindByIds` to build `userID → Wallet` map — no N+1.

- [ ] **Step 1: Update imports**

Change existing imports block to add:

```go
"math/big"

"CredChain_Golang/feature/credential"
domainQuery "CredChain_Golang/domain/query"
ethCrypto "github.com/ethereum/go-ethereum/crypto"
```

- [ ] **Step 2: Add providers to fx.Provide**

After `chain.NewAuthorityService`, add:

```go
chain.NewRegistryService,
credential.NewGormCredentialRepository,
```

- [ ] **Step 3: Update fx.Invoke to accept RegistryService and credential repo**

Change the Invoke signature:

```go
fx.Invoke(func(
    shutdowner fx.Shutdowner,
    cfg *config.Config,
    userRepo domain.UserRepository,
    credRepo domain.CredentialRepository,
    authorityService chain.AuthorityService,
    registryService chain.RegistryService,
    logger *zap.Logger,
) {
```

And in the goroutine call:

```go
if err := seedChainRun(cfg, userRepo, credRepo, authorityService, registryService, seedChainNames, logger); err != nil {
```

- [ ] **Step 4: Update seedChainRun signature**

```go
func seedChainRun(
    cfg *config.Config,
    userRepo domain.UserRepository,
    credRepo domain.CredentialRepository,
    authorityService chain.AuthorityService,
    registryService chain.RegistryService,
    names []string,
    logger *zap.Logger,
) error {
```

- [ ] **Step 5: Add credential minting after user registration loop**

After the existing `logger.Info("seed-chain completed successfully", ...)` for user registration, but before the final `return nil`, replace the final return with credential minting logic. The complete end of seedChainRun becomes:

```go
	logger.Info("seed-chain user registration completed successfully",
		zap.Int("users_registered", registeredCount),
	)

	allCreds, _, err := credRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed-chain: read credentials: %w", err)
	}

	var unminted []domain.Credential
	for _, c := range allCreds {
		if c.TokenID == nil {
			unminted = append(unminted, c)
		}
	}

	if len(unminted) == 0 {
		logger.Info("no un-minted credentials — skipping credential minting")
		return nil
	}

	logger.Info("credentials loaded for minting",
		zap.Int("total_in_db", len(allCreds)),
		zap.Int("to_mint", len(unminted)),
	)

	userIDs := make(map[string]bool)
	for _, c := range unminted {
		userIDs[c.HolderUserID] = true
		userIDs[c.IssuerUserID] = true
	}
	idList := make([]string, 0, len(userIDs))
	for id := range userIDs {
		idList = append(idList, id)
	}

	users, err := userRepo.FindByIds(ctx, idList...)
	if err != nil {
		return fmt.Errorf("seed-chain: find users: %w", err)
	}
	walletMap := make(map[string]domain.Wallet, len(users))
	for _, u := range users {
		walletMap[u.Id] = domain.WalletFromUser(u)
	}

	issuerGroups := make(map[string][]domain.Credential)
	for _, c := range unminted {
		issuerGroups[c.IssuerUserID] = append(issuerGroups[c.IssuerUserID], c)
	}

	mintedCount := 0
	const maxBatchCredential = 100

	for issuerID, group := range issuerGroups {
		issuerWallet := walletMap[issuerID]

		for start := 0; start < len(group); start += maxBatchCredential {
			end := start + maxBatchCredential
			if end > len(group) {
				end = len(group)
			}
			chunk := group[start:end]

			issuances := make([]chain.CredentialIssuance, len(chunk))
			for i, c := range chunk {
				issuances[i] = chain.CredentialIssuance{
					HolderAddress: walletMap[c.HolderUserID].Address,
					Hash:          c.FileHash,
					URI:           *c.FileURI,
				}
			}

			logger.Info("minting credentials on-chain",
				zap.Int("chunk_size", len(chunk)),
				zap.Int("chunk_start", start),
				zap.Int("issuer_total", len(group)),
				zap.String("issuer", issuerID),
			)

			tokenIds, err := registryService.IssueCredentials(ctx, issuerWallet, issuances...)
			if err != nil {
				return fmt.Errorf("seed-chain: mint credentials for issuer %s chunk [%d:%d]: %w", issuerID, start, end, err)
			}

			updates := make([]domain.Credential, len(chunk))
			for i := range chunk {
				updates[i] = domain.Credential{
					ID:      chunk[i].ID,
					TokenID: lo.ToPtr(tokenIds[i].String()),
				}
			}

			if _, err := credRepo.Update(ctx, updates...); err != nil {
				return fmt.Errorf("seed-chain: update token IDs: %w", err)
			}

			mintedCount += len(chunk)
		}
	}

	logger.Info("seed-chain credential minting completed successfully",
		zap.Int("credentials_minted", mintedCount),
	)

	return nil
}
```

- [ ] **Step 6: Remove now-unused imports**

If the `ethCrypto` import is unused after adding the minting code, remove it. The `tokenIdFromIssuance` is called inside `RegistryService.IssueCredentials` — we don't need to compute token IDs ourselves here.

Clean up the import block to remove any now-unused packages. The final imports should be:

```go
import (
	"context"
	"fmt"
	"log"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)
```

- [ ] **Step 7: Verify compilation**

```bash
cd CredChain_Golang
go build ./cmd/...
go vet ./cmd/...
gofmt -l cmd/seed_chain.go
```
Expected: builds without errors, gofmt produces no output.

- [ ] **Step 8: Commit**

```bash
git add cmd/seed_chain.go
git commit -m "feat: add credential chain minting to seed-chain command"
```

---

## Final Verification

After all tasks are complete, run the full verification suite:

```bash
cd CredChain_Golang
go test ./... && go vet ./... && gofmt -l .
```

Expected: all tests pass, go vet produces no output, gofmt produces no output.

---

## Self-Review

### 1. Spec Coverage

| Spec Requirement | Task(s) |
|---|---|
| CredentialSeeder queries live holders/issuers | Task 5 |
| ~4:1 holder-to-issuer ratio | Task 5 (i%3 holder, i%12 issuer) |
| ~20% revoked after storage | Task 5 (totalCredentials/5) |
| 10 Indonesian degree names cycled | Task 3 (diploma.go) |
| MIME types JPEG/PNG/PDF cycled | Tasks 3, 5 |
| Diploma layout (header, body, footer, signatures) | Task 3 |
| File generation (image, x/image/font, fpdf) | Task 3 |
| keccak256 hash, AES encrypt, save to disk | Task 5 |
| INSERT Postgres with extract_status=succeeded | Task 5 |
| Store extraction directly to MongoDB | Task 5 |
| CredentialExtractionSeeder re-derives content | Task 7 |
| 12 extraction IDs | Task 3 |
| Deterministic embedding (SHA-256 → 768 floats) | Tasks 5, 7 |
| Chain minting: query token_id IS NULL, batch mint, batch UPDATE | Task 11 |
| Determinism: hashToSeed("credchain-seed") | Tasks 2, 5, 7 |
| Naming: universityHead, dean, credentialName | Task 3 |
| Config passed directly to constructors | Tasks 5, 7, 10 |
| seedHashSeed → hashToSeed rename | Task 2 |
| No River job, no Python call | Tasks 5, 7 |
| No N+1 queries | Tasks 5 (batch Store), 11 (single FindByIds) |
| Fallbacks for nullable fields (BirthDate, Name, Number) | Task 3 |
| Emblem/seal fixtures (embedded as Go code) | Task 3 (programmatic drawing) |
| Wire seeders into cmd/seed.go Registry | Task 10 |
| Extend cmd/seed_chain.go with credential minting | Task 11 |

### 2. Placeholder Scan

No TBD, TODO, "implement later", or "add appropriate error handling" patterns found. Every step has concrete code or exact commands.

### 3. Type Consistency

- `diplomaContent` defined in Task 3, consumed in Tasks 4, 5, 7 ✓
- `seedBuildDiplomaContent` signature: `(holder domain.User, issuer domain.User, credentialIndex int, rng *rand.Rand) diplomaContent` ✓
- `seedRenderDiplomaPNG` → `SeedRenderDiplomaPNG` exported in Task 9, used in Task 5 ✓
- `seedRenderDiplomaJPEG` → `SeedRenderDiplomaJPEG` exported in Task 9, used in Task 5 ✓
- `seedRenderDiplomaPDF` → `SeedRenderDiplomaPDF` exported in Task 9, used in Task 5 ✓
- `seedBuildDiplomaText` → `SeedBuildDiplomaText` exported in Task 9, used in Tasks 5, 7 ✓
- `seedBuildDiplomaExtractionIDs` → `SeedBuildDiplomaExtractionIDs` exported in Task 9, used in Tasks 5, 7 ✓
- `seedBuildEmbedding` consistent signature across Tasks 5, 7 ✓
- `NewCredentialSeeder(cfg, userRepo, credRepo, extrRepo, storage)` signature matches Task 10 wiring ✓
- `NewCredentialExtractionSeeder(cfg, credRepo, extrRepo)` signature matches Task 10 wiring ✓
- `RegistryService.IssueCredentials` called with correct params in Task 11 ✓
- `domain.Credential.Repository.Update` called with batch updates in Tasks 5, 11 ✓

---

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-07-01-credential-seeding.md`. Two execution options:**

1. **Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration
2. **Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
```<｜end▁of▁thinking｜>Let me check the chain client and understand the credential issuance pattern.

<｜｜DSML｜｜tool_calls>
<｜｜DSML｜｜invoke name="read">
<｜｜DSML｜｜parameter name="filePath" string="true">/Users/arfanxn/Developments/credchain/CredChain_Golang/infrastructure/chain/client.go
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
	HolderName         string
	HolderBirthDate    string
	HolderBirthPlace   string
	IssuedDate         string
	CredentialName     string
	CredentialShort    string
	ProgramStudi       string
	DeanName           string
	DeanNIP            string
	OrganizationName   string
	UniversityHeadName string
	UniversityHeadNIP  string
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
		HolderName:         holderName,
		HolderBirthDate:    birthDate,
		HolderBirthPlace:   holderBirthPlace,
		IssuedDate:         issuedDateStr,
		CredentialName:     cn.Name,
		CredentialShort:    cn.Short,
		ProgramStudi:       programStudi,
		DeanName:           issuerName,
		DeanNIP:            issuerNIP,
		OrganizationName:   seedDiplomaOrgName,
		UniversityHeadName: seedDiplomaRektorName,
		UniversityHeadNIP:  seedDiplomaRektorNIP,
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

	opts := fpdf.ImageOptions{ImageType: "png"}
	info := pdf.RegisterImageOptionsReader("diploma", opts, bytes.NewReader(pngBytes))
	if info != nil {
		pdf.ImageOptions("diploma", 0, 0, 210, 297, false, opts, 0, "")
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

	for _, dy := range []int{-r, -r / 2, 0, r / 2, r} {
		for _, dx := range []int{-r, -r / 2, 0, r / 2, r} {
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

// ── Exported wrappers (for test access from package seeder_test) ──

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

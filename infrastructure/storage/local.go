package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/oklog/ulid/v2"
)

// Storage handles local file saving. Callers manage subdirectory structure
// as needed (e.g. credential-issuance code creates ./uploads/credentials/).
type Storage struct {
	BaseDir string
}

// NewStorage creates the base uploads directory if it doesn't exist.
func NewStorage() (*Storage, error) {
	dir := "uploads"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Storage{BaseDir: dir}, nil
}

// SaveFile takes an uploaded multipart file, assigns a ULID name, and saves
// it locally.
func (s *Storage) SaveFile(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s%s", ulid.Make().String(), ext)
	outPath := filepath.Join(s.BaseDir, filename)

	dst, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	return outPath, nil
}

// SaveBytes persists raw bytes under a ULID-derived filename and returns the
// resulting file path. Used by credential issuance where we already have the
// file bytes in memory (for hashing) before persisting.
func (s *Storage) SaveBytes(data []byte, ext string) (string, error) {
	filename := fmt.Sprintf("%s%s", ulid.Make().String(), ext)
	outPath := filepath.Join(s.BaseDir, filename)
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return "", err
	}
	return outPath, nil
}

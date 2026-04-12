package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/oklog/ulid/v2"
)

// Storage handles local file saving
type Storage struct {
	BaseDir string
}

// NewStorage creates the uploads directory if it doesn't exist
func NewStorage() (*Storage, error) {
	dir := "uploads"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Storage{BaseDir: dir}, nil
}

// SaveFile takes an uploaded multipart file, assigns a ULID name, and saves it locally
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

package storage

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"CredChain_Golang/config"

	"go.uber.org/fx"
)

type Storage struct {
	Config *config.Config
}

type StorageParams struct {
	fx.In
	Config *config.Config
}

func NewStorage(p StorageParams) (*Storage, error) {
	baseDir := *p.Config.StoragePath
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &Storage{Config: p.Config}, nil
}

func (s *Storage) fullPath(path string) string {
	return filepath.Join(*s.Config.StoragePath, path)
}

// SaveFile takes an uploaded multipart file, saves it under StoragePath/path.
// MkdirAll ensures intermediate directories exist. Returns path on success.
func (s *Storage) SaveFile(file *multipart.FileHeader, path string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	full := s.fullPath(path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}

	dst, err := os.Create(full)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}
	return path, nil
}

// SaveBytes persists raw data under StoragePath/path. MkdirAll ensures
// intermediate directories exist. Returns path on success.
func (s *Storage) SaveBytes(data []byte, path string) (string, error) {
	full := s.fullPath(path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ReadBytes returns the contents of StoragePath/path.
func (s *Storage) ReadBytes(path string) ([]byte, error) {
	return os.ReadFile(s.fullPath(path))
}

// Delete removes StoragePath/path. Best-effort: returns nil if file not found.
func (s *Storage) Delete(path string) error {
	err := os.Remove(s.fullPath(path))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

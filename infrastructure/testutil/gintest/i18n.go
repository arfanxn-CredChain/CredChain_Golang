package gintest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	testBundleOnce sync.Once
	testBundle     *i18n.Bundle
)

// LoadTestI18nBundle loads the real locales/ directory relative to repo root.
// Uses runtime.Caller to locate locales regardless of test cwd. Cached per process.
func LoadTestI18nBundle(t *testing.T) *i18n.Bundle {
	t.Helper()
	testBundleOnce.Do(func() {
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatalf("runtime.Caller failed")
		}
		// From infrastructure/testutil/gintest/i18n.go up to CredChain_Golang/
		repoRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
		localesDir := filepath.Join(repoRoot, "locales")

		bundle := i18n.NewBundle(language.Indonesian)
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

		files, err := os.ReadDir(localesDir)
		if err != nil {
			t.Fatalf("ReadDir locales (%s): %v", localesDir, err)
		}
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
				if _, err := bundle.LoadMessageFile(filepath.Join(localesDir, f.Name())); err != nil {
					t.Fatalf("LoadMessageFile %s: %v", f.Name(), err)
				}
			}
		}
		testBundle = bundle
	})
	return testBundle
}

package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

const (
	LocalizerContextKey = "localizer"
)

func NewBundle() (*i18n.Bundle, error) {
	bundle := i18n.NewBundle(language.Indonesian)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load all files in locales directory
	localesDir := "./locales"
	files, err := os.ReadDir(localesDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			_, err := bundle.LoadMessageFile(filepath.Join(localesDir, file.Name()))
			if err != nil {
				return nil, err
			}
		}
	}

	return bundle, nil
}

// GetLocalizer retrieves the *i18n.Localizer from the Gin context.
func GetLocalizer(c *gin.Context) *i18n.Localizer {
	val, exists := c.Get(LocalizerContextKey)
	if !exists {
		return nil
	}
	localizer, ok := val.(*i18n.Localizer)
	if !ok {
		return nil
	}
	return localizer
}

// SetLocalizer injects the *i18n.Localizer into the Gin context.
func SetLocalizer(c *gin.Context, localizer *i18n.Localizer) {
	c.Set(LocalizerContextKey, localizer)
}

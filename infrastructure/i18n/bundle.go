package i18n

import (
	"CredChain_Golang/config"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"go.uber.org/fx"
)

const (
	I18nLocalizerContextKey = "i18n_localizer"
)

type I18nBundleParams struct {
	fx.In
	Config *config.Config
}

func NewI18nBundle(p I18nBundleParams) (*i18n.Bundle, error) {
	bundle := i18n.NewBundle(language.Indonesian)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	localesDir := *p.Config.I18nLocalesDir
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

func GetI18nLocalizer(c *gin.Context) *i18n.Localizer {
	val, exists := c.Get(I18nLocalizerContextKey)
	if !exists {
		return nil
	}
	localizer, ok := val.(*i18n.Localizer)
	if !ok {
		return nil
	}
	return localizer
}

func SetI18nLocalizer(c *gin.Context, localizer *i18n.Localizer) {
	c.Set(I18nLocalizerContextKey, localizer)
}

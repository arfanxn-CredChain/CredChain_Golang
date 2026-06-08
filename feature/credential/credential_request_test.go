package credential

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCredentialReExtractRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := CredentialReExtractRequest{Ids: []string{"01J0000000000000000000000A"}}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty", func(t *testing.T) {
		assert.Error(t, CredentialReExtractRequest{Ids: []string{}}.Validate())
	})
	t.Run("too many", func(t *testing.T) {
		ids := make([]string, 101)
		for i := range ids {
			ids[i] = "x"
		}
		assert.Error(t, CredentialReExtractRequest{Ids: ids}.Validate())
	})
}

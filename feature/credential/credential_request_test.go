package credential

import (
	"strings"
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

func TestCredentialIssueInput_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "01J0000000000000000000000A", Name: "Degree"}
		assert.NoError(t, in.Validate())
	})
	t.Run("missing holder", func(t *testing.T) {
		in := CredentialIssueInput{Name: "Degree"}
		assert.Error(t, in.Validate())
	})
	t.Run("missing name", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "01J0000000000000000000000A"}
		assert.Error(t, in.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "h", Name: strings.Repeat("a", 257)}
		assert.Error(t, in.Validate())
	})
}

func TestCredentialIssueInput_ToDomain(t *testing.T) {
	meta := map[string]any{"k": "v"}
	in := CredentialIssueInput{HolderUserID: "holder-1", Name: "Degree", Meta: meta}
	got := in.ToDomain()
	assert.Equal(t, "holder-1", got.HolderUserID)
	assert.Equal(t, "Degree", got.Name)
	assert.Equal(t, meta, got.Meta)
}

func TestCredentialIssueRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := CredentialIssueRequest{Items: []CredentialIssueInput{
			{HolderUserID: "h", Name: "n"},
		}}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty items", func(t *testing.T) {
		r := CredentialIssueRequest{Items: []CredentialIssueInput{}}
		assert.Error(t, r.Validate())
	})
	t.Run("too many items", func(t *testing.T) {
		items := make([]CredentialIssueInput, 101)
		for i := range items {
			items[i] = CredentialIssueInput{HolderUserID: "h", Name: "n"}
		}
		r := CredentialIssueRequest{Items: items}
		assert.Error(t, r.Validate())
	})
	t.Run("invalid nested item", func(t *testing.T) {
		r := CredentialIssueRequest{Items: []CredentialIssueInput{{Name: "no-holder"}}}
		assert.Error(t, r.Validate())
	})
}

func TestCredentialIssueRequest_ToDomain(t *testing.T) {
	r := CredentialIssueRequest{Items: []CredentialIssueInput{
		{HolderUserID: "h1", Name: "n1"},
		{HolderUserID: "h2", Name: "n2"},
	}}
	got := r.ToDomain()
	assert.Len(t, got, 2)
	assert.Equal(t, "h1", got[0].HolderUserID)
	assert.Equal(t, "n2", got[1].Name)
}

func TestCredentialRevokeRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, CredentialRevokeRequest{Ids: []string{"01J0"}}.Validate())
	})
	t.Run("empty", func(t *testing.T) {
		assert.Error(t, CredentialRevokeRequest{Ids: []string{}}.Validate())
	})
	t.Run("too many", func(t *testing.T) {
		ids := make([]string, 101)
		for i := range ids {
			ids[i] = "x"
		}
		assert.Error(t, CredentialRevokeRequest{Ids: ids}.Validate())
	})
}

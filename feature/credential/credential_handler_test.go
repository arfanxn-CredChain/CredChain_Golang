package credential

import (
	"mime/multipart"
	"testing"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestParseItemIndex(t *testing.T) {
	tests := []struct {
		key     string
		wantIdx int
		wantOK  bool
	}{
		{"items[0][holder_user_id]", 0, true},
		{"items[99][name]", 99, true},
		{"items[abc][x]", 0, false},
		{"not_items[0][x]", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseItemIndex(tt.key)
		assert.Equal(t, tt.wantOK, ok, "key=%s", tt.key)
		if tt.wantOK {
			assert.Equal(t, tt.wantIdx, got, "key=%s", tt.key)
		}
	}
}

func TestMapCredentialsToResponse(t *testing.T) {
	creds := []domain.Credential{
		{ID: "c1", Name: "n1"},
		{ID: "c2", Name: "n2"},
	}
	out := mapCredentialsToResponse(creds)
	assert.Len(t, out, 2)
	assert.Equal(t, "c1", out[0].ID)
	assert.Equal(t, "n2", out[1].Name)
}

func TestMapCredentialsToResponse_Empty(t *testing.T) {
	out := mapCredentialsToResponse([]domain.Credential{})
	assert.Len(t, out, 0)
}

func TestBuildIssueItems(t *testing.T) {
	form := &multipart.Form{
		Value: map[string][]string{
			"items[0][holder_user_id]": {"holder-1"},
			"items[0][name]":           {"Degree"},
			"items[1][holder_user_id]": {"holder-2"},
			"items[1][name]":           {"Diploma"},
		},
		File: map[string][]*multipart.FileHeader{},
	}
	items, err := buildIssueItems(form)
	assert.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "holder-1", items[0].HolderUserID)
	assert.Equal(t, "Degree", items[0].Name)
	assert.Equal(t, "holder-2", items[1].HolderUserID)
}

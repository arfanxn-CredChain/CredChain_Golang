package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Regression: a revoked credential has revoker_user_id set, but verify never
// preloads the revoker. ToDomain must NOT fabricate an empty Revoker from the
// zero-value association — doing so surfaced a phantom blank avatar in the UI.
func TestCredential_ToDomain_NoPhantomRelationsWhenUnpreloaded(t *testing.T) {
	revokerID := "01REVOKER"
	m := Credential{
		Id:            "01CRED",
		HolderUserId:  "01HOLDER",
		IssuerUserId:  "01ISSUER",
		RevokerUserId: &revokerID,
		// HolderUser / IssuerUser / RevokerUser left as zero User{} (not preloaded).
	}

	d := m.ToDomain()

	assert.Equal(t, &revokerID, d.RevokerUserID, "FK still mapped")
	assert.Nil(t, d.Holder, "un-preloaded holder must be nil")
	assert.Nil(t, d.Issuer, "un-preloaded issuer must be nil")
	assert.Nil(t, d.Revoker, "un-preloaded revoker must be nil (no phantom empty user)")
}

func TestCredential_ToDomain_MapsPreloadedRelations(t *testing.T) {
	revokerID := "01REVOKER"
	m := Credential{
		Id:            "01CRED",
		HolderUserId:  "01HOLDER",
		IssuerUserId:  "01ISSUER",
		RevokerUserId: &revokerID,
		HolderUser:    User{Id: "01HOLDER"},
		IssuerUser:    User{Id: "01ISSUER"},
		RevokerUser:   User{Id: "01REVOKER"},
	}

	d := m.ToDomain()

	assert.NotNil(t, d.Holder)
	assert.NotNil(t, d.Issuer)
	assert.NotNil(t, d.Revoker)
	assert.Equal(t, "01REVOKER", d.Revoker.Id)
}

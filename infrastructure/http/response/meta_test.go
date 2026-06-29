package response

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeta_JSONMarshal(t *testing.T) {
	m := Meta{
		IssuingOrganizationName: "University of Indonesia",
		AuthorityContract:       "0xAAA",
		RegistryContract:        "0xBBB",
		ChainID:                 137,
		LastBlock:               42000000,
	}
	b, err := json.Marshal(m)
	require.NoError(t, err)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &out))
	assert.Equal(t, "University of Indonesia", out["issuing_organization_name"])
	assert.Equal(t, "0xAAA", out["authority_contract"])
	assert.Equal(t, "0xBBB", out["registry_contract"])
	assert.Equal(t, float64(137), out["chain_id"])
	assert.Equal(t, float64(42000000), out["last_block"])
}

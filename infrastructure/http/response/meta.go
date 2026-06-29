package response

type Meta struct {
	IssuingOrganizationName string `json:"issuing_organization_name"`
	AuthorityContract       string `json:"authority_contract"`
	RegistryContract        string `json:"registry_contract"`
	ChainID                 uint64 `json:"chain_id"`
	LastBlock               uint64 `json:"last_block"`
}

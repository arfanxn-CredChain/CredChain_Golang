package credential

import (
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/ai"
	"CredChain_Golang/infrastructure/chain"
	"CredChain_Golang/infrastructure/database"
	"CredChain_Golang/infrastructure/storage"

	"go.uber.org/fx"
)

type Service struct {
	repo   domain.CredentialRepository
	mongo  *database.MongoDB
	local  *storage.Storage
	ipfs   *storage.IPFSClient
	gemini *ai.Client
	chain  *chain.Client
}

type CredServiceParams struct {
	fx.In
	Repo   domain.CredentialRepository
	Mongo  *database.MongoDB
	Local  *storage.Storage
	Ipfs   *storage.IPFSClient
	Gemini *ai.Client
	Chain  *chain.Client
}

func NewService(p CredServiceParams) *Service {
	return &Service{
		repo:   p.Repo,
		mongo:  p.Mongo,
		local:  p.Local,
		ipfs:   p.Ipfs,
		gemini: p.Gemini,
		chain:  p.Chain,
	}
}

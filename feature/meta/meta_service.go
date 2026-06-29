package meta

import (
	"context"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain"
	"CredChain_Golang/infrastructure/http/response"

	"go.uber.org/fx"
)

type chainClient interface {
	BlockNumber(ctx context.Context) (uint64, error)
	ChainID(ctx context.Context) (uint64, error)
}

type MetaService interface {
	Get(ctx context.Context) (*response.Meta, error)
}

type metaService struct {
	cfg   *config.Config
	chain chainClient
}

type MetaServiceParams struct {
	fx.In
	Config      *config.Config
	ChainClient *chain.Client
}

func NewMetaService(p MetaServiceParams) MetaService {
	return newMetaService(p.Config, p.ChainClient)
}

func newMetaService(cfg *config.Config, chain chainClient) *metaService {
	return &metaService{cfg: cfg, chain: chain}
}

func (s *metaService) Get(ctx context.Context) (*response.Meta, error) {
	lastBlock, err := s.chain.BlockNumber(ctx)
	if err != nil {
		return nil, domain.NewError(domain.CodeMetaInternal, domain.WithError(err))
	}

	chainID, err := s.chain.ChainID(ctx)
	if err != nil {
		return nil, domain.NewError(domain.CodeMetaInternal, domain.WithError(err))
	}

	return &response.Meta{
		IssuingOrganizationName: *s.cfg.IssuingOrganizationName,
		AuthorityContract:       *s.cfg.AuthorityContract,
		RegistryContract:        *s.cfg.RegistryContract,
		ChainID:                 chainID,
		LastBlock:               lastBlock,
	}, nil
}

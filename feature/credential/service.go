package credential

import (
	"CredChain_Golang/domain"

	"go.uber.org/fx"
)

type Service struct {
	repo domain.CredentialRepository
}

type CredServiceParams struct {
	fx.In
	Repo domain.CredentialRepository
}

func NewCredentialService(p CredServiceParams) *Service {
	return &Service{
		repo: p.Repo,
	}
}

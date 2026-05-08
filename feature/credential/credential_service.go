package credential

import (
	"CredChain_Golang/domain"
	"go.uber.org/fx"
)

type CredentialService interface {}

type credentialService struct {
	repo domain.CredentialRepository
}

type CredentialServiceParams struct {
	fx.In
	Repo domain.CredentialRepository
}

func NewCredentialService(p CredentialServiceParams) CredentialService {
	return &credentialService{repo: p.Repo}
}

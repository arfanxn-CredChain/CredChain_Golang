package chain

import (
	"context"
	"fmt"
	"time"

	"CredChain_Golang/domain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// RoleService provides access to user roles stored on the CredentialAuthority
// blockchain contract. It is the source of truth for authorization decisions.
//
// Current methods are read-only (Find, Verify). Future write operations such as
// UpdateRole, TransferSuperAdmin, and BatchUpdateRoles will be added as the
// credential issuance and role management features are implemented.
type RoleService interface {
	Find(ctx context.Context, walletAddress string) (domain.Role, error)
	Verify(ctx context.Context, walletAddress string, minRole domain.Role) error
}

type roleService struct {
	authority *Authority
	timeout   time.Duration
}

func NewRoleService(client *Client) RoleService {
	return &roleService{
		authority: client.Authority,
		timeout:   5 * time.Second,
	}
}

func (s *roleService) Find(ctx context.Context, walletAddress string) (domain.Role, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	addr := common.HexToAddress(walletAddress)
	roleUint8, err := s.authority.UserToRole(&bind.CallOpts{Context: ctx}, addr)
	if err != nil {
		return domain.Role(""), fmt.Errorf("failed to fetch on-chain role: %w", err)
	}
	return domain.RoleFromUint8(roleUint8), nil
}

func (s *roleService) Verify(ctx context.Context, walletAddress string, minRole domain.Role) error {
	role, err := s.Find(ctx, walletAddress)
	if err != nil {
		return domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
	}
	if role.Rank() < minRole.Rank() {
		return domain.NewError(domain.CodeAuthForbidden)
	}
	return nil
}

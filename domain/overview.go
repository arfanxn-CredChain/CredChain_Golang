package domain

import (
	"context"

	domainQuery "CredChain_Golang/domain/query"
)

type OverviewCredentialCounts struct {
	Total   int
	Active  int
	Revoked int
	Pending int
	Failed  int
}

type OverviewUserCounts struct {
	Total      int
	Holder     int
	Issuer     int
	Admin      int
	SuperAdmin int
	Active     int
	Trashed    int
}

type OverviewRepository interface {
	CredentialCounts(ctx context.Context, q *domainQuery.Query, holderUserID *string) (*OverviewCredentialCounts, error)
	UserCounts(ctx context.Context, q *domainQuery.Query) (*OverviewUserCounts, error)
}

package mocks

import (
	"context"

	"CredChain_Golang/infrastructure/oauth"

	"github.com/stretchr/testify/mock"
	"google.golang.org/api/idtoken"
)

type MockGoogleOAuthClient struct {
	mock.Mock
}

func (m *MockGoogleOAuthClient) Validate(ctx context.Context, idToken, audience string) (*idtoken.Payload, error) {
	args := m.Called(ctx, idToken, audience)
	if v := args.Get(0); v != nil {
		return v.(*idtoken.Payload), args.Error(1)
	}
	return nil, args.Error(1)
}

var _ oauth.GoogleOAuthClient = (*MockGoogleOAuthClient)(nil)

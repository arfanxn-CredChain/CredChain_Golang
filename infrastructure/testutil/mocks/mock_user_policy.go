package mocks

import (
	"context"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/mock"
)

type MockUserPolicy struct {
	mock.Mock
}

func (m *MockUserPolicy) Store(ctx context.Context, users ...domain.User) error {
	args := m.Called(ctx, users)
	return args.Error(0)
}

func (m *MockUserPolicy) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) error {
	args := m.Called(ctx, updates)
	return args.Error(0)
}

func (m *MockUserPolicy) Delete(ctx context.Context, ids ...string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

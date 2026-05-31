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

func (m *MockUserPolicy) UpdatePreFetch(ctx context.Context, users ...domain.User) error {
	args := m.Called(ctx, users)
	return args.Error(0)
}

func (m *MockUserPolicy) UpdatePostFetch(ctx context.Context, targets []domain.User, updates []domain.User) error {
	args := m.Called(ctx, targets, updates)
	return args.Error(0)
}

func (m *MockUserPolicy) UpdateRolePreFetch(ctx context.Context, updates ...domain.UserRoleUpdate) error {
	args := m.Called(ctx, updates)
	return args.Error(0)
}

func (m *MockUserPolicy) UpdateRolePostFetch(ctx context.Context, targets []domain.User, updates ...domain.UserRoleUpdate) error {
	args := m.Called(ctx, targets, updates)
	return args.Error(0)
}

func (m *MockUserPolicy) DeletePreFetch(ctx context.Context, ids ...string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockUserPolicy) DeletePostFetch(ctx context.Context, targets []domain.User) error {
	args := m.Called(ctx, targets)
	return args.Error(0)
}

func (m *MockUserPolicy) TransferSuperAdminPreFetch(ctx context.Context, targetId string) error {
	args := m.Called(ctx, targetId)
	return args.Error(0)
}

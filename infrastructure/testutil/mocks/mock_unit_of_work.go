package mocks

import (
	"context"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/mock"
)

type MockUnitOfWork struct {
	mock.Mock
}

func (m *MockUnitOfWork) Execute(ctx context.Context, fn func(domain.UnitOfWork) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockUnitOfWork) User() domain.UserRepository {
	args := m.Called()
	return args.Get(0).(domain.UserRepository)
}

func (m *MockUnitOfWork) Credential() domain.CredentialRepository {
	args := m.Called()
	return args.Get(0).(domain.CredentialRepository)
}

func (m *MockUnitOfWork) UserToken() domain.UserTokenRepository {
	args := m.Called()
	return args.Get(0).(domain.UserTokenRepository)
}

// RunUnitOfWorkFn configures the mock to invoke the function passed to Execute.
func RunUnitOfWorkFn(m *MockUnitOfWork, innerUoW domain.UnitOfWork) {
	m.On("Execute", mock.Anything, mock.AnythingOfType("func(domain.UnitOfWork) error")).
		Return(nil).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(domain.UnitOfWork) error)
			_ = fn(innerUoW)
		})
}

var _ domain.UnitOfWork = (*MockUnitOfWork)(nil)

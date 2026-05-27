package context

import (
	"context"
	"testing"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func testUser() *domain.User {
	return &domain.User{Id: "u1", Email: "a@b.com", Role: domain.RoleHolder}
}

func TestGetUser_Found(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserKey, testUser())
	u, err := GetUser(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "u1", u.Id)
}

func TestGetUser_NotInContext(t *testing.T) {
	_, err := GetUser(context.Background())
	assert.Error(t, err)
}

func TestGetUser_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserKey, "not-a-user")
	_, err := GetUser(ctx)
	assert.Error(t, err)
}

func TestGetUser_NilPointer(t *testing.T) {
	var u *domain.User
	ctx := context.WithValue(context.Background(), UserKey, u)
	_, err := GetUser(ctx)
	assert.Error(t, err)
}

func TestMustGetUser_Found(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserKey, testUser())
	u := MustGetUser(ctx)
	assert.Equal(t, "u1", u.Id)
}

func TestMustGetUser_Panics(t *testing.T) {
	assert.Panics(t, func() {
		MustGetUser(context.Background())
	})
}

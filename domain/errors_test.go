package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewError_OnlyCode(t *testing.T) {
	e := NewError(CodeSystemInternal)
	assert.Equal(t, CodeSystemInternal, e.Code)
	assert.Nil(t, e.Err)
	assert.Nil(t, e.Metadata)
	assert.Equal(t, "operation failed", e.Error())
}

func TestNewError_WithError(t *testing.T) {
	wrapped := errors.New("boom")
	e := NewError(CodeSystemInternal, WithError(wrapped))
	assert.Same(t, wrapped, e.Err)
	assert.Equal(t, "boom", e.Error())
}

func TestNewError_WithMetadata(t *testing.T) {
	e := NewError(CodeUserStoreEmailDuplicateInBatch,
		WithMetadata("emails", []string{"a@x.com", "b@x.com"}))
	assert.Equal(t, []string{"a@x.com", "b@x.com"}, e.Metadata["emails"])
}

func TestNewError_WithMultipleMetadata(t *testing.T) {
	e := NewError(CodeUserRoleAdminUpdatePeerForbidden,
		WithMetadata("auth_user_id", "u1"),
		WithMetadata("target_user_id", "u2"))
	assert.Equal(t, "u1", e.Metadata["auth_user_id"])
	assert.Equal(t, "u2", e.Metadata["target_user_id"])
}

func TestNewError_WithMetadataMap(t *testing.T) {
	e := NewError(CodeSystemInternal, WithMetadataMap(map[string]any{"k1": 1, "k2": "v2"}))
	assert.Equal(t, 1, e.Metadata["k1"])
	assert.Equal(t, "v2", e.Metadata["k2"])
}

func TestNewError_WithMetadataMap_MergesWithSingle(t *testing.T) {
	e := NewError(CodeSystemInternal,
		WithMetadata("a", 1),
		WithMetadataMap(map[string]any{"b": 2}))
	assert.Equal(t, 1, e.Metadata["a"])
	assert.Equal(t, 2, e.Metadata["b"])
}

func TestError_Unwrap(t *testing.T) {
	wrapped := errors.New("original")
	e := NewError(CodeSystemInternal, WithError(wrapped))
	assert.True(t, errors.Is(e, wrapped))
	assert.Same(t, wrapped, errors.Unwrap(e))
}

func TestError_Unwrap_Nil(t *testing.T) {
	e := NewError(CodeSystemInternal)
	assert.Nil(t, errors.Unwrap(e))
}

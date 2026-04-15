package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRole_Rank(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected int
	}{
		{
			name:     "Super Admin Rank",
			role:     RoleSuperAdmin,
			expected: 4,
		},
		{
			name:     "Admin Rank",
			role:     RoleAdmin,
			expected: 3,
		},
		{
			name:     "Issuer Rank",
			role:     RoleIssuer,
			expected: 2,
		},
		{
			name:     "Holder Rank",
			role:     RoleHolder,
			expected: 1,
		},
		{
			name:     "Unknown Role Rank",
			role:     Role("unknown_role"),
			expected: 0,
		},
		{
			name:     "Empty Role Rank",
			role:     Role(""),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.Rank())
		})
	}

	// Additional comparison tests
	t.Run("Rank Comparisons", func(t *testing.T) {
		assert.True(t, RoleSuperAdmin.Rank() > RoleAdmin.Rank())
		assert.True(t, RoleAdmin.Rank() > RoleIssuer.Rank())
		assert.True(t, RoleIssuer.Rank() > RoleHolder.Rank())
		assert.True(t, RoleHolder.Rank() > Role("unknown").Rank())
	})
}

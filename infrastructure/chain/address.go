package chain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// mustHexToAddress converts a hex string to common.Address.
// It assumes the caller has validated the address format.
// Invalid addresses are converted to zero address (common.HexToAddress behavior).
//
// Parameters:
//   - s: Ethereum address as a hex string with "0x" prefix (42 characters)
//
// Returns:
//   - common.Address: The converted address (zero address if input is invalid)
func mustHexToAddress(s string) common.Address {
	return common.HexToAddress(s)
}

// validateHexToAddress converts a hex string to common.Address with validation.
// It returns an error if the address format is invalid.
//
// Parameters:
//   - s: Ethereum address as a hex string with "0x" prefix (42 characters)
//
// Returns:
//   - common.Address: The converted address
//   - error: Non-nil if the address format is invalid
func validateHexToAddress(s string) (common.Address, error) {
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("invalid hex address: %s", s)
	}
	return common.HexToAddress(s), nil
}

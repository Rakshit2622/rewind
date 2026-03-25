package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Compute returns the SHA256 hex digest of data.
func Compute(data []byte) (string, error) {
	if data == nil {
		return "", fmt.Errorf("empty data")
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

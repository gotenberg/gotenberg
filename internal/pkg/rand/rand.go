package rand

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Get returns a random string.
func Get() (string, error) {
	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", fmt.Errorf("creating random string: %v", err)
	}
	return hex.EncodeToString(randBytes), nil
}

package gotenberg

import (
	"fmt"
)

// EncryptionOptions represents options for encryption.
type EncryptionOptions struct {
	// Password for rights of PDF file.
	// Required, but can be empty.
	OwnerPassword string

	// Password for opening PDF file.
	// Required, but can be empty.
	UserPassword string

	// Encryption key length.
	// Required.
	KeyLength int
}

func NewEncryptionOptions(keyLength int, ownerPassword, userPassword string) *EncryptionOptions {
	//check for valid KeyLength
	if !isValidKeyLength(keyLength) {
		panic(fmt.Sprintf("Invalid keyLength specified: %d", keyLength))
	}
	//Both Passwords can be empty, but not a single one
	if (len(ownerPassword) == 0 || len(userPassword) == 0) && (len(ownerPassword)+len(userPassword) != 0) {
		panic("Can't have one single empty password for encryption")
	}
	settings := EncryptionOptions{KeyLength: keyLength, OwnerPassword: ownerPassword, UserPassword: userPassword}

	return &settings
}

func isValidKeyLength(keyLength int) bool {
	switch keyLength {
	case 40, 128, 256:
		return true
	default:
		return false
	}
}

func (e *EncryptionOptions) AreValidForEncryption() bool {
	return (len(e.OwnerPassword) != 0 && len(e.UserPassword) != 0)
}

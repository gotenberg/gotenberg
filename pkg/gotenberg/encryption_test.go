package gotenberg

import "testing"

func TestNewEncryptionOptions(t *testing.T) {
	for i, tc := range []struct {
		OwnerPassword         string
		UserPassword          string
		KeyLength             int
		expectPanic           bool
		AreValidForEncryption bool
	}{
		{
			OwnerPassword:         "a",
			UserPassword:          "a",
			KeyLength:             256,
			expectPanic:           false,
			AreValidForEncryption: true,
		},
		{
			OwnerPassword:         "a",
			UserPassword:          "a",
			KeyLength:             128,
			expectPanic:           false,
			AreValidForEncryption: true,
		},
		{
			OwnerPassword:         "a",
			UserPassword:          "a",
			KeyLength:             40,
			expectPanic:           false,
			AreValidForEncryption: true,
		},
		{
			OwnerPassword:         "a",
			UserPassword:          "a",
			KeyLength:             1337,
			expectPanic:           true,
			AreValidForEncryption: false,
		},
		{
			OwnerPassword:         "",
			UserPassword:          "",
			KeyLength:             256,
			expectPanic:           false,
			AreValidForEncryption: false,
		},
		{
			OwnerPassword:         "1",
			UserPassword:          "",
			KeyLength:             256,
			expectPanic:           true,
			AreValidForEncryption: false,
		},
		{
			OwnerPassword:         "",
			UserPassword:          "1",
			KeyLength:             256,
			expectPanic:           true,
			AreValidForEncryption: false,
		},
	} {
		func() {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("test %d: expected panic but got none", i)
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("test %d: expected no panic but got: %v", i, r)
					}
				}()
			}

			options := NewEncryptionOptions(tc.KeyLength, tc.OwnerPassword, tc.UserPassword)
			passValid := options.AreValidForEncryption()
			if tc.AreValidForEncryption != passValid {
				t.Errorf("test %d: expected encryptionoptions to be: %t, got: %t", i, tc.AreValidForEncryption, passValid)
			}
		}()
	}
}

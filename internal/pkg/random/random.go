package random

import (
	"github.com/labstack/gommon/random"
)

// Get returns a random string.
func Get() string {
	return random.String(32)
}

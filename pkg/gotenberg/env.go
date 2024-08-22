package gotenberg

import (
	"fmt"
	"os"
	"strconv"
)

// StringEnv retrieves the value of the environment variable named by the key.
// If the variable is present in the environment and not empty, the value is
// returned.
func StringEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("environment variable '%s' does not exist", key)
	}
	if val == "" {
		return "", fmt.Errorf("environment variable '%s' is empty", key)
	}
	return val, nil
}

// IntEnv relies on [StringEnv] and converts the values if it exists and is not
// empty.
func IntEnv(key string) (int, error) {
	val, err := StringEnv(key)
	if err != nil {
		return 0, err
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("get int value of environment variable '%s': %w", key, err)
	}
	return intVal, nil
}

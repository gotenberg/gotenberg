package gotenberg

import (
	"encoding/json"
	"fmt"
)

// ParseMetadata parses string-encoded JSON into a Go representation, i.e. map[string]interface{}.
// Should the string-encoded JSON be invalid an error will be returned.
func ParseMetadata(rawInput string) (map[string]interface{}, error) {
	parsed := make(map[string]interface{})
	if rawInput == "" {
		return parsed, nil
	}

	err := json.Unmarshal([]byte(rawInput), &parsed)
	if err != nil {
		return parsed, fmt.Errorf("metadata provided is invalid JSON and cannot be processed: %s", err)
	}

	return parsed, nil
}

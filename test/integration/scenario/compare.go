package scenario

import (
	"fmt"
	"reflect"
)

func compareJson(expected, actual interface{}) error {
	// Handle maps (JSON objects).
	expectedMap, ok := expected.(map[string]interface{})
	if ok {
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected an object, but actual is: %T", actual)
		}
		// For each key in expected, compare if the expected value is not
		// "ignore".
		for key, expVal := range expectedMap {
			if str, isStr := expVal.(string); isStr && str == "ignore" {
				continue // Skip.
			}
			actVal, exists := actualMap[key]
			if !exists {
				return fmt.Errorf("missing expected key %q", key)
			}
			if err := compareJson(expVal, actVal); err != nil {
				return fmt.Errorf("key %q: %w", key, err)
			}
		}
		return nil
	}

	// Handle slices (JSON arrays).
	expectedSlice, ok := expected.([]interface{})
	if ok {
		actualSlice, ok := actual.([]interface{})
		if !ok {
			return fmt.Errorf("expected an array, but actual is: %T", actual)
		}
		if len(expectedSlice) != len(actualSlice) {
			return fmt.Errorf("expected array length to be: %d, but actual is: %d", len(expectedSlice), len(actualSlice))
		}
		for i := range expectedSlice {
			if err := compareJson(expectedSlice[i], actualSlice[i]); err != nil {
				return fmt.Errorf("at index %d: %w", i, err)
			}
		}
		return nil
	}

	// For other types, compare directly.
	if !reflect.DeepEqual(expected, actual) {
		return fmt.Errorf("expected %v (%T) but got %v (%T)", expected, expected, actual, actual)
	}

	return nil
}

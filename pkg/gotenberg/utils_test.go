package gotenberg

import (
	"reflect"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	for i, tc := range []struct {
		input     string
		expect    map[string]interface{}
		expectErr bool
	}{
		{
			input:  ``,
			expect: make(map[string]interface{}),
		},
		{
			input:  `{}`,
			expect: make(map[string]interface{}),
		},
		{
			input:     `{`,
			expectErr: true,
			expect:    make(map[string]interface{}),
		},
		{
			input: `{ "foo": "foo" }`,
			expect: map[string]interface{}{
				"foo": "foo",
			},
		},
		{
			input: `{ "foo": "foo", "bar": "bar" }`,
			expect: map[string]interface{}{
				"foo": "foo",
				"bar": "bar",
			},
		},
		{
			input: `{ "foo": "foo", "bar": 123, "baz": 4.56, "qux": true, "quux": null }`,
			expect: map[string]interface{}{
				"foo":  "foo",
				"bar":  float64(123),
				"baz":  4.56,
				"qux":  true,
				"quux": nil,
			},
		},
		{
			input:     `{ "foo": "foo", "bar": 123, "baz": 4.56, "qux": true, "quux": null ] }`,
			expectErr: true,
			expect:    make(map[string]interface{}),
		},
		{
			input: `{ "foo": [ "bar", "baz", "qux" ] }`,
			expect: func() map[string]interface{} {
				input := []string{"bar", "baz", "qux"}
				elements := make([]interface{}, len(input))
				for i := range elements {
					elements[i] = input[i]
				}
				return map[string]interface{}{
					"foo": elements,
				}
			}(),
		},
		{
			input: `{ "foo": { "bar": "bar", "baz": "baz" } }`,
			expect: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "bar",
					"baz": "baz",
				},
			},
		},
	} {
		actual, err := ParseMetadata(tc.input)

		if !reflect.DeepEqual(actual, tc.expect) {
			t.Errorf("test %d: expected %+v but got: %+v", i, tc.expect, actual)
		}

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

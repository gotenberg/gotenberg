package api

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestFormData_Validate(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		expectError bool
	}{
		{
			scenario: "errors",
			form: &FormData{
				errors: errors.New("foo"),
			},
			expectError: true,
		},
		{
			scenario: "success",
			form: &FormData{
				errors: nil,
			},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			err := tc.form.Validate()

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

func TestFormData_String(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		form         *FormData
		defaultValue string
		expect       string
	}{
		{
			scenario:     "key does not exist, fallback to default zero value",
			form:         &FormData{},
			defaultValue: "",
			expect:       "",
		},
		{
			scenario:     "key does not exist, fallback to default value",
			form:         &FormData{},
			defaultValue: "foo",
			expect:       "foo",
		},
		{
			scenario: "key does exist, but empty value, fallback to default value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			defaultValue: "",
			expect:       "",
		},
		{
			scenario: "key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			defaultValue: "",
			expect:       "foo",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual string

			tc.form.String("foo", &actual, tc.defaultValue)

			if actual != tc.expect {
				t.Errorf("expected '%s' but got '%s'", tc.expect, actual)
			}

			if tc.form.errors != nil {
				t.Errorf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_MandatoryString(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		expect      string
		expectError bool
	}{
		{
			scenario:    "missing mandatory key",
			form:        &FormData{},
			expect:      "",
			expectError: true,
		},
		{
			scenario: "mandatory value is empty",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expect:      "",
			expectError: true,
		},
		{
			scenario: "mandatory key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expect:      "foo",
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual string

			tc.form.MandatoryString("foo", &actual)

			if actual != tc.expect {
				t.Errorf("expected '%s' but got '%s'", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_Bool(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		form         *FormData
		defaultValue bool
		expect       bool
		expectError  bool
	}{
		{
			scenario:     "key does not exist, fallback to default zero value",
			form:         &FormData{},
			defaultValue: false,
			expect:       false,
			expectError:  false,
		},
		{
			scenario:     "key does not exist, fallback to default value",
			form:         &FormData{},
			defaultValue: true,
			expect:       true,
			expectError:  false,
		},
		{
			scenario: "key does exist, but empty value, fallback to default value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			defaultValue: false,
			expect:       false,
			expectError:  false,
		},
		{
			scenario: "key does exist, but value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			defaultValue: false,
			expect:       false,
			expectError:  true,
		},
		{
			scenario: "key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"true",
					},
				},
			},
			defaultValue: false,
			expect:       true,
			expectError:  false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual bool

			tc.form.Bool("foo", &actual, tc.defaultValue)

			if actual != tc.expect {
				t.Errorf("expected %t but got %t", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_MandatoryBool(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		expect      bool
		expectError bool
	}{
		{
			scenario:    "missing mandatory key",
			form:        &FormData{},
			expect:      false,
			expectError: true,
		},
		{
			scenario: "mandatory value is empty",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expect:      false,
			expectError: true,
		},
		{
			scenario: "mandatory value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expect:      false,
			expectError: true,
		},
		{
			scenario: "mandatory key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"true",
					},
				},
			},
			expect:      true,
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual bool

			tc.form.MandatoryBool("foo", &actual)

			if actual != tc.expect {
				t.Errorf("expected %t but got %t", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_Int(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		form         *FormData
		defaultValue int
		expect       int
		expectError  bool
	}{
		{
			scenario:     "key does not exist, fallback to default zero value",
			form:         &FormData{},
			defaultValue: 0,
			expect:       0,
			expectError:  false,
		},
		{
			scenario:     "key does not exist, fallback to default value",
			form:         &FormData{},
			defaultValue: 2,
			expect:       2,
			expectError:  false,
		},
		{
			scenario: "key does exist, but empty value, fallback to default value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			defaultValue: 0,
			expect:       0,
			expectError:  false,
		},
		{
			scenario: "key does exist, but value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			defaultValue: 0,
			expect:       0,
			expectError:  true,
		},
		{
			scenario: "key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"3",
					},
				},
			},
			defaultValue: 0,
			expect:       3,
			expectError:  false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual int

			tc.form.Int("foo", &actual, tc.defaultValue)

			if actual != tc.expect {
				t.Errorf("expected %d but got %d", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_MandatoryInt(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		expect      int
		expectError bool
	}{
		{
			scenario:    "missing mandatory key",
			form:        &FormData{},
			expect:      0,
			expectError: true,
		},
		{
			scenario: "mandatory value is empty",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expect:      0,
			expectError: true,
		},
		{
			scenario: "mandatory value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expect:      0,
			expectError: true,
		},
		{
			scenario: "mandatory key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"2",
					},
				},
			},
			expect:      2,
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual int

			tc.form.MandatoryInt("foo", &actual)

			if actual != tc.expect {
				t.Errorf("expected %d but got %d", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_Float64(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		form         *FormData
		defaultValue float64
		expect       float64
		expectError  bool
	}{
		{
			scenario:     "key does not exist, fallback to default zero value",
			form:         &FormData{},
			defaultValue: 0.0,
			expect:       0.0,
			expectError:  false,
		},
		{
			scenario:     "key does not exist, fallback to default value",
			form:         &FormData{},
			defaultValue: 2.5,
			expect:       2.5,
			expectError:  false,
		},
		{
			scenario: "key does exist, but empty value, fallback to default value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			defaultValue: 0.0,
			expect:       0.0,
			expectError:  false,
		},
		{
			scenario: "key does exist, but value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			defaultValue: 0.0,
			expect:       0.0,
			expectError:  true,
		},
		{
			scenario: "key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"3.5",
					},
				},
			},
			defaultValue: 0.0,
			expect:       3.5,
			expectError:  false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual float64

			tc.form.Float64("foo", &actual, tc.defaultValue)

			if actual != tc.expect {
				t.Errorf("expected %f but got %f", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_MandatoryFloat64(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		expect      float64
		expectError bool
	}{
		{
			scenario:    "missing mandatory key",
			form:        &FormData{},
			expect:      0.0,
			expectError: true,
		},
		{
			scenario: "mandatory value is empty",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expect:      0.0,
			expectError: true,
		},
		{
			scenario: "mandatory value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expect:      0.0,
			expectError: true,
		},
		{
			scenario: "mandatory key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"2.5",
					},
				},
			},
			expect:      2.5,
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual float64

			tc.form.MandatoryFloat64("foo", &actual)

			if actual != tc.expect {
				t.Errorf("expected %f but got %f", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_Duration(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		form         *FormData
		defaultValue time.Duration
		expect       time.Duration
		expectError  bool
	}{
		{
			scenario:     "key does not exist, fallback to default zero value",
			form:         &FormData{},
			defaultValue: time.Duration(0),
			expect:       time.Duration(0),
			expectError:  false,
		},
		{
			scenario:     "key does not exist, fallback to default value",
			form:         &FormData{},
			defaultValue: time.Duration(1) * time.Second,
			expect:       time.Duration(1) * time.Second,
		},
		{
			scenario: "key does exist, but empty value, fallback to default value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			defaultValue: time.Duration(0),
			expect:       time.Duration(0),
			expectError:  false,
		},
		{
			scenario: "key does exist, but value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			defaultValue: time.Duration(0),
			expect:       time.Duration(0),
			expectError:  true,
		},
		{
			scenario: "key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"1s",
					},
				},
			},
			defaultValue: time.Duration(0),
			expect:       time.Duration(1) * time.Second,
			expectError:  false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual time.Duration

			tc.form.Duration("foo", &actual, tc.defaultValue)

			if actual != tc.expect {
				t.Errorf("expected '%s' but got '%s'", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_MandatoryDuration(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		expect      time.Duration
		expectError bool
	}{
		{
			scenario:    "missing mandatory key",
			form:        &FormData{},
			expect:      time.Duration(0),
			expectError: true,
		},
		{
			scenario: "mandatory value is empty",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expect:      time.Duration(0),
			expectError: true,
		},
		{
			scenario: "mandatory value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expect:      time.Duration(0),
			expectError: true,
		},
		{
			scenario: "mandatory key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"1s",
					},
				},
			},
			expect:      time.Duration(1) * time.Second,
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual time.Duration

			tc.form.MandatoryDuration("foo", &actual)

			if actual != tc.expect {
				t.Errorf("expected '%s' but got '%s'", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_Custom(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		form         *FormData
		defaultValue map[string]string
		expect       map[string]string
		expectError  bool
	}{
		{
			scenario:     "key does not exist, fallback to default zero value",
			form:         &FormData{},
			defaultValue: nil,
			expect:       nil,
			expectError:  false,
		},
		{
			scenario:     "key does not exist, fallback to default value",
			form:         &FormData{},
			defaultValue: map[string]string{"foo": "foo"},
			expect:       map[string]string{"foo": "foo"},
			expectError:  false,
		},
		{
			scenario: "key does exist, but empty value, fallback to default value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			defaultValue: nil,
			expect:       nil,
			expectError:  false,
		},
		{
			scenario: "key does exist, but value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			defaultValue: nil,
			expect:       nil,
			expectError:  true,
		},
		{
			scenario: "key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						`{ "foo": "foo" }`,
					},
				},
			},
			defaultValue: nil,
			expect:       map[string]string{"foo": "foo"},
			expectError:  false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual map[string]string

			tc.form.Custom("foo", func(value string) error {
				if value == "" {
					actual = tc.defaultValue

					return nil
				}

				err := json.Unmarshal([]byte(value), &actual)
				if err != nil {
					return err
				}

				return nil
			})

			if !reflect.DeepEqual(actual, tc.expect) {
				t.Errorf("expected %+v but got: %+v", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_MandatoryCustom(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		expect      map[string]string
		expectError bool
	}{
		{
			scenario:    "missing mandatory key",
			form:        &FormData{},
			expect:      nil,
			expectError: true,
		},
		{
			scenario: "mandatory value is empty",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expect:      nil,
			expectError: true,
		},
		{
			scenario: "mandatory key does exist, but value is invalid",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expect:      nil,
			expectError: true,
		},
		{
			scenario: "mandatory key does exist with a value",
			form: &FormData{
				values: map[string][]string{
					"foo": {
						`{ "foo": "foo" }`,
					},
				},
			},
			expect:      map[string]string{"foo": "foo"},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual map[string]string

			tc.form.MandatoryCustom("foo", func(value string) error {
				err := json.Unmarshal([]byte(value), &actual)
				if err != nil {
					return err
				}

				return nil
			})

			if !reflect.DeepEqual(actual, tc.expect) {
				t.Errorf("expected %+v but got: %+v", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_Path(t *testing.T) {
	for _, tc := range []struct {
		scenario string
		form     *FormData
		expect   string
	}{
		{
			scenario: "no file, fallback to zero value",
			form:     &FormData{},
			expect:   "",
		},
		{
			scenario: "file does not exist, fallback to zero value",
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
			expect: "",
		},
		{
			scenario: "file does exist",
			form: &FormData{
				files: map[string]string{
					"foo": "/foo",
				},
			},
			expect: "/foo",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual string

			tc.form.Path("foo", &actual)

			if actual != tc.expect {
				t.Errorf("expected '%s' but got '%s'", tc.expect, actual)
			}

			if tc.form.errors != nil {
				t.Errorf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_MandatoryPath(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		expect      string
		expectError bool
	}{
		{
			scenario:    "missing mandatory file: no file",
			form:        &FormData{},
			expect:      "",
			expectError: true,
		},
		{
			scenario: "missing mandatory file: file does not exist",
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
			expect:      "",
			expectError: true,
		},
		{
			scenario: "mandatory file does exist",
			form: &FormData{
				files: map[string]string{
					"foo": "/foo",
				},
			},
			expect:      "/foo",
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual string

			tc.form.MandatoryPath("foo", &actual)

			if actual != tc.expect {
				t.Errorf("expected '%s' but got '%s'", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_Content(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		form         *FormData
		filename     string
		defaultValue string
		expect       string
		expectError  bool
	}{
		{
			scenario:     "no file, fallback to zero value",
			form:         &FormData{},
			filename:     "",
			defaultValue: "",
			expect:       "",
			expectError:  false,
		},
		{
			scenario: "file does not exist, fallback to zero value",
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
			filename:     "foo",
			defaultValue: "",
			expect:       "",
			expectError:  false,
		},
		{
			scenario: "file does not exist, fallback to default value",
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
			filename:     "foo",
			defaultValue: "foo",
			expect:       "foo",
			expectError:  false,
		},
		{
			scenario: "file does not exist, cannot read its content",
			form: &FormData{
				files: map[string]string{
					"foo": "/foo",
				},
			},
			filename:     "foo",
			defaultValue: "",
			expect:       "",
			expectError:  true,
		},
		{
			scenario: "file does exist without file extension",
			form: &FormData{
				files: map[string]string{
					"foo": "/tests/test/testdata/api/sample1.txt",
				},
			},
			filename:     "foo",
			defaultValue: "",
			expect:       "foo",
			expectError:  false,
		},
		{
			scenario: "file does exist with an uppercase file extension",
			form: &FormData{
				files: map[string]string{
					"foo.TXT": "/tests/test/testdata/api/sample1.txt",
				},
			},
			filename:     "foo.txt",
			defaultValue: "",
			expect:       "foo",
			expectError:  false,
		},
		{
			scenario: "file does exist without a lowercase file extension",
			form: &FormData{
				files: map[string]string{
					"foo.txt": "/tests/test/testdata/api/sample1.txt",
				},
			},
			filename:     "foo.txt",
			defaultValue: "",
			expect:       "foo",
			expectError:  false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual string

			tc.form.Content(tc.filename, &actual, tc.defaultValue)

			if actual != tc.expect {
				t.Errorf("expected '%s' but got '%s'", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_MandatoryContent(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		filename    string
		expect      string
		expectError bool
	}{
		{
			scenario:    "missing mandatory file: no file",
			form:        &FormData{},
			filename:    "foo",
			expect:      "",
			expectError: true,
		},
		{
			scenario: "missing mandatory file: file does not exist",
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
			filename:    "foo",
			expect:      "",
			expectError: true,
		},
		{
			scenario: "mandatory file does exist, cannot read its content",
			form: &FormData{
				files: map[string]string{
					"foo": "/foo",
				},
			},
			filename:    "foo",
			expect:      "",
			expectError: true,
		},
		{
			scenario: "mandatory file does exist without file extension",
			form: &FormData{
				files: map[string]string{
					"foo": "/tests/test/testdata/api/sample1.txt",
				},
			},
			filename:    "foo",
			expect:      "foo",
			expectError: false,
		},
		{
			scenario: "mandatory file does exist with an uppercase file extension",
			form: &FormData{
				files: map[string]string{
					"foo.TXT": "/tests/test/testdata/api/sample1.txt",
				},
			},
			filename:    "foo.txt",
			expect:      "foo",
			expectError: false,
		},
		{
			scenario: "mandatory file does exist without a lowercase file extension",
			form: &FormData{
				files: map[string]string{
					"foo.txt": "/tests/test/testdata/api/sample1.txt",
				},
			},
			filename:    "foo.txt",
			expect:      "foo",
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual string

			tc.form.MandatoryContent(tc.filename, &actual)

			if actual != tc.expect {
				t.Errorf("expected '%s' but got '%s'", tc.expect, actual)
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_Paths(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		extensions  []string
		expect      []string
		expectCount int
	}{
		{
			scenario:    "no file, fallback to zero value",
			form:        &FormData{},
			extensions:  nil,
			expect:      nil,
			expectCount: 0,
		},
		{
			scenario: "no file with given file extension, fallback to zero value",
			form: &FormData{
				files: map[string]string{
					"foo.zip": "/foo.zip",
					"foo.pdf": "/foo.pdf",
				},
			},
			extensions:  []string{".txt"},
			expect:      nil,
			expectCount: 0,
		},
		{
			scenario: "files do exist with given file extension",
			form: &FormData{
				files: map[string]string{
					"foo.zip": "/foo.zip",
					"b.pdf":   "/b.PDF",
					"a.pdf":   "/a.pdf",
				},
			},
			extensions: []string{".pdf"},
			expect: []string{
				"/a.pdf",
				"/b.PDF",
			},
			expectCount: 2,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual []string

			tc.form.Paths(tc.extensions, &actual)

			if !reflect.DeepEqual(actual, tc.expect) {
				t.Errorf("expected %v but got: %v", tc.expect, actual)
			}

			if len(actual) != tc.expectCount {
				t.Errorf("expected %d files but got %d", tc.expectCount, len(actual))
			}

			if tc.form.errors != nil {
				t.Errorf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_MandatoryPaths(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		form        *FormData
		extensions  []string
		expect      []string
		expectCount int
		expectError bool
	}{
		{
			scenario:    "missing mandatory files: no file",
			form:        &FormData{},
			extensions:  nil,
			expect:      nil,
			expectCount: 0,
			expectError: true,
		},
		{
			scenario: "missing mandatory files: no file with given file extension",
			form: &FormData{
				files: map[string]string{
					"foo.zip": "/foo.zip",
					"foo.pdf": "/foo.pdf",
				},
			},
			extensions:  []string{".txt"},
			expect:      nil,
			expectCount: 0,
			expectError: true,
		},
		{
			scenario: "mandatory files do exist with given file extension",
			form: &FormData{
				files: map[string]string{
					"foo.zip": "/foo.zip",
					"b.PDF":   "/b.PDF",
					"a.pdf":   "/a.pdf",
				},
			},
			extensions: []string{".pdf"},
			expect: []string{
				"/a.pdf",
				"/b.PDF",
			},
			expectCount: 2,
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var actual []string

			tc.form.MandatoryPaths(tc.extensions, &actual)

			if !reflect.DeepEqual(actual, tc.expect) {
				t.Errorf("expected %v but got: %v", tc.expect, actual)
			}

			if len(actual) != tc.expectCount {
				t.Errorf("expected %d files but got %d", tc.expectCount, len(actual))
			}

			if tc.expectError && tc.form.errors == nil {
				t.Fatal("expected error but got none", tc.form.errors)
			}

			if !tc.expectError && tc.form.errors != nil {
				t.Fatalf("expected no error but got: %v", tc.form.errors)
			}
		})
	}
}

func TestFormData_append(t *testing.T) {
	form := &FormData{}
	form.append(errors.New("foo"))

	if form.errors == nil {
		t.Errorf("expected error but got: %v", form.errors)
	}
}

func TestFormData_mustValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic but got none")
		}
	}()

	form := &FormData{}
	defaultValue := []string{"foo"}

	var target []string
	form.mustValue("foo", &target, defaultValue)
}

func TestFormData_mustAssign(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic but got none")
		}
	}()

	form := &FormData{}

	var target []string
	form.mustAssign("foo", "foo", &target)
}

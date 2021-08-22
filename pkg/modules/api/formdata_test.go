package api

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestFormData_Validate(t *testing.T) {
	for i, tc := range []struct {
		form      FormData
		expectErr bool
	}{
		{
			form: FormData{
				errors: errors.New("foo"),
			},
			expectErr: true,
		},
		{
			form: FormData{
				errors: nil,
			},
		},
	} {
		err := tc.form.Validate()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestFormData_String(t *testing.T) {
	for i, tc := range []struct {
		form         *FormData
		defaultValue string
		expect       string
	}{
		{
			form: &FormData{},
		},
		{
			form:         &FormData{},
			defaultValue: "foo",
			expect:       "foo",
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expect: "foo",
		},
	} {
		var actual string

		tc.form.String("foo", &actual, tc.defaultValue)

		if actual != tc.expect {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expect, actual)
		}

		if tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_MandatoryString(t *testing.T) {
	for i, tc := range []struct {
		form      *FormData
		expect    string
		expectErr bool
	}{
		{
			form:      &FormData{},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expect: "foo",
		},
	} {
		var actual string

		tc.form.MandatoryString("foo", &actual)

		if actual != tc.expect {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_Bool(t *testing.T) {
	for i, tc := range []struct {
		form         *FormData
		defaultValue bool
		expect       bool
		expectErr    bool
	}{
		{
			form: &FormData{},
		},
		{
			form:         &FormData{},
			defaultValue: true,
			expect:       true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"true",
					},
				},
			},
			expect: true,
		},
	} {
		var actual bool

		tc.form.Bool("foo", &actual, tc.defaultValue)

		if actual != tc.expect {
			t.Errorf("test %d: expected %t but got %t", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_MandatoryBool(t *testing.T) {
	for i, tc := range []struct {
		form      *FormData
		expect    bool
		expectErr bool
	}{
		{
			form:      &FormData{},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"true",
					},
				},
			},
			expect: true,
		},
	} {
		var actual bool

		tc.form.MandatoryBool("foo", &actual)

		if actual != tc.expect {
			t.Errorf("test %d: expected %t but got %t", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_Int(t *testing.T) {
	for i, tc := range []struct {
		form         *FormData
		defaultValue int
		expect       int
		expectErr    bool
	}{
		{
			form: &FormData{},
		},
		{
			form:         &FormData{},
			defaultValue: 2,
			expect:       2,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"3",
					},
				},
			},
			expect: 3,
		},
	} {
		var actual int

		tc.form.Int("foo", &actual, tc.defaultValue)

		if actual != tc.expect {
			t.Errorf("test %d: expected %d but got %d", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_MandatoryInt(t *testing.T) {
	for i, tc := range []struct {
		form      *FormData
		expect    int
		expectErr bool
	}{
		{
			form:      &FormData{},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"2",
					},
				},
			},
			expect: 2,
		},
	} {
		var actual int

		tc.form.MandatoryInt("foo", &actual)

		if actual != tc.expect {
			t.Errorf("test %d: expected %d but got %d", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_Float64(t *testing.T) {
	for i, tc := range []struct {
		form         *FormData
		defaultValue float64
		expect       float64
		expectErr    bool
	}{
		{
			form: &FormData{},
		},
		{
			form:         &FormData{},
			defaultValue: 2.5,
			expect:       2.5,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"3.5",
					},
				},
			},
			expect: 3.5,
		},
	} {
		var actual float64

		tc.form.Float64("foo", &actual, tc.defaultValue)

		if actual != tc.expect {
			t.Errorf("test %d: expected %f but got %f", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_MandatoryFloat64(t *testing.T) {
	for i, tc := range []struct {
		form      *FormData
		expect    float64
		expectErr bool
	}{
		{
			form:      &FormData{},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"2.5",
					},
				},
			},
			expect: 2.5,
		},
	} {
		var actual float64

		tc.form.MandatoryFloat64("foo", &actual)

		if actual != tc.expect {
			t.Errorf("test %d: expected %f but got %f", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_Duration(t *testing.T) {
	for i, tc := range []struct {
		form         *FormData
		defaultValue time.Duration
		expect       time.Duration
		expectErr    bool
	}{
		{
			form: &FormData{},
		},
		{
			form:         &FormData{},
			defaultValue: time.Duration(1) * time.Second,
			expect:       time.Duration(1) * time.Second,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"1s",
					},
				},
			},
			expect: time.Duration(1) * time.Second,
		},
	} {
		var actual time.Duration

		tc.form.Duration("foo", &actual, tc.defaultValue)

		if actual != tc.expect {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_MandatoryDuration(t *testing.T) {
	for i, tc := range []struct {
		form      *FormData
		expect    time.Duration
		expectErr bool
	}{
		{
			form:      &FormData{},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"1s",
					},
				},
			},
			expect: time.Duration(1) * time.Second,
		},
	} {
		var actual time.Duration

		tc.form.MandatoryDuration("foo", &actual)

		if actual != tc.expect {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_Custom(t *testing.T) {
	for i, tc := range []struct {
		form         *FormData
		defaultValue map[string]string
		expect       map[string]string
		expectErr    bool
	}{
		{
			form: &FormData{},
		},
		{
			form:         &FormData{},
			defaultValue: map[string]string{"foo": "foo"},
			expect:       map[string]string{"foo": "foo"},
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						`{ "foo": "foo" }`,
					},
				},
			},
			expect: map[string]string{"foo": "foo"},
		},
	} {
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
			t.Errorf("test %d: expected %+v but got: %+v", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_MandatoryCustom(t *testing.T) {
	for i, tc := range []struct {
		form      *FormData
		expect    map[string]string
		expectErr bool
	}{
		{
			form:      &FormData{},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						"foo",
					},
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				values: map[string][]string{
					"foo": {
						`{ "foo": "foo" }`,
					},
				},
			},
			expect: map[string]string{"foo": "foo"},
		},
	} {
		var actual map[string]string

		tc.form.MandatoryCustom("foo", func(value string) error {
			err := json.Unmarshal([]byte(value), &actual)
			if err != nil {
				return err
			}

			return nil
		})

		if !reflect.DeepEqual(actual, tc.expect) {
			t.Errorf("test %d: expected %+v but got: %+v", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_Path(t *testing.T) {
	for i, tc := range []struct {
		form   *FormData
		expect string
	}{
		{
			form: &FormData{},
		},
		{
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo": "/foo",
				},
			},
			expect: "/foo",
		},
	} {
		var actual string

		tc.form.Path("foo", &actual)

		if actual != tc.expect {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expect, actual)
		}

		if tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_MandatoryPath(t *testing.T) {
	for i, tc := range []struct {
		form      *FormData
		expect    string
		expectErr bool
	}{
		{
			form:      &FormData{},
			expectErr: true,
		},
		{
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo": "/foo",
				},
			},
			expect: "/foo",
		},
	} {
		var actual string

		tc.form.MandatoryPath("foo", &actual)

		if actual != tc.expect {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_Content(t *testing.T) {
	for i, tc := range []struct {
		form         *FormData
		defaultValue string
		expect       string
		expectErr    bool
	}{
		{
			form: &FormData{},
		},
		{
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
		},
		{
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
			defaultValue: "foo",
			expect:       "foo",
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo": "/foo",
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo": "/tests/test/testdata/api/sample1.txt",
				},
			},
			expect: "foo",
		},
	} {
		var actual string

		tc.form.Content("foo", &actual, tc.defaultValue)

		if actual != tc.expect {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_MandatoryContent(t *testing.T) {
	for i, tc := range []struct {
		form      *FormData
		expect    string
		expectErr bool
	}{
		{
			form:      &FormData{},
			expectErr: true,
		},
		{
			form: &FormData{
				files: map[string]string{
					"bar": "/bar",
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo": "/foo",
				},
			},
			expectErr: true,
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo": "/tests/test/testdata/api/sample1.txt",
				},
			},
			expect: "foo",
		},
	} {
		var actual string

		tc.form.MandatoryContent("foo", &actual)

		if actual != tc.expect {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expect, actual)
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_Paths(t *testing.T) {
	for i, tc := range []struct {
		form        *FormData
		extensions  []string
		expect      []string
		expectCount int
	}{
		{
			form: &FormData{},
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo.zip": "/foo.zip",
					"foo.pdf": "/foo.pdf",
				},
			},
			extensions: []string{".txt"},
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo.zip": "/foo.zip",
					"b.pdf":   "/b.pdf",
					"a.pdf":   "/a.pdf",
				},
			},
			extensions: []string{".pdf"},
			expect: []string{
				"/a.pdf",
				"/b.pdf",
			},
			expectCount: 2,
		},
	} {
		var actual []string

		tc.form.Paths(tc.extensions, &actual)

		if !reflect.DeepEqual(actual, tc.expect) {
			t.Errorf("test %d: expected %v but got: %v", i, tc.expect, actual)
		}

		if len(actual) != tc.expectCount {
			t.Errorf("test %d: expected %d files but got %d", i, tc.expectCount, len(actual))
		}

		if tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
	}
}

func TestFormData_MandatoryPaths(t *testing.T) {
	for i, tc := range []struct {
		form        *FormData
		extensions  []string
		expect      []string
		expectCount int
		expectErr   bool
	}{
		{
			form:      &FormData{},
			expectErr: true,
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo.zip": "/foo.zip",
					"foo.pdf": "/foo.pdf",
				},
			},
			extensions: []string{".txt"},
			expectErr:  true,
		},
		{
			form: &FormData{
				files: map[string]string{
					"foo.zip": "/foo.zip",
					"b.pdf":   "/b.pdf",
					"a.pdf":   "/a.pdf",
				},
			},
			extensions: []string{".pdf"},
			expect: []string{
				"/a.pdf",
				"/b.pdf",
			},
			expectCount: 2,
		},
	} {
		var actual []string

		tc.form.MandatoryPaths(tc.extensions, &actual)

		if !reflect.DeepEqual(actual, tc.expect) {
			t.Errorf("test %d: expected %v but got: %v", i, tc.expect, actual)
		}

		if len(actual) != tc.expectCount {
			t.Errorf("test %d: expected %d files but got %d", i, tc.expectCount, len(actual))
		}

		if tc.expectErr && tc.form.errors == nil {
			t.Errorf("test %d: expected error but got: %v", i, tc.form.errors)
		}

		if !tc.expectErr && tc.form.errors != nil {
			t.Errorf("test %d: expected no error but got: %v", i, tc.form.errors)
		}
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

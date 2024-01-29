package gotenberg

import (
	"reflect"
	"testing"
)

func TestMustRegisterModule(t *testing.T) {
	descriptorsMu.RLock()
	descriptors = map[string]ModuleDescriptor{
		"a": {ID: "a"},
	}
	descriptorsMu.RUnlock()

	for _, tc := range []struct {
		scenario    string
		ID          string
		New         func() Module
		expectPanic bool
	}{
		{
			scenario:    "no ID",
			ID:          "",
			New:         func() Module { return new(ModuleMock) },
			expectPanic: true,
		},
		{
			scenario:    "nil New method",
			ID:          "b",
			New:         nil,
			expectPanic: true,
		},
		{
			scenario:    "nil module",
			ID:          "b",
			New:         func() Module { return nil },
			expectPanic: true,
		},
		{
			scenario:    "existing module",
			ID:          "a",
			New:         func() Module { return new(ModuleMock) },
			expectPanic: true,
		},
		{
			scenario:    "success",
			ID:          "b",
			New:         func() Module { return new(ModuleMock) },
			expectPanic: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := &struct{ ModuleMock }{}
			mod.DescriptorMock = func() ModuleDescriptor { return ModuleDescriptor{ID: tc.ID, New: tc.New} }

			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("expected no panic but got: %v", r)
					}
				}()
			}

			MustRegisterModule(mod)
		})
	}

	descriptorsMu.RLock()
	descriptors = make(map[string]ModuleDescriptor)
	descriptorsMu.RUnlock()
}

func TestGetModuleDescriptors(t *testing.T) {
	descriptorsMu.RLock()
	descriptors = map[string]ModuleDescriptor{
		"d": {ID: "d"},
		"c": {ID: "c"},
		"b": {ID: "b"},
		"a": {ID: "a"},
	}
	descriptorsMu.RUnlock()

	expect := []ModuleDescriptor{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
		{ID: "d"},
	}

	actual := GetModuleDescriptors()

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("expected %v but got %v", expect, actual)
	}

	descriptorsMu.RLock()
	descriptors = make(map[string]ModuleDescriptor)
	descriptorsMu.RUnlock()
}

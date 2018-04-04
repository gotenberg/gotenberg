package app

import (
	"path/filepath"
	"testing"
)

func TestNewApp(t *testing.T) {
	// case 1: uses an empty configuration file path.
	if _, err := NewApp("tests", ""); err == nil {
		t.Error("App should not have been instantiated!")
	}

	// case 2: uses an correct configuration file.
	path, _ := filepath.Abs("../_tests/configurations/gotenberg.yml")
	if _, err := NewApp("tests", path); err != nil {
		t.Error("App should have been instantiated!")
	}
}

func TestRun(t *testing.T) {
	path, _ := filepath.Abs("../_tests/configurations/gotenberg.yml")
	a, _ := NewApp("tests", path)

	quit := make(chan bool, 1)
	go func() {
		a.Run()
	}()
	quit <- true
}

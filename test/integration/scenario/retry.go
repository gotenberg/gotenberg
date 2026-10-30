package scenario

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/cucumber/godog"
)

var (
	failedScenariosMu sync.Mutex
	failedScenarios   []string // file:line paths for re-run
	// unmappedFailures counts the failed scenarios no file:line path could be
	// found for. A retry would skip them, so the run must fail instead.
	unmappedFailures int
)

// ResetFailedScenarios clears the collected failures.
func ResetFailedScenarios() {
	failedScenariosMu.Lock()
	defer failedScenariosMu.Unlock()
	failedScenarios = nil
	unmappedFailures = 0
}

// UnmappedFailures returns the number of failed scenarios a retry cannot
// target.
func UnmappedFailures() int {
	failedScenariosMu.Lock()
	defer failedScenariosMu.Unlock()
	return unmappedFailures
}

// FailedScenarioPaths returns file:line paths of failed scenarios.
func FailedScenarioPaths() []string {
	failedScenariosMu.Lock()
	defer failedScenariosMu.Unlock()
	result := make([]string, len(failedScenarios))
	copy(result, failedScenarios)
	return result
}

func recordFailedScenario(sc *godog.Scenario) {
	// When godog runs with file:line paths, the Uri may contain
	// the line suffix (e.g., "features/root.feature:4"). Strip it
	// to get the actual file path for reading.
	filePath := sc.Uri
	if idx := strings.LastIndex(filePath, ":"); idx > 0 {
		if _, err := strconv.Atoi(filePath[idx+1:]); err == nil {
			filePath = filePath[:idx]
		}
	}

	line := findScenarioLine(filePath, sc.Name)
	if line <= 0 {
		failedScenariosMu.Lock()
		defer failedScenariosMu.Unlock()
		unmappedFailures++
		return
	}

	path := fmt.Sprintf("%s:%d", filePath, line)

	failedScenariosMu.Lock()
	defer failedScenariosMu.Unlock()
	failedScenarios = append(failedScenarios, path)
}

func findScenarioLine(filePath, name string) int {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}

	// A retry of a Scenario Outline line runs all of its examples.
	targets := []string{"Scenario: " + name, "Scenario Outline: " + name}
	for i, line := range strings.Split(string(data), "\n") {
		if slices.Contains(targets, strings.TrimSpace(line)) {
			return i + 1
		}
	}

	return 0
}

package notify

import (
	"fmt"
	"os"

	"github.com/labstack/gommon/color"
)

// Print prints a message to stdout.
func Print(message string) {
	stdout := color.New()
	stdout.SetOutput(os.Stdout)
	stdout.Printf("⇨ %s\n", message)
}

// Printf prints a formatted message to stdout.
func Printf(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	Print(message)
}

// WarnPrint prints a warning to stderr.
func WarnPrint(err error) {
	stderr := color.New()
	stderr.SetOutput(os.Stderr)
	stderr.Printf("%s\n", color.Yellow(fmt.Sprintf("⇨ warn: %v", err)))
}

// ErrPrint prints an error to stderr.
func ErrPrint(err error) {
	stderr := color.New()
	stderr.SetOutput(os.Stderr)
	stderr.Printf("%s\n", color.Red(fmt.Sprintf("⇨ error: %v", err)))
}

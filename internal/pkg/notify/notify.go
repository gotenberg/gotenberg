package notify

import (
	"fmt"
	"os"

	"github.com/labstack/gommon/color"
)

var (
	stdout *color.Color
	stderr *color.Color
)

func init() {
	stdout = color.New()
	stdout.SetOutput(os.Stdout)
	stderr = color.New()
	stderr.SetOutput(os.Stderr)
}

// Println prints a message to stdout.
func Println(message string) {
	stdout.Printf("⇨ %s\n", message)
}

// ErrPrintln prints an error to stderr.
func ErrPrintln(err error) {
	stderr.Printf("%s\n", color.Red(fmt.Sprintf("⇨ error: %v", err)))
}

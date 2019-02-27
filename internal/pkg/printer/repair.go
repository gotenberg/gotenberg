package printer

import (
	"context"
	"os/exec"
)

// Repair a PDF’s Corrupted XREF Table and Stream Lengths (If Possible):
type Repair struct {
	Context   context.Context
	FilePaths []string
}

func (r *Repair) Print(destination string) error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, r.FilePaths[0])
	cmdArgs = append(cmdArgs, "output", destination)
	cmd := exec.Command("pdftk", cmdArgs...)
	return cmd.Run()
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(Repair))
)

//go:build linux

package chromium

import (
	"os"
	"os/exec"
	"syscall"
)

// setParentDeathSignal makes the kernel kill Chromium when Gotenberg dies, as
// chromedp's exec allocator does. AWS Lambda does not support it.
func setParentDeathSignal(cmd *exec.Cmd) {
	if _, ok := os.LookupEnv("LAMBDA_TASK_ROOT"); ok {
		return
	}

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = new(syscall.SysProcAttr)
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}

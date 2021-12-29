package gotenberg

import (
	"fmt"
	"os/exec"
	"syscall"
)

// Stop stops the command by sending its process group a SIGTERM signal.
// Stop is idempotent. An error should only be returned in the rare case that
// Stop is called immediately after the command ends but before Start can
// update its internal state.
func terminateProcess(pid int) error {
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", fmt.Sprintf("%d", pid))
	return kill.Run()
}

func setProcessGroupID(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{}
}

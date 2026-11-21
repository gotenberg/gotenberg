//go:build !linux

package chromium

import "os/exec"

// setParentDeathSignal does nothing: only Linux has a parent death signal.
func setParentDeathSignal(_ *exec.Cmd) {}

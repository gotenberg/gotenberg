//go:build linux

package gotenberg

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// landlockRulesetAttr is struct landlock_ruleset_attr cut to its first
// field, the only one Landlock ABI 1 knows. Every kernel with Landlock
// accepts it.
type landlockRulesetAttr struct {
	handledAccessFs uint64
}

// RestrictExec forbids the process, and every process it starts afterward,
// to execute any file but paths and the files beneath them. It relies on
// Landlock (Linux 5.13 and later), and cannot be undone. It leaves reads and
// writes alone.
//
// Landlock checks execve, not executable memory mappings, so two paths stay
// open to code that already runs in a restricted process: the allowed
// program interpreter loading a file ("ld.so ./binary"), which a noexec
// mount of every writable directory stops, and executing an anonymous
// memfd, which the vm.memfd_noexec=2 sysctl stops (Linux 6.3 and later).
//
// Call it before the process starts children, with the result of
// [ExecRestrictionPaths]. It fails with [ErrExecRestrictionUnavailable] when
// the kernel lacks Landlock, a seccomp profile denies its system calls, or
// the binary uses cgo.
func RestrictExec(paths []string) error {
	_, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if errno != 0 {
		return fmt.Errorf("%w: probe Landlock: %s", ErrExecRestrictionUnavailable, landlockErrnoReason(errno))
	}

	attr := landlockRulesetAttr{handledAccessFs: unix.LANDLOCK_ACCESS_FS_EXECUTE}
	rawFd, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_CREATE_RULESET,
		uintptr(unsafe.Pointer(&attr)), //nolint:gosec // G103: the kernel ABI takes a pointer to the struct
		unsafe.Sizeof(attr),
		0,
	)
	if errno != 0 {
		return fmt.Errorf("create Landlock ruleset: %w", errno)
	}

	rulesetFd := int(rawFd)
	defer unix.Close(rulesetFd) //nolint:errcheck // the ruleset is enforced or abandoned either way

	for _, path := range paths {
		err := addLandlockExecRule(rulesetFd, path)
		if err != nil {
			return fmt.Errorf("allow %q: %w", path, err)
		}
	}

	// Landlock restricts a thread, and the threads it creates afterward.
	// The Go runtime runs goroutines on threads that already exist, so every
	// one of them must restrict itself. Without CAP_SYS_ADMIN, Landlock
	// requires no_new_privs first.
	_, _, errno = syscall.AllThreadsSyscall(unix.SYS_PRCTL, unix.PR_SET_NO_NEW_PRIVS, 1, 0)
	if errno != 0 {
		return fmt.Errorf("set no_new_privs on every thread: %w", allThreadsErr(errno))
	}

	_, _, errno = syscall.AllThreadsSyscall(unix.SYS_LANDLOCK_RESTRICT_SELF, uintptr(rulesetFd), 0, 0)
	if errno != 0 {
		return fmt.Errorf("enforce Landlock ruleset on every thread: %w", allThreadsErr(errno))
	}

	return nil
}

// addLandlockExecRule allows executing path, or the files beneath it if it
// is a directory. Landlock ties the rule to the inode, so a symbolic link
// allows its target.
func addLandlockExecRule(rulesetFd int, path string) error {
	fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer unix.Close(fd) //nolint:errcheck // O_PATH descriptor

	attr := unix.LandlockPathBeneathAttr{
		Allowed_access: unix.LANDLOCK_ACCESS_FS_EXECUTE,
		Parent_fd:      int32(fd), //nolint:gosec // G115: file descriptors fit in an int32
	}

	_, _, errno := unix.Syscall6(
		unix.SYS_LANDLOCK_ADD_RULE,
		uintptr(rulesetFd),
		unix.LANDLOCK_RULE_PATH_BENEATH,
		uintptr(unsafe.Pointer(&attr)), //nolint:gosec // G103: the kernel ABI takes a pointer to the struct
		0, 0, 0,
	)
	if errno != 0 {
		return fmt.Errorf("add Landlock rule: %w", errno)
	}

	return nil
}

// landlockErrnoReason explains why probing Landlock failed.
func landlockErrnoReason(errno syscall.Errno) string {
	switch {
	case errors.Is(errno, unix.ENOSYS):
		return "the kernel has no Landlock support (Linux 5.13 or later, built with CONFIG_SECURITY_LANDLOCK)"
	case errors.Is(errno, unix.EOPNOTSUPP):
		return "Landlock is disabled at boot (add landlock to the lsm= kernel parameter)"
	case errors.Is(errno, unix.EPERM):
		return "a seccomp profile denies the Landlock system calls (allow landlock_create_ruleset, landlock_add_rule and landlock_restrict_self)"
	default:
		return errno.Error()
	}
}

// allThreadsErr explains a failed [syscall.AllThreadsSyscall].
func allThreadsErr(errno syscall.Errno) error {
	if errors.Is(errno, unix.ENOTSUP) {
		return fmt.Errorf("%w: the binary uses cgo, which prevents restricting every thread", ErrExecRestrictionUnavailable)
	}

	return errno
}

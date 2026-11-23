//go:build linux

package scriptplugins

import (
	"os"
	"syscall"
)

// redirectStderr makes the descriptor of stdout point at stderr, so anything a
// script prints can't reach the protocol pipe. Linux on arm64 has no Dup2, so
// it is Dup3 with no flags.
func redirectStderr(stderr, stdout *os.File) error {
	return syscall.Dup3(int(stderr.Fd()), int(stdout.Fd()), 0)
}

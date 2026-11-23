//go:build darwin

package scriptplugins

import (
	"os"
	"syscall"
)

// redirectStderr makes the descriptor of stdout point at stderr, so anything a
// script prints can't reach the protocol pipe.
func redirectStderr(stderr, stdout *os.File) error {
	return syscall.Dup2(int(stderr.Fd()), int(stdout.Fd()))
}

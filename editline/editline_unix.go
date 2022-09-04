//go:build !windows
// +build !windows

package editline

import "syscall"

// sigTermStop aliases syscall.SIGTSTP.
const sigTermStop = syscall.SIGTSTP

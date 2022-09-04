//go:build windows
// +build windows

package editline

import "syscall"

// sigTermStop is a fake signal - it's not supported on Windows.
const sigTermStop = syscall.Signal(0)

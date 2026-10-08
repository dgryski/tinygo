//go:build (linux && !baremetal && !wasm_unknown) || (wasip1 && !wasip3) || wasip2 || wasip3

package os

import "syscall"

func pipe(p []int) error {
	return syscall.Pipe2(p, syscall.O_CLOEXEC)
}

//go:build (wasip1 && !wasip3) || wasip2 || wasip3 || js

package syscall

// Use a go:extern definition to access the errno from wasi-libc
//
//go:extern errno
var libcErrno Errno

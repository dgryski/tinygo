//go:build (wasip1 && !wasip3) || wasip2 || js

package syscall

// Use a go:extern definition to access the errno from wasi-libc
//
//go:extern errno
var libcErrno Errno

func getLibcErrno() Errno {
	return libcErrno
}

func setLibcErrno(e Errno) {
	libcErrno = e
}

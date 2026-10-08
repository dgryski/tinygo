//go:build !js && (!wasip1 || wasip3) && !wasip2 && !wasip3

package syscall

func (e Errno) Is(target error) bool { return false }

//go:build wasip2

package runtime

import (
	"unsafe"

	"internal/wasi/cli/v0.2.0/environment"
	monotonicclock "internal/wasi/clocks/v0.2.0/monotonic-clock"

	"internal/cm"
)

// Entry point for the wasi:cli/run export. The version is the one that
// wasi-libc uses, see lib/wasi-libc/wasi/p2/wit. The result is true for an
// error.
//
//go:wasmexport wasi:cli/run@0.2.12#run
func wasiCliRun() cm.BoolResult {
	callMain()
	return false
}

var args []string

//go:linkname os_runtime_args os.runtime_args
func os_runtime_args() []string {
	if args == nil {
		args = environment.GetArguments().Slice()
	}
	return args
}

//export cabi_realloc
func cabi_realloc(ptr, oldsize, align, newsize unsafe.Pointer) unsafe.Pointer {
	// Use libc_realloc (not the GC-internal realloc) so this allocation is
	// tracked in the same allocs map as malloc/free: wasi-libc's own C code
	// (e.g. wasip2_string_free) takes ownership of buffers allocated here
	// during canonical-ABI lifting and later frees them with a plain free().
	return libc_realloc(ptr, uintptr(newsize))
}

func ticksToNanoseconds(ticks timeUnit) int64 {
	return int64(ticks)
}

func nanosecondsToTicks(ns int64) timeUnit {
	return timeUnit(ns)
}

func sleepTicks(d timeUnit) {
	p := monotonicclock.SubscribeDuration(monotonicclock.Duration(d))
	p.Block()
}

func ticks() timeUnit {
	return timeUnit(monotonicclock.Now())
}

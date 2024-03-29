//go:build wasip2

package runtime

import (
	"unsafe"

	"syscall/wasi/cli/v0.2.0/environment"
	monotonicclock "syscall/wasi/clocks/v0.2.0/monotonic-clock"
)

type timeUnit int64

// libc constructors
//
//export __wasm_call_ctors
func __wasm_call_ctors()

//export wasi:cli/run@0.2.0#run
func __wasi_cli_run_run() uint32 {
	_start()
	return 0
}

//export _start
func _start() {
	// These need to be initialized early so that the heap can be initialized.
	heapStart = uintptr(unsafe.Pointer(&heapStartSymbol))
	heapEnd = uintptr(wasm_memory_size(0) * wasmPageSize)
	run()
}

// Read the command line arguments from WASI.
// For example, they can be passed to a program with wasmtime like this:
//
//	wasmtime run ./program.wasm arg1 arg2
func init() {
	__wasm_call_ctors()
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
	return realloc(ptr, uintptr(newsize))
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

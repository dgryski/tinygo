//go:build wasm && !wasi && !scheduler.none && (!wasip1 || wasip3) && !wasip2 && !wasip3 && !wasm_unknown

package runtime

//export resume
func resume() {
	go func() {
		handleEvent()
	}()

	scheduler(false)
}

//export go_scheduler
func go_scheduler() {
	scheduler(false)
}

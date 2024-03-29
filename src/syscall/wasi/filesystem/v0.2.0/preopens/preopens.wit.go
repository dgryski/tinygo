// Code generated by wit-bindgen-go. DO NOT EDIT.

//go:build !wasip1

// Package preopens represents the interface "wasi:filesystem/preopens@0.2.0".
package preopens

import (
	"github.com/ydnar/wasm-tools-go/cm"
	"syscall/wasi/filesystem/v0.2.0/types"
)

// Descriptor represents the resource "wasi:filesystem/types@0.2.0#descriptor".
//
// See [types.Descriptor] for more information.
type Descriptor = types.Descriptor

// GetDirectories represents function "wasi:filesystem/preopens@0.2.0#get-directories".
//
// Return the set of preopened directories, and their path.
//
//	get-directories: func() -> list<tuple<own<descriptor>, string>>
//
//go:nosplit
func GetDirectories() cm.List[cm.Tuple[Descriptor, string]] {
	var result cm.List[cm.Tuple[Descriptor, string]]
	wasmimport_GetDirectories(&result)
	return result
}

//go:wasmimport wasi:filesystem/preopens@0.2.0 get-directories
//go:noescape
func wasmimport_GetDirectories(result *cm.List[cm.Tuple[Descriptor, string]])

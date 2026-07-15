//go:build windows

package main

import "unsafe"

func unsafePointer[T any](value *T) unsafe.Pointer {
	return unsafe.Pointer(value)
}

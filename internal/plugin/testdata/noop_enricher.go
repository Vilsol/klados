//go:build ignore

package main

import "unsafe"

var heap [65536]byte
var heapPtr uint32

//export plugin_init
func pluginInit() int32 {
	heapPtr = uint32(uintptr(unsafe.Pointer(&heap[0])))
	return 0
}

//export plugin_destroy
func pluginDestroy() {}

//export plugin_alloc
func pluginAlloc(size uint32) uint32 {
	ptr := heapPtr
	heapPtr += size
	return ptr
}

//export plugin_free
func pluginFree(_, _ uint32) {}

// plugin_enrich copies the obj bytes unchanged and returns packed (ptr<<32|len).
//
//export plugin_enrich
func pluginEnrich(_, _ uint32, objPtr, objLen uint32) uint64 {
	if objLen == 0 {
		return 0
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(objPtr))), int(objLen))
	ptr := pluginAlloc(objLen)
	dst := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), int(objLen))
	copy(dst, src)
	return uint64(ptr)<<32 | uint64(objLen)
}

func main() {}

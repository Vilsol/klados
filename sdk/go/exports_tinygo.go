//go:build wasip1 && tinygo

package sdk

import "unsafe"

// TinyGo WASI exports using //export (TinyGo-style).

var guestHeap [2 << 20]byte
var guestHeapPtr uintptr

func tinygoAlloc(size uint32) uint32 {
	ptr := uint32(guestHeapPtr)
	guestHeapPtr += uintptr(size)
	return ptr
}

//export plugin_init
func pluginInit() int32 {
	guestHeapPtr = uintptr(unsafe.Pointer(&guestHeap[0]))
	return 0
}

//export plugin_destroy
func pluginDestroy() {}

//export plugin_alloc
func pluginAlloc(size uint32) uint32 { return tinygoAlloc(size) }

//export plugin_free
func pluginFree(_, _ uint32) {}

//export plugin_enrich
func pluginEnrich(gvrPtr, gvrLen, objPtr, objLen uint32) uint64 {
	return EnrichDispatch(tinygoAlloc, gvrPtr, gvrLen, objPtr, objLen)
}

//export plugin_on_event
func pluginOnEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) {
	DispatchOnEvent(typePtr, typeLen, payloadPtr, payloadLen)
}

//export plugin_command
func pluginCommand(idPtr, idLen uint32) {
	DispatchCommand(idPtr, idLen)
}

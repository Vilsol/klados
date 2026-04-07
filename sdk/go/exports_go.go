//go:build wasip1 && !tinygo

package sdk

import "unsafe"

// Standard Go WASI exports using //go:wasmexport (Go 1.24+).

var guestHeap [2 << 20]byte
var guestHeapPtr uintptr

//go:wasmexport plugin_init
func PluginInit() int32 {
	guestHeapPtr = uintptr(unsafe.Pointer(&guestHeap[0]))
	return 0
}

//go:wasmexport plugin_destroy
func PluginDestroy() {}

//go:wasmexport plugin_alloc
func PluginAlloc(size uint32) uint32 {
	ptr := uint32(guestHeapPtr)
	guestHeapPtr += uintptr(size)
	return ptr
}

//go:wasmexport plugin_free
func PluginFree(_, _ uint32) {}

//go:wasmexport plugin_enrich
func PluginEnrich(gvrPtr, gvrLen, objPtr, objLen uint32) uint64 {
	return EnrichDispatch(PluginAlloc, gvrPtr, gvrLen, objPtr, objLen)
}

//go:wasmexport plugin_on_event
func PluginOnEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) {
	DispatchOnEvent(typePtr, typeLen, payloadPtr, payloadLen)
}

//go:wasmexport plugin_command
func PluginCommand(idPtr, idLen uint32) {
	DispatchCommand(idPtr, idLen)
}

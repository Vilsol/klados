//go:build wasip1

package internal

import "unsafe"

// hostCall invokes a method on the klados host and returns the response length.
// Call host_read_response to retrieve the bytes after calling this.
//
//go:wasmimport klados_host host_call
func rawHostCall(methodPtr, methodLen, reqPtr, reqLen uint32) uint32

// hostReadResponse copies the pending response into the guest-provided buffer.
//
//go:wasmimport klados_host host_read_response
func rawHostReadResponse(bufPtr, bufLen uint32)

// hostLog sends a log message to the klados host.
// level: 0=debug, 1=info, 2=warn, 3=error
//
//go:wasmimport klados_host host_log
func rawHostLog(level, msgPtr, msgLen uint32)

// allocGuest calls the guest's plugin_alloc to allocate memory in guest address space.
//
//go:wasmimport klados_host host_alloc
func rawHostAlloc(size uint32) uint32

// freeGuest calls the guest's plugin_free.
//
//go:wasmimport klados_host host_free
func rawHostFree(ptr, size uint32)

// Call invokes a host method with a JSON request and returns the JSON response bytes.
func Call(method string, reqJSON []byte) []byte {
	methodBytes := []byte(method)
	methodPtr := uint32(uintptr(unsafe.Pointer(&methodBytes[0])))
	methodLen := uint32(len(methodBytes))

	var reqPtr, reqLen uint32
	if len(reqJSON) > 0 {
		reqPtr = uint32(uintptr(unsafe.Pointer(&reqJSON[0])))
		reqLen = uint32(len(reqJSON))
	}

	respLen := rawHostCall(methodPtr, methodLen, reqPtr, reqLen)
	if respLen == 0 {
		return nil
	}

	out := make([]byte, respLen)
	rawHostReadResponse(uint32(uintptr(unsafe.Pointer(&out[0]))), respLen)
	return out
}

// Log sends a log line to the host.
func Log(level uint32, msg string) {
	b := []byte(msg)
	rawHostLog(level, uint32(uintptr(unsafe.Pointer(&b[0]))), uint32(len(b)))
}

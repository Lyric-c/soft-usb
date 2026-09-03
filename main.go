// Package main — 软加密替代 ViKey 加密狗，导出兼容的 C 接口。
//
// 编译产物（.so / .dll）可直接替换原 libViKey.so / ViKey.dll，
// 无需修改调用方代码。
//
// 构建（Linux）：
//
//	CGO_ENABLED=1 go build -buildmode=c-shared -o libViKey.so .
//
// 然后将 libViKey.so 放到 pkg/system/usb/lib/x64_linux/ 下替换原文件。
//
// 构建（Windows，在 Windows 上执行）：
//
//	set CGO_ENABLED=1
//	go build -buildmode=c-shared -o ViKey.dll .
package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

// VikeyFind 查找"加密狗"（模拟：始终返回 1 个虚拟设备）。
//
//export VikeyFind
func VikeyFind(pdwCount *C.uint32_t) C.uint32_t {
	*pdwCount = 1
	return 0 // VIKEY_SUCCESS
}

// VikeyGetHID 获取"硬件 ID"（基于当前机器硬件信息生成的稳定标识）。
//
// 返回值：SHA-256(硬件标识) 的前 4 字节，以 uint32 形式返回。
// 同一机器始终返回相同值；不同机器返回不同值。
//
//export VikeyGetHID
func VikeyGetHID(index C.uint16_t, pdwHID *C.uint32_t) C.uint32_t {
	*pdwHID = C.uint32_t(machineIdUint32())
	return 0 // VIKEY_SUCCESS
}

// GetMachineId 返回完整 64 位十六进制机器 ID 字符串（排障用）。
// 返回值由调用方通过 FreeString() 释放。
//
//export GetMachineId
func GetMachineId() *C.char {
	id := getMachineId()
	return C.CString(id)
}

// DebugIdentifiers 返回采集到的全部原始硬件标识（排障用）。
// 返回值由调用方通过 FreeString() 释放。
//
//export DebugIdentifiers
func DebugIdentifiers() *C.char {
	parts := debugIdentifiers()
	if len(parts) == 0 {
		return C.CString("")
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n"
		}
		result += p
	}
	return C.CString(result)
}

// VikeyGetType   —— 未使用，返回"不支持"
//
//export VikeyGetType
func VikeyGetType(index C.uint16_t, pType *C.uint32_t) C.uint32_t {
	_ = index
	_ = unsafe.Pointer(pType)
	return 0x80000016 // VIKEY_ERROR_NO_SUPPORT
}

// FreeString 释放由 GetMachineId / DebugIdentifiers 返回的 C 字符串。
// 跨语言调用方应使用此函数释放内存，避免跨 DLL 边界 free() 导致堆损坏。
//
//export FreeString
func FreeString(s *C.char) {
	C.free(unsafe.Pointer(s))
}

func main() {}

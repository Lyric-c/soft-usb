// Package main — 软加密机器 ID 生成库（编译为 .so / .dll）
//
// 采集多个稳定硬件标识（系统安装 ID、主板 UUID/序列号、CPU 特征、
// 磁盘序列号等），用 SHA-256 综合哈希后输出 64 位十六进制字符串。
//
// 设计目标：
//   - 不同硬件（机器）生成不同的机器 ID
//   - 同一台机器重复调用生成相同的机器 ID
//   - 更换 CPU / 主板 / 硬盘 / 重装系统后允许变化（需重新授权）
//
// 采集来源越多，区分度越高、防复制能力越强。
// 单个来源缺失时自动跳过；只要至少采集到一个来源即可生成 ID。
//
// 构建（Linux → .so）：
//
//	CGO_ENABLED=1 go build -buildmode=c-shared -o libViKey.so .
//
// 构建（Windows → .dll，需在 Windows 上执行或配置 mingw 交叉编译器）：
//
//	CGO_ENABLED=1 go build -buildmode=c-shared -o ViKey.dll .
//
// 构建（macOS → .dylib）：
//
//	CGO_ENABLED=1 go build -buildmode=c-shared -o libViKey.dylib .
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
)

// getMachineId 返回基于当前机器硬件信息生成的稳定标识。
// 返回 64 位十六进制字符串；无法采集到任何硬件标识时返回空字符串。
func getMachineId() string {
	parts := collectIdentifiers()
	if len(parts) == 0 {
		return ""
	}
	return hashIdentifiers(parts...)
}

// debugIdentifiers 返回采集到的全部原始硬件标识（不哈希），供排障使用。
func debugIdentifiers() []string {
	return collectIdentifiers()
}

// hashIdentifiers 将各部分用 NUL 字节分隔后做 SHA-256 哈希，输出 hex。
// NUL 分隔避免 "a|b" 与 "a"+"|b" 之类的前缀碰撞。
func hashIdentifiers(parts ...string) string {
	h := sha256.New()
	for i, p := range parts {
		if i > 0 {
			h.Write([]byte{0})
		}
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// readFileTrimmed 读取文件并去除首尾空白；文件不存在或读取失败返回空字符串。
func readFileTrimmed(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// appendNonEmpty 将非空且未出现过的值追加到 dst 并返回，保持传入顺序与去重。
func appendNonEmpty(dst []string, vals ...string) []string {
	seen := make(map[string]bool, len(dst)+len(vals))
	for _, v := range dst {
		seen[v] = true
	}
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		dst = append(dst, v)
	}
	return dst
}

// machineIdUint32 返回机器 ID 的前 4 字节（用于 VikeyGetHID 的 DWORD 输出）。
func machineIdUint32() uint32 {
	id := getMachineId()
	if id == "" {
		return 0
	}
	// 取 SHA-256 hex 的前 8 字符（= 前 4 字节），解析为 uint32
	v, err := strconv.ParseUint(id[:8], 16, 32)
	if err != nil {
		return 0
	}
	return uint32(v)
}

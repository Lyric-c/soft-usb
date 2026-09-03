package main

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

func collectIdentifiers() []string {
	var parts []string

	// ---- 系统安装标识 ----
	// 注册表 MachineGuid：Windows 安装时生成，重装系统 / 换系统盘后变化。
	// 所有 Windows 版本（含 Server / Server Core）均有此键。
	if v := readMachineGuid(); v != "" {
		parts = append(parts, v)
	}

	// ---- 主板 UUID ----
	// 重装系统不变；更换主板后变化。
	if vals := wmicValues("csproduct", "uuid"); len(vals) > 0 {
		parts = appendNonEmpty(parts, vals...)
	} else if vals := psValues("(Get-CimInstance Win32_ComputerSystemProduct).UUID"); len(vals) > 0 {
		parts = appendNonEmpty(parts, vals...)
	}

	// ---- BIOS 序列号 ----
	if vals := wmicValues("bios", "serialnumber"); len(vals) > 0 {
		parts = appendNonEmpty(parts, vals...)
	} else if vals := psValues("(Get-CimInstance Win32_BIOS).SerialNumber"); len(vals) > 0 {
		parts = appendNonEmpty(parts, vals...)
	}

	// ---- CPU ProcessorId ----
	// x86/x64 CPU 的唯一标识（CPUID 派生）；同型号不同 CPU 不同值，
	// Docker/VM 中通常为虚拟化平台生成的固定值。
	if vals := wmicValues("cpu", "processorid"); len(vals) > 0 {
		parts = appendNonEmpty(parts, vals...)
	} else if vals := psValues("(Get-CimInstance Win32_Processor).ProcessorId"); len(vals) > 0 {
		parts = appendNonEmpty(parts, vals...)
	}

	return parts
}

// readMachineGuid 读取 Windows 注册表中的 MachineGuid。
func readMachineGuid() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Cryptography`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

// wmicValues 执行 wmic <class> get <prop> 并返回非标题行的值列表。
// 带 2 秒超时；wmic 不可用时返回 nil。
func wmicValues(class, prop string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "wmic", class, "get", prop).Output()
	if err != nil {
		return nil
	}
	return parseTableOutput(string(out), prop)
}

// psValues 执行 PowerShell 脚本并返回非空行列表。
// 带 3 秒超时；PowerShell 不可用时返回 nil。
func psValues(script string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx,
		"powershell", "-NoProfile", "-NonInteractive", "-Command", script,
	).Output()
	if err != nil {
		return nil
	}
	var vals []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			vals = append(vals, line)
		}
	}
	return vals
}

// parseTableOutput 解析 wmic 的单列表格输出，跳过标题行和空行。
func parseTableOutput(out, header string) []string {
	var vals []string
	headerLower := strings.ToLower(strings.TrimSpace(header))
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 跳过标题行（如 "UUID"）
		if strings.EqualFold(line, headerLower) {
			continue
		}
		vals = append(vals, line)
	}
	return vals
}

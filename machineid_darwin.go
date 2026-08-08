//go:build darwin

package main

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

func collectIdentifiers() []string {
	var parts []string

	// ---- IOPlatformUUID（主板 UUID，最稳定的标识） ----
	if v := ioregUUID(); v != "" {
		parts = appendNonEmpty(parts, v)
	}

	// ---- 系统硬件序列号（兜底） ----
	if v := sysProfilerSerial(); v != "" {
		parts = appendNonEmpty(parts, v)
	}

	return parts
}

func ioregUUID() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ioreg", "-d2", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "IOPlatformUUID") {
			continue
		}
		if _, v, ok := strings.Cut(line, "="); ok {
			return strings.Trim(strings.TrimSpace(v), `" `)
		}
	}
	return ""
}

func sysProfilerSerial() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "system_profiler", "SPHardwareDataType").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "Serial Number") ||
			strings.Contains(line, "Hardware UUID") {
			if _, v, ok := strings.Cut(line, ":"); ok {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

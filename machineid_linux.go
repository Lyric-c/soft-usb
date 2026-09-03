package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func collectIdentifiers() []string {
	var parts []string

	// ---- 系统安装标识 ----
	// /etc/machine-id：systemd 系统唯一标识，重装后变化。
	// Docker 容器中默认使用镜像层的值（所有容器相同），
	// 推荐挂载宿主：-v /etc/machine-id:/etc/machine-id:ro
	parts = appendNonEmpty(parts,
		readFileTrimmed("/etc/machine-id"),
		readFileTrimmed("/var/lib/dbus/machine-id"),
	)

	// ---- DMI 主板/整机信息 ----
	// 物理机/VM 可读；默认 Docker 容器中 /sys 为 tmpfs，读不到。
	// 推荐挂载宿主：-v /sys/class/dmi/id:/sys/class/dmi/id:ro
	parts = appendNonEmpty(parts,
		readFileTrimmed("/sys/class/dmi/id/product_uuid"),
		readFileTrimmed("/sys/class/dmi/id/board_serial"),
		readFileTrimmed("/sys/class/dmi/id/product_serial"),
		readFileTrimmed("/sys/class/dmi/id/board_name"),
		readFileTrimmed("/sys/class/dmi/id/product_name"),
	)

	// ---- CPU 信息 ----
	// /proc/cpuinfo 在物理机/VM/Docker 中均可读（共享内核）。
	// x86 上通常只有 model name（区分度低但稳定）；
	// ARM / 树莓派上有唯一 Serial 字段。
	parts = appendNonEmpty(parts, collectCPUInfo()...)

	// ---- 块设备序列号 ----
	// SATA / SCSI / NVMe 磁盘序列号；物理机/VM 可读，
	// 默认 Docker 容器中 /sys/block 不可见。
	parts = appendNonEmpty(parts, collectDiskSerials()...)

	return parts
}

// collectCPUInfo 解析 /proc/cpuinfo，提取 CPU 型号和序列号。
func collectCPUInfo() []string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return nil
	}
	var serials, models []string
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "Serial":
			serials = append(serials, v)
		case "model name":
			models = append(models, v)
		}
	}
	// 保持稳定顺序：先 serial 后 model name
	var out []string
	seen := map[string]bool{}
	for _, s := range serials {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, m := range models {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// collectDiskSerials 读取块设备和 NVMe 设备的序列号。
func collectDiskSerials() []string {
	var out []string

	// SATA / SCSI / virtio 等
	blocks, _ := filepath.Glob("/sys/block/*")
	// 跳过 loop、ram、dm 等虚拟设备
	sort.Strings(blocks)
	for _, b := range blocks {
		name := filepath.Base(b)
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "dm-") {
			continue
		}
		// 跳过可移动设备（U 盘、外接硬盘），避免热插拔导致 ID 漂移
		if readFileTrimmed(filepath.Join(b, "removable")) == "1" {
			continue
		}
		s := readFileTrimmed(filepath.Join(b, "device", "serial"))
		if s == "" {
			s = readFileTrimmed(filepath.Join(b, "serial"))
		}
		if s != "" {
			out = append(out, s)
		}
	}

	// NVMe
	nvmes, _ := filepath.Glob("/sys/class/nvme/*")
	sort.Strings(nvmes)
	for _, n := range nvmes {
		s := readFileTrimmed(filepath.Join(n, "serial"))
		if s != "" {
			out = append(out, s)
		}
	}

	return out
}

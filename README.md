# soft-usb — 软加密替代 ViKey 加密狗

编译产物（`.so` / `.dll`）可直接替换原 `libViKey.so` / `ViKey.dll`，
**无需修改调用方代码**。

导出的 C 接口与原 ViKey SDK 完全兼容（函数名、参数类型、返回值均一致）。

## C 接口（ViKey 兼容）

```c
// 查找加密狗 — 模拟返回 1 个虚拟设备，始终成功
uint32_t VikeyFind(uint32_t* pdwCount);

// 获取硬件 ID — 返回基于机器硬件信息生成的稳定 uint32 标识
uint32_t VikeyGetHID(uint16_t index, uint32_t* pdwHID);
```

此外还导出两个排障辅助函数：

```c
char* GetMachineId();       // 完整 64 位 hex 机器 ID（调用方 FreeString()）
char* DebugIdentifiers();   // 采集到的原始硬件标识（调用方 FreeString()）
void  FreeString(char* s);  // 释放上述函数返回的 C 字符串（推荐跨语言调用方使用）
```

## 设计原理

采集多个稳定硬件标识 → SHA-256 哈希 → 取前 4 字节 → uint32。

**不同机器**生成不同 ID；**同一台机器**重复调用生成相同 ID。
单个来源缺失时自动跳过，至少一个来源即可生成 ID。

### 触发 ID 变化的操作

| 操作 | Linux 物理机 | Linux Docker | Windows |
|---|---|---|---|
| 重启 | ❌ 不变 | ❌ 不变 | ❌ 不变 |
| 软件升级 | ❌ 不变 | ❌ 不变 | ❌ 不变 |
| 插拔 U 盘 | ❌ 不变 | ❌ 不变 | ❌ 不变 |
| 加装硬盘 | ⚠️ **变化** | ❌ 不变 | ❌ 不变 |
| 更换系统盘 | ⚠️ **变化** | ⚠️ **变化** | ⚠️ **变化** |
| 更换 CPU | ⚠️ **变化** | ⚠️ **变化** | ⚠️ **变化** |
| 更换主板 | ⚠️ **变化** | ⚠️ **变化** | ⚠️ **变化** |
| 重装系统 | ⚠️ **变化** | ⚠️ **变化** | ⚠️ **变化** |

> 加装硬盘仅在 **Linux 物理机** 场景影响 ID（详见下方采集来源说明）。
> Windows 不采集磁盘信息，不受影响。
> Docker 容器默认读不到 `/sys/block`，不受影响。

---

## 各平台采集来源详解

### Linux

| 序号 | 来源 | 路径 / 方式 | 说明 | 变动条件 |
|---|---|---|---|---|
| 1 | 系统安装 ID | `/etc/machine-id` | systemd 安装时生成，32 位 hex | 重装系统 |
| 2 | DBus 机器 ID | `/var/lib/dbus/machine-id` | 通常与 machine-id 相同，去重后合并 | 重装系统 |
| 3 | 主板 UUID | `/sys/class/dmi/id/product_uuid` | 主板固件 UUID | 更换主板 |
| 4 | 主板序列号 | `/sys/class/dmi/id/board_serial` | 主板出厂序列号 | 更换主板 |
| 5 | 整机序列号 | `/sys/class/dmi/id/product_serial` | 整机出厂序列号 | 更换整机 |
| 6 | 主板型号 | `/sys/class/dmi/id/board_name` | 主板产品名称 | 更换主板 |
| 7 | 整机型号 | `/sys/class/dmi/id/product_name` | 整机产品名称 | 更换整机 |
| 8 | CPU 序列号 | `/proc/cpuinfo` → `Serial` 字段 | ARM 平台特有（树莓派等），x86 无 | 更换 CPU |
| 9 | CPU 型号 | `/proc/cpuinfo` → `model name` 字段 | 所有平台均有 | 更换 CPU |
| 10 | 磁盘序列号 | `/sys/block/*/device/serial` | 非 removable 内置磁盘（跳过了 loop/ram/dm 等虚拟设备，也跳过了 U 盘等可移动设备） | 加装/更换内置磁盘 |

> **Docker 注意**：
> - DMI 信息（3-7）：默认容器 /sys 为 tmpfs，**读不到**，需挂载 `-v /sys/class/dmi/id:/sys/class/dmi/id:ro`
> - 机器 ID（1-2）：默认使用**镜像层**的 machine-id（同镜像所有容器相同），需挂载 `-v /etc/machine-id:/etc/machine-id:ro`
> - CPU 信息（8-9）：容器共享内核，**始终可读**宿主的真实值
> - 磁盘序列号（10）：默认容器 /sys/block 不可见，**读不到**，需挂载 `-v /sys/block:/sys/block:ro`

### Windows

| 序号 | 来源 | 方式 | 说明 | 变动条件 |
|---|---|---|---|---|
| 1 | 系统安装 ID | 注册表 `HKLM\SOFTWARE\Microsoft\Cryptography\MachineGuid` | Windows 安装时生成 | 重装系统 |
| 2 | 主板 UUID | `wmic csproduct get uuid`（超时 2s） | 主板固件 UUID | 更换主板 |
| 3 | BIOS 序列号 | `wmic bios get serialnumber`（超时 2s） | 主板出厂序列号 | 更换主板 |
| 4 | CPU ID | `wmic cpu get processorid`（超时 2s） | x86/x64 CPU 唯一标识 | 更换 CPU |

> wmic 不可用时（如 Windows 11 24H2+），自动降级为 PowerShell `Get-CimInstance`。
> **不采集磁盘序列号**，加装硬盘不影响 ID。

### macOS

| 序号 | 来源 | 方式 | 说明 | 变动条件 |
|---|---|---|---|---|
| 1 | 平台 UUID | `ioreg -d2 -c IOPlatformExpertDevice` 提取 `IOPlatformUUID` | 主板固件 UUID | 更换主板 |
| 2 | 序列号 | `system_profiler SPHardwareDataType` 提取 `Serial Number` / `Hardware UUID` | 硬件兜底 | 更换主板 |

## 编译

### Linux → libViKey.so

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o libViKey.so .
```

### Windows → ViKey.dll

在 **Windows 命令提示符**（需安装 TDM-GCC 或 MSYS2 mingw-w64）：

```cmd
set CGO_ENABLED=1
go build -buildmode=c-shared -o ViKey.dll .
```

### macOS → libViKey.dylib

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o libViKey.dylib .
```

## 替换步骤

1. 在目标平台编译对应的 `.so` / `.dll`
2. 将产物放到 TW 平台的 `pkg/system/usb/lib/` 对应目录下：
   - Linux x64：`pkg/system/usb/lib/x64_linux/libViKey.so`
   - Windows x64：`pkg/system/usb/lib/windows/ViKey.dll`
3. 重新编译 TW 平台主程序即可

## 测试

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o /tmp/libViKey.so .

cat > /tmp/test.c << 'EOF'
#include <stdio.h>
#include <stdint.h>
extern uint32_t VikeyFind(uint32_t *pdwCount);
extern uint32_t VikeyGetHID(uint16_t index, uint32_t *pdwHID);
int main() {
    uint32_t n = 0, hid = 0;
    VikeyFind(&n);
    VikeyGetHID(0, &hid);
    printf("keyNum=%u  hid=%u\n", n, hid);
    return 0;
}
EOF
gcc -o /tmp/test /tmp/test.c -L/tmp -lViKey -Wl,-rpath,/tmp
/tmp/test
```

## Docker 部署

Docker 容器默认不暴露宿主的 `/sys/class/dmi`，且 `/etc/machine-id` 来自镜像层。

**推荐挂载宿主信息**（只需只读 bind mount，无需 `--privileged`）：

```bash
docker run \
    -v /etc/machine-id:/etc/machine-id:ro \
    -v /sys/class/dmi/id:/sys/class/dmi/id:ro \
    -v /sys/block:/sys/block:ro \
    your-image
```

不挂载时降级使用 CPU 型号 + 镜像层的 machine-id，同宿主上 ID 稳定但区分度低（无法区分不同宿主）。

## 依赖

- **无运行时依赖**：.so/.dll 自包含 Go runtime，无需安装任何第三方包
- 所有信息源均为操作系统标准导出（`/proc`、`/sys`、注册表、ioreg），无需 root 权限

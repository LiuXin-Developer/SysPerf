# 系统性能监控

一个轻量级的跨平台系统性能监控工具，支持 **Windows** 与 **macOS（Intel x86_64 与 Apple Silicon ARM 均支持）**。运行单个可执行文件即可启动内置 Web 服务，通过浏览器以**仪表盘**形式实时查看 **CPU、内存、网速、GPU** 等性能数据，内网内任意设备均可访问。

## 功能特性

- **跨平台**：同一套代码可编译出 Windows（amd64）、macOS（amd64 / arm64）可执行文件
- **四项核心指标**：CPU 使用率、内存使用率、网络上行/下行速率、GPU 使用率
- **实时更新**：服务端通过 SSE（Server-Sent Events）每秒推送数据，仪表盘平滑刷新
- **内网可访问**：自动选取真实局域网网卡，手机/平板/其它电脑在同一局域网内即可打开
- **扫码即用**：启动时控制台显示内网地址二维码，手机扫码直达
- **单文件部署**：前端页面通过 `go:embed` 内嵌进可执行文件，无需任何外部文件或联网
- **极致轻量**：使用 Go 编写，单个静态可执行文件，运行时内存占用约 10-20MB
- **开箱即用**：双击/命令行运行，控制台直接显示访问地址

## 快速开始

### Windows

双击 `windowsPerformance.exe`，或在命令行中运行：

```powershell
.\windowsPerformance.exe
```

### macOS（Intel 与 Apple Silicon 通用）

在终端中运行（首次运行需赋予执行权限）：

```bash
# Apple Silicon (M1/M2/M3...) 使用 arm64 版本;Intel Mac 使用 amd64 版本
chmod +x ./sysperf-mac-arm64
./sysperf-mac-arm64
```

> macOS 首次运行若提示"无法验证开发者"，可在「系统设置 → 隐私与安全性」中点击「仍要打开」，或执行 `xattr -d com.apple.quarantine ./sysperf-mac-arm64` 解除隔离属性。

启动后控制台会显示类似如下信息：

```
============================================================
  系统性能监控  已启动
============================================================
  本机访问:   http://localhost:8080
  内网访问:   http://192.168.1.100:8080
============================================================
  扫码访问(手机与本机需在同一局域网):
  (此处显示内网访问地址对应的二维码)
  内网内其它设备可用上述「内网访问」地址打开仪表盘
  按 Ctrl+C 退出
============================================================
```

> 内网地址会自动选取真实局域网网卡(优先 `192.168.x.x`),并过滤掉 VMware / Hyper-V / WSL 等虚拟网卡;手机与本机在同一局域网时,直接用相机扫描控制台二维码即可打开仪表盘。

### 访问仪表盘

- **本机**：浏览器打开 `http://localhost:8080`
- **内网其它设备**：在同一局域网内的手机或电脑浏览器中打开控制台显示的「内网访问」地址，例如 `http://192.168.1.100:8080`

> 若内网设备无法访问，通常是防火墙拦截了入站连接：
> - **Windows**：首次运行若弹出防火墙提示，请选择「允许访问」；或手动在防火墙中放行对应端口。
> - **macOS**：在「系统设置 → 网络 → 防火墙」中允许该程序接受入站连接。

## 命令行参数

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `-port` | Web 服务监听端口 | `8080` |
| `-interval` | 采集与推送间隔（毫秒） | `1000` |

示例：使用 9000 端口、每 2 秒刷新一次：

```bash
# Windows
.\windowsPerformance.exe -port 9000 -interval 2000

# macOS
./sysperf-mac-arm64 -port 9000 -interval 2000
```

## GPU 监控说明

GPU 使用率的数据源因平台而异，且**仅提供使用率**，无法获取显存占用、温度、功耗等详细信息；如有 GPU 但暂未产生负载，使用率会显示为 0%。

### Windows

通过 **Windows 性能计数器**（`\GPU Engine(*)\Utilization Percentage`）读取：

- **支持范围**：NVIDIA、AMD、Intel 等几乎所有显卡（与任务管理器的数据源一致）
- 对所有 GPU 引擎实例求和并封顶 100%

### macOS

通过 **`ioreg`**（IOAccelerator 的 `PerformanceStatistics`）读取，无需 sudo：

- **支持范围**：Intel Mac 的集成/独立显卡，以及 Apple Silicon（M 系列）的内置 GPU
- 多 GPU 时取最大利用率
- 个别机型或系统版本若未暴露相关字段，仪表盘会显示「无可用 GPU 计数器」

## 从源码编译

### 环境要求

- Go 1.21 或更高版本（构建时会自动按 `go.mod` 拉取所需工具链）
- 跨平台编译无需 CGO（设置 `CGO_ENABLED=0` 即可），可在任意系统上交叉编译出各平台产物

`-ldflags "-s -w"` 用于去除调试信息，减小可执行文件体积。

### 在本机直接编译

```bash
# Windows
go build -ldflags "-s -w" -o dist/windowsPerformance.exe

# macOS（在 Mac 上编译,自动匹配当前架构）
go build -ldflags "-s -w" -o dist/sysperf
```

### 交叉编译（一份代码产出全平台）

在 **Windows PowerShell** 上：

```powershell
$env:CGO_ENABLED="0"

# Windows amd64
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags "-s -w" -o dist/windowsPerformance.exe .

# macOS Apple Silicon (ARM)
$env:GOOS="darwin"; $env:GOARCH="arm64"; go build -ldflags "-s -w" -o dist/sysperf-mac-arm64 .

# macOS Intel (x86_64)
$env:GOOS="darwin"; $env:GOARCH="amd64"; go build -ldflags "-s -w" -o dist/sysperf-mac-amd64 .
```

在 **macOS / Linux (bash)** 上：

```bash
export CGO_ENABLED=0
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o dist/windowsPerformance.exe .
GOOS=darwin  GOARCH=arm64 go build -ldflags "-s -w" -o dist/sysperf-mac-arm64 .
GOOS=darwin  GOARCH=amd64 go build -ldflags "-s -w" -o dist/sysperf-mac-amd64 .
```

### 进一步压缩（可选）

如已安装 [UPX](https://upx.github.io/)，可进一步压缩体积（注意:UPX 对 macOS arm64 支持有限，建议仅压缩 Windows 产物）：

```bash
upx --best dist/windowsPerformance.exe
```

## 技术栈

| 模块 | 技术 |
| --- | --- |
| 开发语言 | Go（跨平台，无需 CGO） |
| 指标采集 | [gopsutil](https://github.com/shirou/gopsutil)（CPU/内存/网络）+ Windows PDH / macOS ioreg（GPU） |
| Web 服务 | Go 标准库 `net/http` |
| 实时推送 | SSE（Server-Sent Events） |
| 二维码 | [qrterminal](https://github.com/mdp/qrterminal)（控制台二维码） |
| 前端 | 原生 HTML/CSS/JS + Canvas 仪表盘（无外部依赖，离线可用） |
| 资源内嵌 | `go:embed` |

## 项目结构

```
windowsPerformance/
├── go.mod
├── main.go                      # 入口:解析参数、启动采集与服务、打印访问地址与二维码
├── internal/
│   ├── metrics/
│   │   ├── collector.go         # 后台定时采集,聚合为快照
│   │   ├── system.go            # CPU/内存/网速采集(跨平台)
│   │   ├── gpu_windows.go       # Windows GPU 使用率采集(PDH)
│   │   ├── gpu_darwin.go        # macOS GPU 使用率采集(ioreg)
│   │   └── gpu_other.go         # 其它平台(如 Linux)空实现
│   └── server/
│       └── server.go            # HTTP 路由 + SSE 推送 + 内网 IP 探测
└── web/                         # 内嵌前端
    ├── index.html
    ├── app.js
    └── style.css
```

> GPU 采集按平台通过 Go 构建标签（`//go:build windows` / `darwin` / `!windows && !darwin`）自动选择对应实现。

## 资源占用

- **可执行文件**：约 8-15MB（`-ldflags "-s -w"` 后；UPX 压缩后更小）
- **运行内存**：约 10-20MB
- **CPU**：空闲时几乎可忽略，仅在每个采集周期短暂活动

## 常见问题

**Q：内网设备打不开页面？**
A：检查是否被防火墙拦截（Windows 防火墙 / macOS 防火墙），放行对应端口或允许程序入站；确认设备与运行主机在同一局域网。

**Q：macOS 提示"无法打开,因为无法验证开发者"？**
A：在「系统设置 → 隐私与安全性」中点击「仍要打开」，或执行 `xattr -d com.apple.quarantine <可执行文件>` 解除隔离属性。

**Q：端口被占用？**
A：使用 `-port` 参数指定其它端口，例如 `-port 9000`。

**Q：GPU 一直显示 0%？**
A：属正常现象，表示当前 GPU 无明显负载；可运行游戏或图形程序后观察变化。

## GitHub Pages 官网

项目自带静态官网，位于 [`website/`](website/) 目录，介绍功能特性与 **Vibe Coding** 开发过程。

### 启用步骤

1. 将代码推送到 GitHub 仓库
2. 打开仓库 **Settings → Pages → Build and deployment**
3. **Source** 选择 **GitHub Actions**
4. 推送 `website/` 或 `.github/workflows/pages.yml` 变更后，Actions 会自动部署
5. 也可在 Actions 页手动运行 **Deploy GitHub Pages** workflow

部署完成后访问 `https://<用户名>.github.io/<仓库名>/`。

> 部署前可在 `website/index.html` 中将 GitHub 链接替换为你的仓库地址；或设置 `window.REPO_URL`（见 `website/js/main.js`）。

## 许可证

MIT

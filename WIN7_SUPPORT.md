# Windows 7 支持说明

为了在 Windows 7 上编译和运行 SeqSync Windows，请遵循以下要求。

## 1. 编译环境要求

由于 Go 1.21 及更高版本已不再支持 Windows 7，本项目已将 Go 版本降级为 **Go 1.20**。

- **Go SDK**: 请使用 [Go 1.20.14](https://go.dev/dl/) 或该系列的最新版本进行编译。
- **Wails**: 版本号2.6.0, 正常使用 `wails build` 命令即可。

## 2. 运行环境要求

Windows 7 默认不包含运行 Wails 应用所需的 WebView2 运行时。

### 安装 WebView2 运行时

- **版本限制**: Windows 7 仅支持最高 **109** 版本的 WebView2 运行时。
- **下载地址**: 请从 Microsoft 官网或存档镜像中寻找 **WebView2 Runtime version 109** 的 Evergreen Bootstrapper 或 Standalone Installer。
- **安装步骤**:
  1. 下载 `MicrosoftEdgeWebView2RuntimeInstallerX64.exe` (v109)。
  2. 在 Windows 7 上运行并完成安装。

## 3. 已知限制

- **安全性**: 微软已停止支持 Windows 7，建议仅在受控的实验室环境中使用。
- **功能**: 部分现代 UI 效果在 Windows 7 上可能表现不佳，但不影响核心同步功能的稳定性。

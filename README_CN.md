# go-mitmproxy

<div align="center" markdown="1">
   <a href="https://apps.apple.com/cn/app/sqlman-mysql-gui-%E6%95%B0%E6%8D%AE%E5%BA%93%E5%AE%A2%E6%88%B7%E7%AB%AF/id6498632117?mt=12">
     <img src="./assets/sqlman-cn.png" alt="sqlman" width="650"/>
   </a>

[欢迎使用作者开发的软件：Sqlman - MySQL GUI 数据库客户端](https://apps.apple.com/cn/app/sqlman-mysql-gui-%E6%95%B0%E6%8D%AE%E5%BA%93%E5%AE%A2%E6%88%B7%E7%AB%AF/id6498632117?mt=12)
<br>

</div>

[English](./README.md)

# gomitmproxy

<div align="center" markdown="1">
   <a href="https://apps.apple.com/cn/app/sqlman-mysql-gui-%E6%95%B0%E6%8D%AE%E5%BA%93%E5%AE%A2%E6%88%B7%E7%AB%AF/id6498632117?mt=12">
     <img src="./assets/sqlman-cn.png" alt="sqlman" width="650"/>
   </a>

[欢迎使用作者开发的软件：Sqlman - MySQL GUI 数据库客户端](https://apps.apple.com/cn/app/sqlman-mysql-gui-%E6%95%B0%E6%8D%AE%E5%BA%93%E5%AE%A2%E6%88%B7%E7%AB%AF/id6498632117?mt=12)
<br>

</div>

[English](./README.md)

`gomitmproxy` 是一个受 [mitmproxy](https://mitmproxy.org/) 启发，使用 Golang 实现的高性能中间人代理工具。它不仅是一个通用的流量拦截、检查、修改和重放工具，更是一个独立的、高度可扩展的解决方案，支持通过 Go 语言编写插件来轻松扩展功能。

## ✨ 主要功能

- **流量拦截**: 具有完整的中间人 (MITM) 能力，可拦截 HTTP 和 HTTPS 流量。
- **Web 界面**: 内置 Web UI（默认端口 9081），用于实时流量监控和检查。
- **插件系统**: 高度可扩展的架构，允许编写 Go 插件通过 `Addon` 接口实时修改请求/响应。
- **TLS 指纹模拟**: 模拟不同的浏览器指纹 (JA3/JA4) 以规避反爬虫检测。
- **流量存储与搜索**: 将拦截的流量保存到磁盘 (DuckDB) 并支持本地全文搜索 (Bleve)。
- **Map Remote (远程映射)**: 根据规则重写请求 URL 以重定向流量到不同的目标。
- **Map Local (本地映射)**: 针对特定请求直接服务本地文件，而不是从远程服务器获取。
- **HTTP/2 支持**: 完全兼容 HTTP/2 协议。
- **证书管理**: 自动生成和管理 CA 证书，与 mitmproxy 兼容。

## 📦 安装

### 使用 `go install` (推荐)

```bash
go install github.com/retutils/gomitmproxy/cmd/go-mitmproxy@latest
```

### 源码编译

```bash
git clone https://github.com/retutils/gomitmproxy.git
cd gomitmproxy
go mod tidy
go build -o gomitmproxy ./cmd/go-mitmproxy
```

## 🚀 命令行使用

使用默认设置启动代理服务器（代理：:9080，Web UI：:9081）：

```bash
gomitmproxy
```

### 常用参数

| 参数 | 描述 | 默认值 |
|------|-------------|---------|
| `-addr` | 代理监听地址 | `:9080` |
| `-web_addr` | Web 界面监听地址 | `:9081` |
| `-ssl_insecure` | 跳过上游证书验证 | `false` |
| `-storage_dir` | 捕获流量的保存目录 | `""` |
| `-tls_fingerprint` | 要模拟的 TLS 指纹 (chrome, firefox, ios, random) | `""` |
| `-map_local` | Map Local 配置文件路径 (JSON) | `""` |
| `-map_remote` | Map Remote 配置文件路径 (JSON) | `""` |
| `-dump` | 将流量转储到文件 | `""` |
| `-proxyauth` | 代理的基础认证 (user:pass) | `""` |

查看所有可用选项：

```bash
gomitmproxy -h
```

### 证书设置
首次运行后，CA 证书将在 `~/.mitmproxy/mitmproxy-ca-cert.pem` 生成。您必须在客户端设备上安装并信任此证书才能拦截 HTTPS 流量。安装说明请参阅 [mitmproxy 文档](https://docs.mitmproxy.org/stable/concepts-certificates/)。

## 🛠 功能详情

### 1. TLS 指纹模拟
通过模仿真实浏览器来规避基于指纹的屏蔽。

**使用:**
```bash
gomitmproxy -tls_fingerprint chrome
```
支持的预设: `chrome`, `firefox`, `edge`, `safari`, `360`, `qq`, `ios`, `android`, `random`, `client`.

**自定义指纹:**
您可以捕获真实指纹并在以后使用。
1. **捕获**: `gomitmproxy -fingerprint_save my_fingerprint`
2. **列表**: `gomitmproxy -fingerprint_list`
3. **使用**: `gomitmproxy -tls_fingerprint my_fingerprint`

### 2. 流量存储与搜索
使用本地数据库 DuckDB 和 Bleve 持久化流量历史并进行搜索。

**启用存储:**
```bash
gomitmproxy -storage_dir ./data
```

**搜索:**
您可以使用有效的 Bleve 查询语法搜索存储的流。
可用字段: `Method`, `URL`, `Status`, `ReqBody`, `ResBody`, `ReqHeader`, `ResHeader`。

```bash
# 搜索特定端点的 POST 请求
gomitmproxy -storage_dir ./data -search "Method:POST +URL:api"

# 搜索特定头部值
gomitmproxy -storage_dir ./data -search "ReqHeader.Content-Type:json"
```

### 3. Map Remote (远程映射)
根据规则将请求位置重写为不同的目标。

**配置文件 (`map_remote.json`):**
```json
{
  "enable": true,
  "items": [
    {
      "from": { "path": "/old-api/*" },
      "to": {
        "protocol": "https",
        "host": "new-api.example.com",
        "path": "/v2/"
      },
      "enable": true
    }
  ]
}
```
**运行:** `gomitmproxy -map_remote map_remote.json`

### 4. Map Local (本地映射)
为特定请求服务本地文件。

**配置文件 (`map_local.json`):**
```json
{
  "enable": true,
  "items": [
    {
      "from": { "url": "https://example.com/style.css" },
      "to": { "path": "./local_style.css" },
      "enable": true
    },
    {
      "from": { "path": "/static/*" },
      "to": { "path": "./local_static_dir" },
      "enable": true
    }
  ]
}
```
**运行:** `gomitmproxy -map_local map_local.json`

## 📚 库使用

您可以将 `gomitmproxy` 用作库来构建自定义代理工具。

### 基础示例

```go
package main

import (
	"log"
	"github.com/retutils/gomitmproxy/proxy"
)

func main() {
	opts := &proxy.Options{
		Addr:              ":9080",
		StreamLargeBodies: 1024 * 1024 * 5,
        SslInsecure:       true,
	}

	p, err := proxy.NewProxy(opts)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(p.Start())
}
```

### 开发自定义 Addons (插件)

通过实现 `Addon` 接口扩展功能。

```go
package main

import (
    "log"
    "github.com/retutils/gomitmproxy/proxy"
)

// 定义您的 addon
type MyAddon struct {
    proxy.BaseAddon // 可选: 嵌入 BaseAddon 以避免实现所有方法
}

// 实现您需要的方法
func (a *MyAddon) Request(f *proxy.Flow) {
    if f.Request.URL.Host == "example.com" {
        f.Request.Header.Add("X-Intercepted-By", "gomitmproxy")
    }
}

func main() {
    opts := &proxy.Options{Addr: ":9080"}
    p, _ := proxy.NewProxy(opts)

    // 注册您的 addon
    p.AddAddon(&MyAddon{})

    p.Start()
}
```

更多详细用例请参阅 [examples](./examples)。

## 📄 License

[MIT License](./LICENSE)

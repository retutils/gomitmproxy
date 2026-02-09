# gomitmproxy

`gomitmproxy` is a robust Golang implementation of a man-in-the-middle proxy, inspired by [go-mitmproxy](https://github.com/lqqyt2423/go-mitmproxy) and [mitmproxy](https://mitmproxy.org/). It serves as a versatile, standalone tool for intercepting, inspecting, modifying, and replaying HTTP/HTTPS traffic. Built with performance and extensibility in mind, it supports a powerful plugin system, making it easy to extend functionality using Go.

## ✨ Key Features

- **Traffic Interception**: Intercepts HTTP & HTTPS traffic with full man-in-the-middle (MITM) capabilities.
- **Web Interface**: A built-in web UI (default port 9081) for real-time traffic monitoring and inspection.
- **Addon System**: Highly extensible architecture allowing you to write Go plugins to modify requests/responses on the fly.
- **TLS Fingerprinting**: Emulate different browser fingerprints (JA3/JA4) to evade bot detection.
- **Flow Storage & Search**: Save intercepted traffic to disk (DuckDB) and perform full-text search (Bleve) locally.
- **Map Remote**: Rewrite request URLs to redirect traffic to different destinations.
- **Map Local**: Serve local files instead of fetching from the remote server.
- **HTTP/2 Support**: Fully compatible with HTTP/2 protocol.
- **Certificate Management**: Automatic generation and management of CA certificates, compatible with mitmproxy.

## 📦 Installation

### Using `go install` (Recommended)

```bash
go install github.com/retutils/gomitmproxy/cmd/go-mitmproxy@latest
```

### From Source

```bash
git clone https://github.com/retutils/gomitmproxy.git
cd gomitmproxy
go mod tidy
go build -o gomitmproxy ./cmd/go-mitmproxy
```

## 🚀 Command Line Usage

Start the proxy server with default settings (Proxy: :9080, Web UI: :9081):

```bash
gomitmproxy
```

### Common Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-addr` | Proxy listen address | `:9080` |
| `-web_addr` | Web interface listen address | `:9081` |
| `-ssl_insecure` | Skip upstream certificate verification | `false` |
| `-storage_dir` | Directory to save captured flows | `""` |
| `-tls_fingerprint` | TLS fingerprint to emulate (chrome, firefox, ios, random) | `""` |
| `-map_local` | Path to Map Local config file (JSON) | `""` |
| `-map_remote` | Path to Map Remote config file (JSON) | `""` |
| `-dump` | Dump flows to file | `""` |
| `-proxyauth` | Basic auth for proxy (user:pass) | `""` |

View all available options:

```bash
gomitmproxy -h
```

### Certificate Setup
After the first run, the CA certificate is generated at `~/.mitmproxy/mitmproxy-ca-cert.pem`. You must install and trust this certificate on your client device to intercept HTTPS traffic. See [mitmproxy docs](https://docs.mitmproxy.org/stable/concepts-certificates/) for installation instructions.

## 🛠 Feature Details

### 1. TLS Fingerprinting
Evade fingerprint-based blocking by mimicking real browsers.

**Usage:**
```bash
gomitmproxy -tls_fingerprint chrome
```
Supported presets: `chrome`, `firefox`, `edge`, `safari`, `360`, `qq`, `ios`, `android`, `random`, `client`.

**Custom Fingerprints:**
You can capture a real fingerprint and use it later.
1. **Capture**: `gomitmproxy -fingerprint_save my_fingerprint`
2. **List**: `gomitmproxy -fingerprint_list`
3. **Use**: `gomitmproxy -tls_fingerprint my_fingerprint`

### 2. Flow Storage & Search
Persist traffic history and search through it using a local database usage DuckDB and Bleve.

**Enable Storage:**
```bash
gomitmproxy -storage_dir ./data
```

**Search (HTTPQL):**
You can search the stored flows using the powerful **HTTPQL** syntax.

Available fields:
- `req.method`
- `req.host`
- `req.path`
- `req.query`
- `req.body`
- `resp.code`
- `resp.body`

Operators: `eq`, `ne`, `cont`, `ncont`, `like` (glob), `regex`, `gt`, `lt`, `gte`, `lte`.

```bash
# Search for POST requests
gomitmproxy -storage_dir ./data -search 'req.method.eq:"POST"'

# Search for requests containing "api" in host and 200 OK response
gomitmproxy -storage_dir ./data -search 'req.host.cont:"api" AND resp.code.eq:200'

# Search request body for a specific user ID
gomitmproxy -storage_dir ./data -search 'req.body.cont:"user_id:12345"'

# Search response body using wildcard (glob)
gomitmproxy -storage_dir ./data -search 'resp.body.like:"*error*"'
```

### 3. Map Remote
Rewrite request locations to different destinations based on rules.

**Config File (`map_remote.json`):**
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
**Run:** `gomitmproxy -map_remote map_remote.json`

### 4. Map Local
Serve local files for specific requests.

**Config File (`map_local.json`):**
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
**Run:** `gomitmproxy -map_local map_local.json`

## 📚 Library Usage

You can use `gomitmproxy` as a library to build custom proxy tools.

### Basic Example

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

### Developing Custom Addons (Plugins)

Extend functionality by implementing the `Addon` interface.

```go
package main

import (
    "log"
    "github.com/retutils/gomitmproxy/proxy"
)

// Define your addon
type MyAddon struct {
    proxy.BaseAddon // optional: embed BaseAddon to avoid implementing all methods
}

// Implement methods you need
func (a *MyAddon) Request(f *proxy.Flow) {
    if f.Request.URL.Host == "example.com" {
        f.Request.Header.Add("X-Intercepted-By", "Go-Mitmproxy")
    }
}

func main() {
    opts := &proxy.Options{Addr: ":9080"}
    p, _ := proxy.NewProxy(opts)

    // Register your addon
    p.AddAddon(&MyAddon{})

    p.Start()
}
```

See [examples](./examples) for more detailed use cases.

## 📄 License

[MIT License](./LICENSE)

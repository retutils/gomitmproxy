# Implementation Plan: High-Performance DNS Caching with FastDialer

#### Phase 1: Core Initialization & Options [checkpoint: ffb6eee]
- [x] Task: Update `Options` struct in `proxy/proxy.go`. [fc9cf17]
    - [ ] Add `DnsResolvers []string` field.
    - [ ] Add `DnsRetries int` field.
- [x] Task: Initialize `fastdialer.Dialer` in `NewProxy`. [fc9cf17]
    - [ ] Import `github.com/projectdiscovery/fastdialer/pkg/fastdialer`.
    - [ ] Add `fastDialer *fastdialer.Dialer` field to `Proxy` struct.
    - [ ] Configure the dialer with options (resolvers, etc.) in `NewProxy`.
- [x] Task: Write tests for Dialer initialization in `proxy/proxy_test.go`. [fc9cf17]
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Core Initialization & Options' (Protocol in workflow.md)

#### Phase 2: Dialer Integration & Logic [checkpoint: 5b56fcc]
- [x] Task: Update `getUpstreamConn` in `proxy/proxy.go` to use `fastDialer`. [9e4767a]
    - [x] Replace `(&net.Dialer{}).DialContext` with `p.fastDialer.Dial`.
    - [x] Implement the retry loop for resolution failures.
- [x] Task: Add integration tests for DNS caching in `proxy/dns_cache_test.go`. [9e4767a]
    - [x] Write failing test: Verify that multiple calls to `getUpstreamConn` for the same host use the cache.
    - [x] Implement logic to satisfy the test.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Dialer Integration & Logic' (Protocol in workflow.md)

#### Phase 3: CLI Integration
- [x] Task: Add CLI flags to `cmd/go-mitmproxy/config.go`. [dc6d058]
    - [ ] `-dns_resolvers`: Comma-separated list of DNS servers.
    - [ ] `-dns_retries`: Number of retries for resolution (default: 2).
- [x] Task: Map CLI flags to proxy `Options` in `cmd/go-mitmproxy/main.go`. [dc6d058]
- [ ] Task: Verify end-to-end functionality with a live request.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: CLI Integration' (Protocol in workflow.md)

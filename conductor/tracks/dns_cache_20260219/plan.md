# Implementation Plan: High-Performance DNS Caching with FastDialer

#### Phase 1: Core Initialization & Options
- [ ] Task: Update `Options` struct in `proxy/proxy.go`.
    - [ ] Add `DnsResolvers []string` field.
    - [ ] Add `DnsRetries int` field.
- [ ] Task: Initialize `fastdialer.Dialer` in `NewProxy`.
    - [ ] Import `github.com/projectdiscovery/fastdialer/pkg/fastdialer`.
    - [ ] Add `fastDialer *fastdialer.Dialer` field to `Proxy` struct.
    - [ ] Configure the dialer with options (resolvers, etc.) in `NewProxy`.
- [ ] Task: Write tests for Dialer initialization in `proxy/proxy_test.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Core Initialization & Options' (Protocol in workflow.md)

#### Phase 2: Dialer Integration & Logic
- [ ] Task: Update `getUpstreamConn` in `proxy/proxy.go` to use `fastDialer`.
    - [ ] Replace `(&net.Dialer{}).DialContext` with `p.fastDialer.Dial`.
    - [ ] Implement the retry loop for resolution failures.
- [ ] Task: Add integration tests for DNS caching in `proxy/dns_cache_test.go`.
    - [ ] Write failing test: Verify that multiple calls to `getUpstreamConn` for the same host use the cache.
    - [ ] Implement logic to satisfy the test.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Dialer Integration & Logic' (Protocol in workflow.md)

#### Phase 3: CLI Integration
- [ ] Task: Add CLI flags to `cmd/go-mitmproxy/config.go`.
    - [ ] `-dns_resolvers`: Comma-separated list of DNS servers.
    - [ ] `-dns_retries`: Number of retries for resolution (default: 2).
- [ ] Task: Map CLI flags to proxy `Options` in `cmd/go-mitmproxy/main.go`.
- [ ] Task: Verify end-to-end functionality with a live request.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: CLI Integration' (Protocol in workflow.md)

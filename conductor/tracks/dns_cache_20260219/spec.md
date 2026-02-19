# Specification: High-Performance DNS Caching with FastDialer

## Overview
Integrate `github.com/projectdiscovery/fastdialer` into the core proxy engine to provide high-performance DNS caching and optimized connection establishment for all upstream traffic. This replaces the standard `net.Dialer` with a feature-rich dialing engine that supports multiple resolvers, TTL-based caching, and configurable retries.

## Functional Requirements
1.  **Core Integration:**
    *   Initialize a singleton `fastdialer.Dialer` instance within the `Proxy` struct.
    *   Replace the default `net.Dialer` in `proxy.getUpstreamConn` with the `fastdialer.Dialer`.
2.  **DNS Caching:**
    *   Leverage `fastdialer`'s internal in-memory DNS cache to store resolved IP addresses.
    *   Automatically respect TTL records from DNS responses.
3.  **Custom Resolvers:**
    *   Expose a configuration option to provide a list of custom DNS resolvers (e.g., `8.8.8.8`, `1.1.1.1`).
4.  **Advanced Connection Handling:**
    *   Implement configurable resolution retries to handle transient network issues or flaky DNS servers.
    *   Support optimized connection protocols (e.g., IPv4/IPv6 preference).

## Non-Functional Requirements
1.  **Performance:** DNS cache hits should resolve in sub-millisecond time, significantly reducing latency for repeat requests.
2.  **Compatibility:** Integration must not interfere with the TLS interception/spoofing logic handled by the `attacker`.
3.  **Stability:** Implement safe fallbacks or error handling for resolution failures.

## Acceptance Criteria
*   [ ] `fastdialer.Dialer` is successfully integrated into the `Proxy` lifecycle.
*   [ ] Repeat requests to the same host show 0ms DNS resolution time in logs/benchmarks.
*   [ ] Custom resolvers provided via configuration are correctly utilized.
*   [ ] Proxy correctly returns 502 error if resolution fails after the configured number of retries.
*   [ ] Existing TLS interception functionality (HTTPS) continues to work without regression.

## Out of Scope
*   Persistent (on-disk) DNS caching.
*   Visualizing DNS cache status in the Web UI.

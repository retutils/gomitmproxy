# Implementation Plan: Wappalyzer Integration for Technology Detection

#### Phase 1: Storage Layer Extension [checkpoint: e15c78f]
- [x] Task: Create `host_technologies` table in DuckDB. [54c605f]
    - [ ] Write migration/initialization logic in `storage/service.go`.
    - [ ] Table schema: `hostname (TEXT)`, `tech_name (TEXT)`, `version (TEXT)`, `categories (TEXT)`, `last_detected (TIMESTAMP)`.
    - [ ] Add unique constraint on `(hostname, tech_name)`.
- [x] Task: Implement `SaveHostTechnologies` in `storage/service.go`. [54c605f]
    - [ ] Logic to UPSERT technology entries for a given host.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Storage Layer Extension' (Protocol in workflow.md)

#### Phase 2: Core Analysis Logic (WappalyzerAddon) [checkpoint: a3b5f8b]
- [x] Task: Initialize `WappalyzerAddon` in `addon/wappalyzer.go`. [b361017]
    - [x] Import `github.com/projectdiscovery/wappalyzergo`.
    - [x] Define `WappalyzerAddon` struct with Host and Pattern caches (LRU).
- [x] Task: Implement Content Sampling Logic. [b361017]
    - [x] Logic to extract `<head>`, top of `<body>`, and footer from HTML responses.
    - [x] Implement size capping (>1MB skip) and sampling limits consistent with reference.
- [x] Task: Implement `Response` hook in `WappalyzerAddon`. [b361017]
    - [x] Perform analysis in a background goroutine.
    - [x] Chain: Check Cache -> Fast Match (URL/Headers/Cookies) -> Sampled Body Match.
    - [x] On detection: Call `storage.SaveHostTechnologies`.
- [x] Task: Validate detection logic against `wappalyzer.com` reference data (Vue.js, Nuxt.js, Cloudflare, etc.). [5f2af64]
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Core Analysis Logic' (Protocol in workflow.md)

#### Phase 3: CLI & Integration [checkpoint: dc84b81]
- [x] Task: Add `-scan_tech` flag to `cmd/go-mitmproxy/config.go` and `main.go`. [515f14c]
- [x] Task: Register `WappalyzerAddon` in `main.go` when the flag is enabled. [515f14c]
- [x] Task: Extend Search Service. [515f14c]
    - [x] Add functionality to retrieve technologies for a specific host from DuckDB.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: CLI & Integration' (Protocol in workflow.md)

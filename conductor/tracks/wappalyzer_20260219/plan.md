# Implementation Plan: Wappalyzer Integration for Technology Detection

#### Phase 1: Storage Layer Extension
- [x] Task: Create `host_technologies` table in DuckDB. [54c605f]
    - [ ] Write migration/initialization logic in `storage/service.go`.
    - [ ] Table schema: `hostname (TEXT)`, `tech_name (TEXT)`, `version (TEXT)`, `categories (TEXT)`, `last_detected (TIMESTAMP)`.
    - [ ] Add unique constraint on `(hostname, tech_name)`.
- [ ] Task: Implement `SaveHostTechnologies` in `storage/service.go`.
    - [ ] Logic to UPSERT technology entries for a given host.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Storage Layer Extension' (Protocol in workflow.md)

#### Phase 2: Core Analysis Logic (WappalyzerAddon)
- [ ] Task: Initialize `WappalyzerAddon` in `addon/wappalyzer.go`.
    - [ ] Import `github.com/projectdiscovery/wappalyzergo`.
    - [ ] Define `WappalyzerAddon` struct with Host and Pattern caches (LRU).
- [ ] Task: Implement Content Sampling Logic.
    - [ ] Logic to extract `<head>`, top of `<body>`, and footer from HTML responses.
    - [ ] Implement size capping (>1MB skip) and sampling limits consistent with reference.
- [ ] Task: Implement `Response` hook in `WappalyzerAddon`.
    - [ ] Perform analysis in a background goroutine.
    - [ ] Chain: Check Cache -> Fast Match (URL/Headers/Cookies) -> Sampled Body Match.
    - [ ] On detection: Call `storage.SaveHostTechnologies`.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Core Analysis Logic' (Protocol in workflow.md)

#### Phase 3: CLI & Integration
- [ ] Task: Add `-scan_tech` flag to `cmd/go-mitmproxy/config.go` and `main.go`.
- [ ] Task: Register `WappalyzerAddon` in `main.go` when the flag is enabled.
- [ ] Task: Extend Search Service.
    - [ ] Add functionality to retrieve technologies for a specific host from DuckDB.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: CLI & Integration' (Protocol in workflow.md)

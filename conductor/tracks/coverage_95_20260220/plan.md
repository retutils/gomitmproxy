# Implementation Plan: Reach 95% Test Coverage

#### Phase 1: CLI and Manifests
- [ ] Task: Test `cmd/go-mitmproxy/config.go` logic.
- [ ] Task: Test `cmd/go-mitmproxy/utils.go` remaining paths.
- [ ] Task: Test `cmd/dummycert/main.go` (via sub-processes or refactoring).

#### Phase 2: Addons and Core Logic
- [ ] Task: Improve `proxy/addon.go` coverage (BaseAddon methods).
- [ ] Task: Fill gaps in `addon/wappalyzer.go` and `addon/pii_addon.go`.
- [ ] Task: Improve `storage/service.go` coverage (especially Search and SaveEntry error paths).

#### Phase 3: Examples and Helpers
- [ ] Task: Add tests for missing examples (e.g., `examples/proxy-auth`).
- [ ] Task: Increase coverage for `internal/helper/`.

#### Phase 4: Final Push
- [ ] Task: Identify remaining 1-2% gaps and add targeted tests.

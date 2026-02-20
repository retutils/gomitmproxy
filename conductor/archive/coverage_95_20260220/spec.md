# Specification: Reach 95% Test Coverage

## Overview
Increase the total project test coverage from ~82.6% to 95%. This involves targeting large, low-coverage areas like CLI tools, examples, and error handling paths in core libraries.

## Functional Requirements
1.  **CLI Tools Coverage:** Add tests for `cmd/go-mitmproxy` and `cmd/dummycert`.
2.  **Addon Coverage:** Fill gaps in `BaseAddon` and specific addons like `WappalyzerAddon` and `StorageAddon`.
3.  **Storage Coverage:** Improve coverage for `storage/service.go`.
4.  **Helper Coverage:** Increase coverage for `internal/helper/`.

## Non-Functional Requirements
1.  **No Behavior Changes:** Tests must not modify existing application logic.
2.  **Stability:** Tests should be reliable and avoid flakiness (especially network-dependent ones).

## Acceptance Criteria
- [ ] Total statement coverage reaches >= 95%.
- [ ] All tests pass in CI environment.

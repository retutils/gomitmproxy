# Tech Stack: gomitmproxy

## Backend
*   **Language:** Go 1.24
*   **Networking:** Standard `net/http` for proxy logic and HTTP handling.
*   **Real-time:** `gorilla/websocket` for the Web UI bridge.
*   **Logging:** `sirupsen/logrus` for structured, leveled logging.
*   **TLS/Security:** `refraction-networking/utls` for low-level TLS manipulation and fingerprinting (JA3/JA4).
*   **Storage & Search:**
    *   **Database:** `marcboeker/go-duckdb` for high-performance flow persistence.
    *   **Indexing:** `blevesearch/bleve/v2` for full-text search capability.
*   **Scanning:** `petar-dambovaliev/aho-corasick` for high-speed string matching (PII detection).
    *   **Technology Profiling:** Custom profiling engine using Wappalyzer patterns, leveraging `github.com/projectdiscovery/wappalyzergo` for core matching logic.

## Frontend (Web UI)
*   **Framework:** React (TypeScript)
*   **Styling:** Bootstrap CSS for a responsive and functional layout.
*   **Communication:** WebSockets for live traffic updates.

## Infrastructure & Tooling
*   **Build System:** `Makefile` for common tasks (build, test, clean).
*   **Distribution:** `goreleaser` for cross-platform binary releases.
*   **Environment:** Standalone binary with embedded static assets.

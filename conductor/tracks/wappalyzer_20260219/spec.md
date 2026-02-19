# Specification: Wappalyzer Integration for Technology Detection

## Overview
Integrate `wappalyzergo` into `gomitmproxy` to perform high-performance, near real-time analysis of HTTP traffic. This feature identifies the frameworks, technologies, and infrastructure used by applications, aggregating results at the host level to build comprehensive technology profiles.

## Functional Requirements
1.  **Near Real-Time Analysis:** Implement a `WappalyzerAddon` that executes on the `Response` hook.
2.  **Multivariate Pattern Matching:** Prioritize fast matching using URL, Headers, and Cookies, followed by sampled body analysis.
3.  **Strategic Content Sampling:**
    *   **HTML Sampling:** For body analysis, prioritize the `<head>`, the beginning of the `<body>`, and the footer.
    *   **Middle Discard:** Ignore middle content (e.g., articles, repeated UI elements) to optimize processing.
    *   **Size Limits:** Enforce maximum content lengths for all analyzed content types.
4.  **Host-Aggregated Persistence:**
    *   Maintain a `host_technologies` table in DuckDB.
    *   Store: Hostname, Tech Name, Version, Category, and Last Detected timestamp.
    *   Incrementally update the host profile as new components are discovered.

## Non-Functional Requirements
1.  **High-Performance Caching:**
    *   **Multi-level Cache:** Implement Host and Pattern caching to achieve ~5ms latency for repeat requests.
    *   **Hash Cache:** Use content hashes to avoid re-analyzing identical response bodies.
2.  **Resource Management:**
    *   **Asynchronous Processing:** Analysis and database writes must occur in a separate goroutine to prevent blocking the proxy loop.
    *   **Memory Efficiency:** Use buffers and pooled objects to minimize GC pressure during heavy traffic analysis.
    *   **Analysis Cap:** Skip body analysis entirely for responses exceeding 1MB.

## Acceptance Criteria
*   [ ] `WappalyzerAddon` successfully identifies tech stacks (e.g., Nginx, React, Cloudflare) using strategic sampling.
*   [ ] Results are correctly aggregated and persisted in the `host_technologies` DuckDB table.
*   [ ] Latency impact for cached patterns is <5ms.
*   [ ] Middle-content discard logic is functional and reduces CPU usage on large HTML pages.
*   [ ] Host profiles are updated incrementally without duplicating entries.

## Out of Scope
*   Storing detection results per individual HTTP flow.
*   Web UI visualization.

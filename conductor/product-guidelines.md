# Product Guidelines: gomitmproxy

## Documentation & Voice
*   **Tone:** Technical and Precise. Documentation must prioritize accuracy, detailed specifications, and objective performance metrics.
*   **Style:** Prose should be clear and formal, avoiding fluff. Technical terms (MITM, TLS, Handshake, etc.) should be used correctly and consistently.

## Visual Identity & UI Design
*   **Characterization:** Minimalist and Functional.
*   **Principles:**
    *   **High Information Density:** The CLI and Web UI should provide as much relevant data as possible without clutter.
    *   **Utility over Aesthetics:** Design choices must serve the user's primary task—inspecting and modifying traffic—rather than pure visual appeal.
    *   **Terminal-Inspired:** Prefer monospaced fonts and high-contrast color schemes that remain readable during long debugging sessions.

## Brand & Community
*   **Messaging Approach:** Tool-First. The project's value proposition is centered entirely on its performance and utility as a software component.
*   **Focus:** Efforts should be directed toward refining the tool's core capabilities rather than marketing or building a complex brand persona.

## Performance Communication
*   **Methodology:** Benchmark-Driven.
*   **Execution:** Any claims regarding speed or efficiency must be backed by objective data, reproducible benchmarks, and resource usage comparisons (CPU/Memory).

## Addon System Design Principles
*   **Idiomatic Go:** The plugin API must leverage standard Go interfaces and patterns, making it natural for Go developers to extend the tool.
*   **Zero-Copy Performance:** The architecture should facilitate efficient, stream-oriented data processing to minimize GC pressure and latency during traffic manipulation.

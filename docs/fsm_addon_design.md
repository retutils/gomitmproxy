# Application FSM Addon Design (Draft)

## 1. Module Overview
The `FSM Addon` is a new component for `gomitmproxy` designed to build a dynamic graph of the target application's structure and behavior. It passively observes traffic, infers application states, and pushes this model to a **FalkorDB** instance for security analysis.

## 2. Architecture

### 2.1 Components
*   **FSM Addon (`addon/fsm`)**: The core Go module integrating into the proxy loop.
*   **State Manager**: Logic to normalize URLs into abstract "States" (e.g., `/user/123` -> `/user/{id}`).
*   **Session Tracker**: Logic to correlate requests to a specific user session (Cookie/Token analysis).
*   **Graph Client**: Interface to communicate with FalkorDB via Redis protocol.

### 2.2 Data Flow
1.  **Request Phase**:
    *   Identify `SessionID` from cookies/headers.
    *   Identify `CurrentState` (the URL being requested).
2.  **Response Phase**:
    *   Analyze response code (did the transition succeed?).
    *   Update the Graph:
        *   Create/Merge `Session` node.
        *   Create/Merge `State` node.
        *   Create `[:TRANSITION]` edge from `PreviousState` to `CurrentState` for this Session.

## 3. Data Model (Graph Schema)

### Nodes
*   **`Session`**
    *   `id`: string (hash of session cookie/token)
    *   `user_id`: string (extracted from JWT payload if available)
    *   `role`: string (inferred or manually tagged)
    *   `ip`: string
*   **`State` (Endpoint)**
    *   `method`: string (GET, POST)
    *   `path_template`: string (/api/v1/user/{id})
    *   `auth_type`: string (None, Bearer, Cookie)

### Edges
*   **`(:Session)-[:PERFORMED]->(:Request)`**: Links session to a specific event.
*   **`(:Request)-[:TARGETS]->(:State)`**: Links specific event to the abstract endpoint.
*   **`(:State)-[:LEADS_TO]->(:State)`**: Represents navigation flow.

## 4. Configuration
The addon will be configured via `fsm_config.yaml`:

```yaml
fsm:
  enabled: true
  falkordb_url: "redis://localhost:6379"
  session_keys: ["session_id", "Authorization"]
  path_normalization:
    - pattern: "/user/\\d+"
      replacement: "/user/{id}"
```

## 5. Security Analysis Capabilities (Future Work)
Once the graph is populated, we can run queries to detect:
*   **BOLA**: `MATCH (s1:Session)-[:TARGETS]->(r:Resource)<-[:TARGETS]-(s2:Session) WHERE s1.user_id <> s2.user_id`
*   **Workflow Bypass**: users accessing `PaymentSuccess` state without passing through `PaymentProcess`.
*   **Privilege Escalation**: identifying `User` sessions accessing `Admin` states.

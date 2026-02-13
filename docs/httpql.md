# HTTP Query Language (HTTPQL) Guide

HTTPQL is a powerful query language designed to filter and search through captured HTTP traffic in `gomitmproxy`. It allows you to construct complex queries to pinpoint specific requests and responses based on their attributes.

## Syntax Overview

A query consists of **clauses** that can be combined using logical operators (`AND`, `OR`) and grouped with parentheses.

### Clauses
A basic clause follows the format:
```
namespace.field.operator:value
```

*   **namespace**: Either `req` (request) or `resp` (response).
*   **field**: The attribute to inspect (e.g., `method`, `host`, `code`).
*   **operator**: The comparison to perform (e.g., `eq`, `cont`, `gt`).
*   **value**: The value to compare against. Strings should be quoted if they contain spaces or special characters.

### Logical Operators
*   `AND`: Both conditions must be true.
*   `OR`: At least one condition must be true.
*   `()`: Grouping for precedence.

**Example**:
```
(req.method.eq:"POST" AND req.host.cont:"api") OR resp.code.gte:500
```

---

## Fields

### Request Fields (`req`)

| Field | Type | Description |
| :--- | :--- | :--- |
| `req.method` | String | HTTP method (e.g., `GET`, `POST`). Case-sensitive. |
| `req.host` | String | The hostname of the request. |
| `req.path` | String | The path component of the URL. |
| `req.query` | String | The raw query string. |
| `req.body` | String | The request body content. (Aliases: `req.raw`) |
| `req.port` | Int | The destination port. |
| `req.tls` | Bool | Whether the request uses TLS/SSL. |

### Response Fields (`resp`)

| Field | Type | Description |
| :--- | :--- | :--- |
| `resp.code` | Int | HTTP status code. |
| `resp.body` | String | The response body content. (Aliases: `resp.raw`) |
| `resp.len` | Int | The content length of the response body. |

---

## Operators

### String Operators

| Operator | Syntax | Description | Example |
| :--- | :--- | :--- | :--- |
| `eq` | `eq:"val"` | Exact match. | `req.method.eq:"POST"` |
| `ne` | `ne:"val"` | Not equal. | `req.method.ne:"GET"` |
| `cont` | `cont:"val"` | Contains substring (or phrase). | `req.host.cont:"google"` |
| `ncont` | `ncont:"val"` | Does not contain. | `req.path.ncont:"/static"` |
| `like` | `like:"*v*"` | Wildcard match (`*` matches any sequence). | `resp.body.like:"*error*"` |
| `regex` | `regex:"^v"` | Regular expression match. | `req.path.regex:"/v[1-9]/"` |

### Integer Operators

| Operator | Syntax | Description | Example |
| :--- | :--- | :--- | :--- |
| `eq` | `eq:200` | Equal to. | `resp.code.eq:200` |
| `ne` | `ne:200` | Not equal to. | `resp.code.ne:200` |
| `gt` | `gt:100` | Greater than. | `resp.len.gt:1024` |
| `gte` | `gte:100` | Greater than or equal to. | `resp.code.gte:400` |
| `lt` | `lt:100` | Less than. | `req.port.lt:1000` |
| `lte` | `lte:100` | Less than or equal to. | `resp.code.lte:299` |

### Boolean Operators

| Operator | Syntax | Description | Example |
| :--- | :--- | :--- | :--- |
| `eq` | `eq:true` | Boolean equality. | `req.tls.eq:true` |
| `ne` | `ne:false` | Boolean inequality. | `req.tls.ne:false` |

---

## Integration with Storage

When using the `-storage_dir` feature, HTTPQL queries are translated into optimized Bleve search queries.

**Indexing behavior**:
*   `req.method`: Exact match (keyword).
*   `req.body`, `resp.body`, `host`, `path`: Standard text analysis (tokenized).
    *   `cont` on these fields performs a phrase match, respecting token order.
    *   `like` works best for pattern matching across the raw content.

**Example CLI Usage**:
```bash
gomitmproxy -storage_dir ./flows -search 'req.method.eq:"POST" AND resp.body.like:"*token*"'
```

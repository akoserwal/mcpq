# `mcpq` — MCP Quality & Context Efficiency Analyzer

[![Go Report Card](https://goreportcard.com/badge/github.com/akoserwal/mcpq)](https://goreportcard.com/report/github.com/akoserwal/mcpq)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`mcpq` is a developer tool, schema linter, live traffic proxy, and benchmark suite designed to measure, audit, and optimize **Model Context Protocol (MCP)** server context consumption, tool design modularity, and progressive disclosure efficiency.

---

## 💡 Why `mcpq`?

LLM context windows are expensive, finite, and prone to attention degradation when flooded with repetitive envelopes or monolithic payloads. Existing MCP tools test protocol compliance, but **do not audit context efficiency or progressive disclosure design**.

`mcpq` fills this gap by:
1. **Auditing Schemas (MCP001–MCP010)**: Catching monolithic tools, unbounded lists, enum bloat, and missing projections.
2. **Measuring Runtime Tokens**: Intercepting live traffic over stdio and SSE to compute P50/P90/P95/P99 token sizes and static metadata waste.
3. **Scoring Quality (0–100)**: Providing an index across Schema Efficiency, Modularity, Response Efficiency, Progressive Disclosure, and Discoverability.
4. **Generating Refactorings**: Emitting drop-in TypeScript and Go code implementing cursor pagination, field selection, and summary/detail splits.

---

## 🚀 Quickstart

### Installation

```bash
go install github.com/akoserwal/mcpq@latest
```

Or build from source:
```bash
git clone https://github.com/akoserwal/mcpq.git
cd mcpq
go build -o bin/mcpq .
```

---

## 🛠 Usage & CLI Commands

### 1. `mcpq analyze` — Static & Dynamic Audit

Audit an MCP server via stdio subprocess or SSE URL:

```bash
# Analyze a stdio MCP server
mcpq analyze -- node build/index.js
mcpq analyze -- python server.py

# Analyze an SSE MCP server
mcpq analyze --sse=http://localhost:8000/sse

# Output formats: term (default), json, md (markdown), sarif (GitHub Code Scanning)
mcpq analyze -f json -- python server.py
mcpq analyze -f md -- python server.py > audit_report.md
```

### 2. `mcpq lint` — CI/CD Pipeline Linter

Run anti-pattern rules in CI/CD and fail builds on violations or low score:

```bash
# Fail if any ERROR severity rule is violated
mcpq lint --fail-on=error -- python server.py

# Enforce a minimum quality score (e.g. 80/100)
mcpq lint --min-score=80 -- python server.py

# Export SARIF for GitHub Code Scanning
mcpq analyze -f sarif -- python server.py > results.sarif
```

### 3. `mcpq record` — Record Live LLM Traffic

Transparently intercept JSON-RPC traffic between an agent/client (e.g. Claude Desktop, Cursor) and your MCP server:

```bash
mcpq record -o traces.json -- node build/index.js
```

### 4. `mcpq eval` — Evaluate Trace Logs

Analyze pre-recorded JSON-RPC trace logs for P50/P95 response distributions and envelope waste:

```bash
mcpq eval traces.json
```

### 5. `mcpq suggest` — Progressive Disclosure Code Generator

Generate ready-to-use TypeScript and Go refactoring snippets for detected anti-patterns:

```bash
mcpq suggest --lang=ts -- python server.py
mcpq suggest --lang=go --tool=list_users -- python server.py
```

### 6. `mcpq bench` — Benchmark Scenarios

Execute automated benchmark scenarios and measure latency and token throughput:

```bash
mcpq bench -- python server.py
mcpq bench --config=scenarios.yaml -- python server.py
```

---

## 📋 Anti-Pattern Rules Catalog (MCP001–MCP010)

| Rule ID | Name | Severity | Description | Recommendation |
| :--- | :--- | :--- | :--- | :--- |
| **MCP001** | `huge-tool-description` | 🔴 Error | Tool description > 1,000 chars or > 250 tokens. | Move extended docs into prompts/resources or condense. |
| **MCP002** | `monolithic-tool` | 🔴 Error | Tool has > 8 parameters or acts as an overloaded polymorphic dispatcher. | Split into focused, single-purpose tools. |
| **MCP003** | `unbounded-response` | 🔴 Error | Collection tool (`list_*`, `search_*`) lacks `limit`, `cursor`, or `page`. | Add `limit` (default 20) and `cursor` parameters. |
| **MCP004** | `no-field-selection` | 🟠 Warning | Data retrieval tool lacks `fields` or `select` projection filter. | Add `fields?: string[]` filter or provide summary vs detail endpoints. |
| **MCP005** | `repeated-metadata` | 🔴 Error | Runtime responses contain > 15% repetitive envelope/debug waste. | Strip static metadata envelopes or move to `initialize`. |
| **MCP006** | `json-key-inefficiency` | 🟠 Warning | JSON property keys use verbose repeated class prefixes (> 28 chars). | Use concise, idiomatic property naming. |
| **MCP007** | `enum-explosion` | 🟠 Warning | Tool input schema embeds enum with > 30 entries directly inline. | Extract large enums to an MCP Resource URI or lookup tool. |
| **MCP008** | `redundant-tool-schema` | 🟠 Warning | Multiple tools share $\ge 80\%$ schema similarity without clear separation. | Consolidate duplicate tools or clarify semantic boundaries. |
| **MCP009** | `missing-property-descriptions` | 🟡 Info | Parameters lack docstrings, increasing model hallucinations. | Add clear descriptions and types for all parameters. |
| **MCP010** | `excessive-single-payload` | 🔴 Error | Single tool response exceeds 8,000 tokens at runtime. | Paginate or chunk response payloads. |

---

## 📊 Quality Scoring Formula (0–100)

$$\text{Quality Score} = \max(0, \min(100, S_{\text{schema}} + S_{\text{design}} + S_{\text{response}} + S_{\text{progressive}} + S_{\text{discover}} - P_{\text{penalties}}))$$

### Sub-Score Allocations:
- **Schema Efficiency (20 pts)**: Average tokens per tool and total schema overhead.
- **Tool Design & Modularity (20 pts)**: Parameter counts and clean property separation.
- **Response Efficiency (30 pts)**: P50/P95 token distribution and envelope waste ratio.
- **Progressive Disclosure (15 pts)**: Cursor pagination and field projection support.
- **Discoverability (10 pts)**: Parameter docstrings and concise tool summaries.

### Quality Grades:
- **95–100**: `A+` (Exceptional / Production Optimized)
- **90–94**: `A` (Great / Context Efficient)
- **80–89**: `B` (Good / Minor Optimizations Needed)
- **70–79**: `C` (Fair / Noticeable Context Waste)
- **60–69**: `D` (Poor / Severe Anti-Patterns)
- **< 60**: `F` (Failing / Bloated Payloads)

---

## 🔄 GitHub Actions CI/CD Integration

Add `mcpq` to your GitHub Actions workflow:

```yaml
name: MCP Quality Gate

on: [push, pull_request]

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      
      - name: Install mcpq
        run: go install github.com/akoserwal/mcpq@latest

      - name: Lint MCP Server
        run: mcpq lint --fail-on=error --min-score=80 -- go run ./server
```

---

## 📄 License

MIT © [akoserwal](https://github.com/akoserwal)

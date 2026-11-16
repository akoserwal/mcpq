# Run mcpq against an MCP server

This guide uses [Microsoft MarkItDown](https://github.com/microsoft/markitdown) to demonstrate server discovery, schema analysis, CI checks, and runtime measurements. Replace its startup command and benchmark arguments to use another server.

Run the examples from the **mcpq repository root**. The shell commands below use a Unix-style shell.

## 1. Build mcpq and choose a transport

Install Go 1.26.1 or newer, then build the current checkout:

```bash
go build -o bin/mcpq .
```

If you installed mcpq with `go install`, substitute `mcpq` for `./bin/mcpq` in the examples. The checkout may contain fixes that are not yet available in a published version.

| Server transport | How to connect |
| --- | --- |
| Stdio | Put the server executable and its arguments after `--`. mcpq starts and stops the subprocess. |
| Legacy HTTP + SSE | Start the server separately, then pass its SSE endpoint with `--sse`. |
| Streamable HTTP | Currently unsupported. Use the server's stdio or legacy SSE mode if available. An `/mcp` URL is not interchangeable with an SSE endpoint. |

For any stdio server, the command pattern is:

```bash
./bin/mcpq analyze --timeout 60s -- node /absolute/path/to/server.js
./bin/mcpq analyze --timeout 60s -- python /absolute/path/to/server.py
./bin/mcpq analyze --timeout 60s -- docker run --rm -i your-mcp-image
```

Choose the command that starts your server in stdio mode. Arguments before `--` belong to mcpq; arguments after it belong to the server. Pass server arguments as separate shell arguments, rather than quoting the whole command as one string. For containers, keep stdin open with `-i` and avoid allocating a TTY.

The subprocess inherits the current directory and environment. Set any credentials or configuration required by your server before launching mcpq. A GitHub repository URL is a source location, not an MCP endpoint; install the server and use its documented startup command.

## 2. Install MarkItDown's MCP server

MarkItDown provides one tool, `convert_to_markdown(uri)`, accepting `http:`, `https:`, `file:`, and `data:` URIs. Its MCP package requires Python 3.10 or newer. With [uv](https://docs.astral.sh/uv/):

```bash
uv tool install markitdown-mcp
markitdown-mcp --help
```

If the command is not found, run `uv tool update-shell` and open a new shell. Install the dependencies before running an audit so package downloads do not consume the audit timeout.

To use the GitHub source instead of the published package, clone it beside mcpq:

```bash
git clone https://github.com/microsoft/markitdown.git ../markitdown
uv sync --project ../markitdown/packages/markitdown-mcp --no-dev
```

Then replace `markitdown-mcp` after `--` in the examples with:

```bash
uv run --no-sync --project ../markitdown/packages/markitdown-mcp markitdown-mcp
```

That runs the MCP wrapper from the checkout with its declared dependencies. Record the checkout commit and installed package versions when comparing runs.

## 3. Analyze the tool schema

```bash
./bin/mcpq analyze --timeout 60s -- markitdown-mcp

# Save a machine-readable report.
./bin/mcpq analyze --format json --timeout 60s -- markitdown-mcp > markitdown-schema.json

# Save a Markdown report.
./bin/mcpq analyze --format md --timeout 60s -- markitdown-mcp > markitdown-schema.md
```

`analyze` performs the MCP handshake, lists tools, and inspects the advertised schemas. It also lists resources and prompts when the server advertises those capabilities. It does **not** call conversion tools unless you use a separate benchmark or provide recorded traces.

In the reviewed MarkItDown version, discovery returned one tool and one informational finding:

```text
MCP009: convert_to_markdown has 1 parameter missing a description: uri.
```

This finding concerns the parameter's JSON Schema description. The tool itself has a description that lists the supported URI schemes. The finding does not mean conversion failed.

## 4. Use lint as a CI gate

```bash
./bin/mcpq lint --fail-on error --min-score 80 --timeout 60s -- markitdown-mcp
```

`lint` exits nonzero when a finding meets the severity threshold or the score falls below `--min-score`. Thresholds are `error`, `warning`, `info`, and `none`. `none` disables the severity gate while leaving the minimum-score gate active.

The MarkItDown `MCP009` finding alone passes `--fail-on error`. To make informational findings fail the gate:

```bash
./bin/mcpq lint --fail-on info --timeout 60s -- markitdown-mcp
```

This is a static quality gate unless you also supply `--trace`. It does not certify runtime performance or conversion correctness.

## 5. Run a conversion benchmark

Use the included [scenario file](../examples/markitdown/scenarios.yaml):

```yaml
scenarios:
  - name: text-conversion
    tool: convert_to_markdown
    iterations: 1
    args:
      uri: "data:text/plain,Hello%20from%20MarkItDown"
```

The data URI supplies a small document directly, so this conversion needs no external document URL or local input file.

```bash
./bin/mcpq bench --config examples/markitdown/scenarios.yaml --timeout 60s -- markitdown-mcp

./bin/mcpq bench --format json --config examples/markitdown/scenarios.yaml --timeout 60s -- markitdown-mcp > markitdown-benchmark.json
```

The expected conversion text is `Hello from MarkItDown`. `bench` reports metrics, rather than the converted document. It fails on protocol errors or a tool result with `isError: true`; it does not assert that the returned text matches an expected value. Progress messages go to stderr, leaving stdout available for the report.

For another server, replace `tool` with an advertised tool name and `args` with valid arguments for that tool. Always provide a scenario file for tools with required arguments. Without `--config`, mcpq calls every discovered tool twice with empty arguments, which is unsuitable for MarkItDown's required `uri` parameter. Benchmark calls execute the tool's real behavior, including any writes it performs.

To measure your workload, add scenarios for representative inputs, such as a `file:///absolute/path/to/document.pdf` URI, and increase `iterations`. Paths must be accessible to the server process; containers need the corresponding volume mounts. A single small text conversion is a smoke check. Repeating it does not establish performance for PDFs, Office files, or large documents. `--timeout` covers the entire benchmark run, not each iteration.

## 6. Interpret the report

| Report field | Meaning |
| --- | --- |
| `qualityScore.scope` | `static`, `runtime`, `combined`, or `none`, according to the available measurements. |
| `qualityScore.measuredPoints` | Available points before normalization: 65 for static dimensions, 30 for runtime, or 95 for both. |
| `qualityScore.score` | Measured points earned, minus penalties, normalized to 0–100. Compare scores with the same scope and workload. |
| `qualityScore.runtimeSamples` | Number of response samples used. The summary explicitly qualifies a single-sample assessment. |
| `schemaAnalysis.totalTokens` | Tokens in the received tool definitions, including output schemas and schema fields unknown to the typed analyzer. |
| `traceAnalysis.totalResponseTokens` | Tokens in measured responses, including structured content. |

Without runtime traces, response efficiency is unmeasured and excluded from scoring. A high static score says nothing about large conversion outputs. Benchmark traces currently contain responses only, so request tokens are reported as zero; the total is not a complete request-and-response session cost.

The default tokenizer is `cl100k_base`; if initialization fails, mcpq falls back to an estimate. Use `--tokenizer fast` explicitly for heuristic counts without tokenizer downloads. Token counts can change with the server version, tokenizer, or input. The reviewed checkout produced 93 schema tokens and 42 response tokens for this tiny conversion; these are observations, not acceptance thresholds.

## 7. Connect over legacy SSE

With a MarkItDown version supporting `--http`, start it in one terminal:

```bash
markitdown-mcp --http --host 127.0.0.1 --port 3001
```

In another terminal, run mcpq from its repository root:

```bash
./bin/mcpq analyze --sse http://127.0.0.1:3001/sse --timeout 60s
./bin/mcpq bench --sse http://127.0.0.1:3001/sse --config examples/markitdown/scenarios.yaml --timeout 60s
```

Use the legacy SSE endpoint `/sse`, not the Streamable HTTP endpoint `/mcp`. Older MarkItDown releases may name the server flag `--sse`; check `markitdown-mcp --help`. Stop the separately started server with Ctrl+C when finished. mcpq currently has no CLI options for HTTP authorization headers or OAuth.

## 8. Evaluate real client traffic

For actual agent workloads, configure your MCP client's server command to invoke mcpq's recorder, with MarkItDown as its child command:

```json
{
  "mcpServers": {
    "markitdown": {
      "command": "/absolute/path/to/mcpq/bin/mcpq",
      "args": [
        "record", "-o", "/absolute/path/to/markitdown-traces.json",
        "--", "/absolute/path/to/markitdown-mcp"
      ]
    }
  }
}
```

Replace all paths and ensure the output directory exists. Use the client to perform conversions, then close the session so the recorder saves the trace. Starting `record` alone does not generate MCP requests.

```bash
# Runtime analysis of the saved traffic.
./bin/mcpq eval /absolute/path/to/markitdown-traces.json

# Combine those traces with a fresh schema inspection.
./bin/mcpq analyze --trace /absolute/path/to/markitdown-traces.json -- markitdown-mcp
```

Unlike the benchmark report, this trace contains the recorded protocol requests and responses. Traces include payload contents, which may include converted document text.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Executable not found | Use an absolute path to the server executable, especially in desktop client configurations. |
| Handshake timeout | Install dependencies first, confirm the server starts in stdio mode, and increase `--timeout` for slow startup. Stdio server stdout must contain MCP messages only. |
| Missing `uri` or other required argument | Supply `--config` with arguments matching the tool schema. |
| Tool reports `isError=true` | Check the input URI, file access, required converters, and server configuration. |
| SSE connection fails | Confirm the server is running and that the URL points to a legacy SSE endpoint. Streamable HTTP alone is unsupported. |
| Good score with only one response | Inspect `scope`, `runtimeSamples`, and the summary. Add representative scenarios before drawing workload-level conclusions. |

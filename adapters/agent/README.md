# External agent adapter

This Node.js adapter exposes selected Blacksand workflows to external coding-agent clients through an authenticated HTTP API. It provides a stdio MCP server and a Pi extension without replacing internal planning or workers.

## Setup

Run npm ci in adapters/agent with Node.js 22 or newer. Complete application password setup first. Set BLACKSAND_TOKEN or BLACKSAND_PASSWORD in the adapter environment. Do not commit credentials in client configuration.

Start MCP with node src/mcp.js. Configure clients to launch that entry point from the adapter directory, or use its absolute path. Pi can load pi-extension.js through its extension configuration.

| Variable | Meaning |
| --- | --- |
| BLACKSAND_URL | Backend origin; defaults to http://127.0.0.1:8787 |
| BLACKSAND_TOKEN | Existing API token |
| BLACKSAND_PASSWORD | Password for token acquisition when no token is supplied |
| BLACKSAND_ALLOW_WRITES | Exactly true exposes mutating tools; default is read-only |
| BLACKSAND_TIMEOUT_MS | 1 to 120000 milliseconds; default 30000 |
| BLACKSAND_LANGUAGE | Request language; use en for the documented interface |

Origins must use HTTPS or loopback HTTP without a path, credentials, query or fragment.

## Behavior

Tools inspect tasks, knowledge and findings. Write-enabled tools can create or control work. Names use the blacksand_ prefix; src/tools.js defines current schemas.

Requests stay on the configured origin and reject redirects. Responses are capped at 2 MiB. Timeouts do not establish whether a write committed; inspect state before repeating one. Configured credentials are redacted from errors, but returned application data still needs privacy review.

## Tests

Run npm test and npm run check. The end-to-end script needs an explicitly configured backend and disposable database; see test/e2e.test.js and CI. Never use production engagement data as its fixture.

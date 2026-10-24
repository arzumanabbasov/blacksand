# E2B execution

The Go backend, models, interface and database remain on the host. In E2B mode, supported generated commands and workspace operations run remotely with no silent local fallback.

## Setup

Use Node.js 22 or newer. Run npm ci in adapters/e2b. In System settings, enter your E2B API key and template, then save and enable E2B. Persisted settings override environment defaults.

Environment setup is also supported:

~~~text
BLACKSAND_EXECUTION=e2b
E2B_API_KEY=YOUR_E2B_KEY
BLACKSAND_E2B_TEMPLATE=base
~~~

BLACKSAND_E2B_BRIDGE can select the absolute bridge.mjs path. Otherwise the backend resolves adapters/e2b/bridge.mjs from its launch directory. Templates need Bash, Python and workflow-specific dependencies; Grep needs rg. Account limits and charges apply.

## Supported operations

The bridge supports foreground shell commands, generated Python tools, workspace file operations, remote WebFetch and remote MCP discovery/calls. MCP supports stdio, HTTP and legacy SSE. Interactive shell and managed background tools are disabled in this mode.

Browser dependencies can be installed remotely when browser MCP initializes. Host Windows commands and host-localhost services are not remote sandbox resources.

## Lifetime

Each working directory maps to a sandbox reused during the backend process lifetime. Operations within a workspace are serialized. The bridge renews a 900,000-millisecond timeout on operations; agent and task clocks are independent.

IDs and artifacts are not persisted as a recovery system. Backend restarts create new in-memory mappings. Deleted or expired workspaces fail rather than silently returning empty replacements. Cancellation and uncertain transport failures can destroy temporary files.

Use relative Linux paths. Assigned host working-directory paths can map to their remote equivalent, but arbitrary host files are not uploaded. Save durable findings through application tools and retrieve remote files before workspace loss.

## Network and checks

The host SDK uses the E2B key without passing it to generated code as an environment variable. Configured MCP secrets are supplied to the remote runtime. Other native backend network paths remain separate. Remote traffic is not automatically captured by the host's loopback proxy.

Run npm test here and go test ./execution at the root. These checks do not allocate a live sandbox. A live test needs your account and should check command execution, file persistence, cancellation and network boundaries.

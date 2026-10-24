<p align="center">
  <img src="docs/assets/blacksand-banner.svg" alt="Blacksand — agentic security research" width="100%" />
</p>

<p align="center">
  <a href="https://github.com/arzumanabbasov/blacksand/actions/workflows/ci.yml"><img src="https://github.com/arzumanabbasov/blacksand/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI" /></a>
  <img src="https://img.shields.io/badge/Go-1.26.10-00ADD8?logo=go&logoColor=white" alt="Go 1.26.10" />
  <img src="https://img.shields.io/badge/Node.js-22%2B-43853D?logo=nodedotjs&logoColor=white" alt="Node.js 22 or newer" />
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Arzuman-d8be8b" alt="Arzuman License" /></a>
</p>

<p align="center">
  <strong>Stateful agents. Evidence that lasts.</strong><br />
  A self-hosted workspace for AI-assisted security research.
</p>

<p align="center">
  <a href="#quick-start">Quick start</a> ·
  <a href="docs/README.md">Documentation</a> ·
  <a href="track-record/README.md">Track record</a> ·
  <a href="https://github.com/arzumanabbasov/blacksand/releases">Releases</a> ·
  <a href="CONTRIBUTING.md">Contribute</a>
</p>

---

Blacksand turns a research objective into a visible, reviewable agent workflow. A planner creates work intents, concurrent workers execute them, and the application keeps findings, evidence, conversations and decisions together.

Built with **Go**, **Next.js**, **PostgreSQL or SQLite**, and the **Norma agent SDK**. Run supported tools locally or in **E2B**. Connect your own model provider, MCP servers and skills.

## Capabilities

| Capability | What you get |
| --- | --- |
| Persistent orchestration | A planner and concurrent workers operating against saved task knowledge. |
| Two connected graphs | Shared asset scope alongside each task's goals, intents, hints and findings. |
| Evidence-first review | Reports, HTTP evidence bindings, retests and exports. |
| Human control | Conversations, approvals, interception rules and task controls. |
| Flexible execution | Local tools or supported remote command/file operations through E2B. |
| Model choice | Anthropic, OpenAI Chat Completions and OpenAI Responses profiles. |
| External clients | An authenticated MCP adapter and Pi extension. |
| A public field journal | Real runs, demos, failures and experiments in the track record. |

Use controlled environments and targets you have permission to assess. Review the evidence behind generated findings; model output is not an independent vulnerability verdict.

## Quick start

### Requirements

- **Go 1.26.10**, as declared in go.mod.
- **Node.js 22+** and npm.
- An API-compatible model provider for agent execution.
- PostgreSQL for Compose, or SQLite for local development.
- An E2B account only when remote execution is enabled.

### Local development · SQLite

~~~sh
git clone https://github.com/arzumanabbasov/blacksand.git
cd blacksand
npm ci --prefix web
~~~

Start the backend in one terminal:

~~~sh
BLACKSAND_DB_DRIVER=sqlite go run ./cmd/blacksand -addr 127.0.0.1:8787 -proxy 127.0.0.1:8788
~~~

On Windows PowerShell:

~~~powershell
$env:BLACKSAND_DB_DRIVER = 'sqlite'
go run ./cmd/blacksand -addr 127.0.0.1:8787 -proxy 127.0.0.1:8788
~~~

Start the interface in another terminal:

~~~sh
npm run dev --prefix web
~~~

Open **http://localhost:3000**, complete password setup, then sign in as **blacksand**. Create and test a model profile in LLM settings before starting a task.

SQLite defaults to data/blacksand.db. BLACKSAND_SQLITE_PATH selects another file. Existing installations should retain their database settings; selecting a new file does not migrate old records.

### Docker Compose · PostgreSQL

~~~sh
cp .env.example .env
# Set POSTGRES_PASSWORD and any provider settings in .env.
docker compose up -d --build
~~~

Open **http://localhost:8787**. Compose builds from source and persists PostgreSQL and application data. Review port exposure and execution permissions before remote deployment.

### E2B · optional

~~~sh
npm ci --prefix adapters/e2b
~~~

Enable E2B in System settings with your own key and template, or set BLACKSAND_EXECUTION=e2b and E2B_API_KEY on the backend. See the [E2B guide](adapters/e2b/README.md) for workspace lifetime and supported operations.

## How it works

~~~mermaid
flowchart LR
    U[You] --> API[Blacksand]
    API --> P[Planner]
    P --> I[Work intents]
    I --> W[Workers]
    W --> T[Local / E2B / MCP tools]
    W --> K[(Knowledge and findings)]
    K --> P
    K --> R[Evidence and reports]
    API --> H[Human review]
~~~

Agent runtime, turn limits, task deadlines, model output and sandbox lifetime are independent. Configure agents individually and consult [operation and budgets](docs/OPERATIONS.md) when a run ends early.

## Track record

The [track record](track-record/README.md) is our field journal: real agent runs, findings, integrations, demos, benchmarks and useful failures. Each entry records its setup, outcome and verification. Start with the [template](track-record/TEMPLATE.md). No results are claimed before an entry is published.

## Build and test

~~~sh
./build.sh
npm test --prefix web
npm test --prefix adapters/agent
npm test --prefix adapters/e2b
go test ./...
~~~

The build script requires Bash and build utilities. Some Go integration tests require disposable PostgreSQL databases; see [testing](docs/VERIFICATION.md). CI validates the frontend, adapters, source hygiene and backend on GitHub-hosted runners.

Version tags build five platform archives, checksums and a multi-architecture GHCR image. Main-branch CI does not publish releases. Automatic upstream updates are disabled; upgrade from a reviewed revision after backing up data and artifacts.

## Documentation

| Guide | Contents |
| --- | --- |
| [Setup](docs/SETUP.md) | Storage, configuration and first launch. |
| [Architecture](docs/ARCHITECTURE_STUDY_GUIDE.md) | Task flow, graphs and source map. |
| [Operation](docs/OPERATIONS.md) | Budgets, cancellation and troubleshooting. |
| [Models](docs/llm-providers.md) | Protocols, routing and token settings. |
| [Traffic evidence](docs/traffic-evidence.md) | Capture, bindings and identifier contracts. |
| [Agent adapter](adapters/agent/README.md) | External MCP clients and Pi. |
| [Contributing](CONTRIBUTING.md) | Development and contribution workflow. |
| [Security](SECURITY.md) | Private reporting and sensitive data. |
| [Releases](docs/RELEASING.md) | Verification, artifacts and publication. |

## Community

[Report a bug or suggest a feature](https://github.com/arzumanabbasov/blacksand/issues/new/choose), or open a focused pull request. Use [private vulnerability reporting](https://github.com/arzumanabbasov/blacksand/security/advisories/new) for sensitive findings.

Maintained by [Arzuman Abbasov](https://github.com/arzumanabbasov).

## Brand

The Blacksand mark combines sand dunes and an hourglass: short-lived runs, lasting evidence. Vector assets are in [docs/assets](docs/assets).

## License

[Arzuman License](LICENSE).

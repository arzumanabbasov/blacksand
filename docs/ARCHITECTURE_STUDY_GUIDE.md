# Architecture

Blacksand is primarily one Go service with a Next.js frontend. Production builds embed a static frontend export; development runs it separately.

~~~mermaid
flowchart LR
 UI[Web interface or external client] --> API[HTTP API and SSE]
 API --> E[Task engine]
 E --> P[Planner]
 E --> W[Workers]
 P --> DB[(Domain database)]
 W --> DB
 W --> G[Guards and approvals]
 G --> X[Local or E2B tools]
 X --> F[Artifacts and evidence]
~~~

The planner reads task knowledge and creates work intents. The engine schedules eligible intents. Workers execute tools, persist results and trigger further planning. Chat and independent side questions provide separate interaction paths.

## Knowledge and storage

Shared assets represent domains, IPs, ports, sites and endpoints. Company and task scope associate them with engagements. Each task's exploration graph represents goals, intents, hints and findings. Finding record IDs and exploration-node IDs are distinct.

Domain storage supports PostgreSQL and SQLite. SQLite uses a driver-boundary dialect adapter and generated schema. Traffic bodies, evidence and workspaces also involve filesystem artifacts. A database-only backup is not a complete installation backup.

## Source map

| Directory | Responsibility |
| --- | --- |
| cmd/blacksand | Startup, flags and shutdown |
| server | HTTP, authentication, scheduling and task wiring |
| agent | Model calls, prompts, planner, workers and tool assembly |
| db | Domain stores, schemas and SQL adaptation |
| execution | Remote command and file-tool routing |
| guard and intercept | Tool review and interception |
| traffic and evidence | Captured exchanges and evidence snapshots |
| llmpool and llmrec | Model routing and call recording |
| report and notify | Reports and notifications |
| sidequestion | Independent questions from captured context |
| adapters | External clients and the E2B bridge |
| web/src | Routes, components and presentation state |
| skills | Tool-specific guidance |

## Boundaries

Norma provides the model/tool loop. Blacksand owns orchestration and persistence. E2B routes supported operations remotely; it does not move every native network path off the host.

Read cmd/blacksand/main.go, server/manager.go, server/server.go and server/engine.go first. Follow agent/planner.go and agent/worker.go next. Read the traffic-evidence guide before changing finding identifiers or report bindings.

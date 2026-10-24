# Testing and verification

Compilation does not establish live provider, E2B, database or deployment compatibility. Record the tested revision, platform, command and configuration without secrets.

## Frontend

Run npm ci, npm test and npm run build:static in web. On Windows PowerShell, set $env:NEXT_EXPORT = '1' and run npm run build when POSIX environment assignment is unavailable. The production build includes TypeScript checking.

## Backend

Run go test ./... from the root. Some integration tests require PostgreSQL and supplementary schema initialization. Follow .github/workflows/ci.yml with separate disposable databases. Never run integration fixtures against production data.

Use go test -p 1 ./... on hosts with limited resources. Windows can expose platform-specific path expectations. SQLite configuration does not satisfy all PostgreSQL integration tests.

## Adapters

Run npm ci and npm test separately in adapters/agent and adapters/e2b. The agent adapter also provides npm run check. End-to-end adapter tests need a configured backend and database; ordinary unit tests do not establish that result.

## Current development receipts

On 2026-10-24, the frontend's 22 tests and production export passed. The external adapter's seven tests and four selected E2B bridge tests passed. Targeted Go checks for agent, locale, config, guard, notify, selfupdate and cmd/blacksand passed. The embedded Windows executable built successfully.

Subsequent checks passed for locale, agent, intercept, report, notify, database, execution, configuration, self-update and side-question packages, plus targeted server regressions. The Windows skill-path normalization failure was fixed. The full PostgreSQL integration suite runs on GitHub-hosted CI with disposable databases; no local Docker is required. Live model/E2B behavior, clean-machine installation and release archives are separate checks.

For documentation, check local links, command paths and packaging references. A documentation check must not be described as a live application test.

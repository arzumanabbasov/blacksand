# Contributing

Read README.md and docs/ARCHITECTURE_STUDY_GUIDE.md first. Reproduce issues with synthetic fixtures or disposable databases. Keep real targets, credentials and private evidence out of issues and patches.

Describe the problem, expected behavior, reproduction and resulting change. Include the checks performed. Keep unrelated formatting and generated output out of patches.

## Checks

Run frontend tests and a static build for interface changes. Run relevant Go packages for backend changes; integration tests can require PostgreSQL. Run the relevant adapter checks after changing integrations. See docs/VERIFICATION.md for environment requirements and known limitations.

SQLite changes need transactional and concurrency checks, not just SQL parsing. Tool changes must preserve identifier and cancellation contracts. Prompt changes need inspection of actions and stored results.

## Documentation and notices

Document current behavior with repository-relative links and accurate commands. Distinguish unit, integration, live-provider and manual checks. Do not invent account or download addresses.

Retain applicable third-party notices. Read SECURITY.md before sharing logs or reporting a vulnerability. Maintainers should establish a private reporting route before accepting sensitive reports.

## Share a run or experiment

Use the [track-record template](track-record/TEMPLATE.md) for agent runs, findings, demos and benchmarks. Include the actual setup, outcome and verification, and add the entry to the [index](track-record/README.md). Sanitize artifacts before submission.

## Dependency maintenance

TypeScript major upgrades require a manual migration of the transpilation-based tests. Keep Node type definitions on the supported Node 22 major until the runtime target changes; Dependabot ignores major updates for these two tooling dependencies. Minor, patch and security maintenance remains enabled.

Dependabot checks every Monday at 09:00 Europe/Budapest. Minor and patch version updates are grouped per ecosystem, with at most two open version-update PRs per configured ecosystem and a seven-day cooldown. Major updates stay separate. Security updates have their own group and do not wait for the version-update cooldown.

Review release notes and lockfile changes, then require the frontend, adapters, PostgreSQL integration and security checks to pass on the combined result before merging. CI rejects high or critical npm audit findings. Consolidated maintenance PRs should identify the individual PRs they replace so the obsolete branches can be closed.

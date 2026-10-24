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

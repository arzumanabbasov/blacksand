# Operation and budgets

| Limit | Meaning |
| --- | --- |
| Agent runtime | Wall-clock seconds for one run; zero means unlimited. |
| Agent turns | Maximum turns for one run; zero means unlimited. |
| Task timeout | Overall deadline that can shorten an agent run. |
| Wrap-up turns | Separate settlement allowance; zero selects the built-in default. |
| Provider output | Per-response tokens, subject to model and provider limits. |
| E2B lifetime | Remote workspace lifetime, independent of task and agent clocks. |

Configure each agent in agent management. Saved installation settings are not the defaults for new users. The SQL schema currently defaults agent runtime to 1,200 seconds and turns to zero.

Settings saved through the application rebuild relevant runtime configuration. Existing runs can retain captured values. Direct database edits may require a restart for cached agents.

## Budget exhaustion

An agent may enter settlement to save completed results. Task deadlines have separate settlement handling. Verify stored task and finding records; a final sentence does not prove that child work completed or a file was saved.

## Sandbox loss

A gone or paused sandbox is a separate execution failure. Raising runtime cannot restore a destroyed workspace. E2B operations renew a 15-minute timeout; cancellation and some transport failures deliberately destroy the sandbox. Artifacts are not automatically recovered.

Save findings through application tools and retrieve remote files while the workspace exists. See the E2B guide for scope and network limitations.

## Troubleshooting and upgrades

Check model binding, task state, queue capacity and agent enablement when work does not start. Inspect task deadlines and terminal reasons when runs end early. Confirm actual proxy routing when traffic is missing.

Back up data and stop active work deliberately before upgrading. Automatic release fetching is disabled. Use a reviewed build and matching adapters. Renamed browser session keys require signing in again.

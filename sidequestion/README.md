# Side questions

Side questions ask about captured agent context without starting another tool-executing agent. Use /btw followed by a question in supported chat and task-agent contexts. The interface supports streaming answers and follow-ups.

The service reads an immutable parent snapshot and recent side-question exchanges. It stores its history separately from the main transcript and exploration graph. New tool calls have no execution path in this service.

Closing the panel or disconnecting the event stream does not cancel the request. Use Stop to cancel the active question. Clear cancels the current request and removes side-question history without deleting the parent context snapshot.

Snapshots and request state are persisted through the domain database. Parent identity uses stable conversation or task/exploration/intent IDs, not a reusable worker slot. Changing or deleting a model profile can require a fresh parent-agent snapshot.

See [context handling](CONTEXT_BUDGET.md) and [validation](VALIDATION.md). Implementation lives in this package, agent/capture.go, server/side_questions.go and the database store.

# Side-question validation

Use synthetic snapshots and controlled provider responses. Check snapshot immutability, structured message relationships, separate history and cancellation of only the intended request.

Run go test ./sidequestion for package tests. Persistence and HTTP tests also live in db and server and may need disposable database configuration.

For a live verification, test submission, follow-up streaming, Stop, Clear, refresh and backend restart with a test account. Verify that side questions cannot execute tools or mutate the parent task graph. Exercise missing profiles, changed profiles and oversized context.

Current verification receipts must identify the tested revision and command without private questions, credentials or target content.

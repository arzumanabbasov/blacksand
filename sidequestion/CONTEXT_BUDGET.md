# Side-question context

Each question must fit the selected model's input and output budget. The service estimates request size and can summarize earlier material before requesting the answer.

The request combines a structured parent snapshot, retained side-question exchanges and the current question. Context construction preserves message relationships where possible and labels summaries as incomplete. A summary is not new evidence.

History has its own bounded allocation. context.go is the authority for current formulas, reserves and fallback behavior; this document does not duplicate numerical constants that can drift.

Summary requests do not execute tools. The final answer also has no executor, even when prior structured context includes tool definitions. Do not assume a side question inspected files or retrieved fresh findings.

Context overflow can trigger the implemented pre-output recovery path. Request cancellation and deadlines remain separate. Changed model-profile identity can invalidate older snapshots.

Use context_test.go and sidequestion_test.go for boundary checks. Size-estimation tests do not establish a live provider's context-window compatibility.

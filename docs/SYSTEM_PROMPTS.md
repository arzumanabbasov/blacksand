# System prompts

Prompts are assembled at runtime. This guide describes ownership rather than copying a snapshot that can drift from the implementation.

The main agent handles user interaction. Goal decomposition structures objectives. The planner produces work intents and assesses progress. Workers execute intents. Configured roles also support reporting, retesting and custom workflows.

Read agent/mainagent.go, agent/goals.go, agent/planner.go and agent/worker.go. Stored overrides and agent bindings are managed through the application and domain database.

Effective instructions can combine built-in role text, overrides, task scope, graph knowledge, constraints, tool descriptions, skills and context summaries. The provider's context window affects what fits in a request.

## Settlement

agent/wrapup.go owns default per-run and task-timeout settlement prompts. Saved settings can override text and turn allowances. Tool restrictions are controlled separately in code. Instructions must distinguish one run ending from an entire task ending and should preserve already justified results without starting fresh exploration.

## Editing

Use the supported override or change the relevant source. Preserve tool names and identifier contracts. Treat retrieved target material as data rather than trusted instructions. Match instructions to actual tool access. Validate changes using controlled tasks and inspect tool actions and stored results, not just the final answer.

# Product scope

Blacksand is a self-hosted workspace for AI-assisted security research. Users define tasks, inspect plans and worker activity, review findings and preserve evidence. Model output must remain distinguishable from independently established facts.

The application combines Go, Next.js, PostgreSQL or SQLite, and local or E2B execution. These deployment modes have different filesystem, lifetime and network boundaries.

Core workflows are task creation, asset/scope review, planner and worker inspection, conversation, approvals, findings, evidence and reporting. The external adapter exposes selected workflows to other clients.

The product identity is Blacksand and the interface is English-only. Branding does not replace attribution. Documentation and public copy must match the actual implementation.

Use fictional or reviewed demo data in public images. Do not invent release availability, adoption, performance or independent audit claims. Source is hosted at github.com/arzumanabbasov/blacksand; releases target ghcr.io/arzumanabbasov/blacksand. The standalone landing page is prepared for blacksand.team. Code behavior takes precedence over earlier screenshots or design notes.

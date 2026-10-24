# Security and sensitive data

Blacksand can execute commands and contact model providers and target systems. Use a controlled environment, explicit scope and suitable access controls. Its administrator workflow is not a multi-tenant authorization model.

## Private reports

Do not publish exploitable details, credentials, private targets or raw traffic in public issues. Use [GitHub private vulnerability reporting](https://github.com/arzumanabbasov/blacksand/security/advisories/new). Security fixes target the current main branch and latest release; older versions are not guaranteed support.

Provide the revision, deployment mode, minimal reproduction and impact using synthetic data where possible.

## Installation data

The domain database can retain provider keys, integration secrets, conversations and findings. jwt.key is a signing secret. Traffic captures can include credentials and cookies. Workspaces, screenshots and debug logs may contain private material.

Keep installation artifacts out of public source snapshots. Keep audit exports, browser logs and personal notes outside the public source tree. Review the Docker build context separately.

If a secret was exposed, revoke or rotate it. Deleting Git history does not revoke a credential or remove hosted copies.

## Boundaries

Local commands run with backend operating-system permissions. E2B routes supported operations remotely but does not isolate every native network path. Cancellation can destroy a sandbox and its files. Scope checks and approvals are specific controls, not proof that arbitrary agent actions are safe.

Bind services appropriately, secure remote transport, review MCP/tool configuration and test private backups before relying on them.

# Traffic evidence

The recording proxy captures HTTP exchanges routed through it. Capture records, textual finding reports and durable evidence bindings have different responsibilities.

Enable capture and configure the actual client to use the proxy, normally 127.0.0.1:8788. TLS interception requires appropriate trust configuration. Do not install a capture CA into unrelated environments.

## Bindings and IDs

Findings can bind captured exchanges as evidence snapshots. Bindings preserve body references for previews, reports and exports. An empty binding list can be valid when a finding has textual or command evidence instead.

get_finding_traffic takes the independent finding record ID returned by report_finding. update_finding_report retains the exploration-node ID as finding_id. These IDs are not interchangeable. A binding ID selects one bound exchange; body reads also select request/response side, offset and length.

Supply the evidence version actually read when updating a report with evidence_version. Re-read after bindings change.

## Remote capture and privacy

An E2B sandbox's localhost is not the host's recording proxy. Remote commands and browser sessions are not automatically captured locally. Establish a supported network path before claiming remote traffic coverage.

Traffic may include credentials, cookies and private target data. Review exports and redact shared copies. Back up the database, evidence metadata and body files together.

Relevant implementation is in server/finding_traffic.go, server/finding_traffic_export.go, traffic and evidence. Use synthetic exchanges and disposable databases for tests.

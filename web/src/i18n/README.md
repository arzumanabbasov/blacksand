# Interface messages

The supported interface is English-only. Locale normalization safely handles legacy preferences. Backend language and protocol compatibility can retain values not exposed as interface choices.

Use useI18n().t in subscribed React components and translate in non-React helpers. Preserve stable message IDs, interpolation parameters and protocol identifiers. Do not rewrite user content, credentials or persisted findings for interface localization.

Resolve static configuration labels when consumed. Use existing localized fetch and SSE helpers to preserve headers and query arguments. Use shared formatting helpers for dates and numbers.

Demo localization applies to authored initial fixtures only; user edits must remain ordinary values. Follow the relevant message namespace when adding features.

Run npm test in web for catalog, transport and fixture regressions. The production build checks TypeScript and static export behavior.

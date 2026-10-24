# Design guide

Blacksand is a research console. Give task state, agent activity, scope, findings and evidence a clear hierarchy. Keep navigation predictable and reserve emphasis for actions or changes that need attention.

Use the scope-grid mark in web/public/blacksand.svg, readable neutral surfaces and restrained accents. Reuse existing components and design tokens before creating new conventions.

Distinguish pending, running, stopped, exhausted and completed states. Do not infer completion from an agent's prose. Keep evidence accessible beside findings and explain consequential controls in user language.

Support keyboard use, visible focus, adequate contrast and smaller screens. Technical details belong in diagnostic views when they help resolve a problem.

The landing directory is static HTML, CSS and JavaScript. It must work without a backend. Use relative assets, label demo screenshots and describe actual setup and release status. Do not advertise obsolete or unconfigured download links.

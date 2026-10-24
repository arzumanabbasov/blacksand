# Model profiles

Blacksand supports Anthropic-format, OpenAI Chat Completions-format and OpenAI Responses-format endpoints. A model name alone does not establish protocol compatibility.

## Configuration

Create a profile in LLM settings. Supply a name, protocol, endpoint, model and API key. Set proxy, context, output and reasoning options for the actual provider. Test it before assigning it to agents.

Presets fill form fields only. They do not grant account access, establish billing entitlement or certify subscription-plan support. Consult your provider's current documentation for capabilities and permitted integrations.

## Tokens and reasoning

Context window describes request capacity. Output limit applies to one response. A zero output setting generally omits the explicit limit; it does not remove the model's own limits.

Some OpenAI-compatible models require max_completion_tokens instead of max_tokens. Select the profile's output-field setting accordingly. Reasoning can share the output allowance with visible text. Unsupported thinking or effort settings can cause request failures.

## Routing and credentials

Profiles can bind to agents and participate in model routing. Rate limits, pool eligibility and retry controls affect availability. An interrupted request does not prove a mutating tool action was rolled back; inspect its state before repeating it.

Keep the domain database private: it retains usable provider credentials even when the interface displays only a hint. Hiding a key is not database encryption.

Implementation references are agent/provider.go and web/src/lib/llm-profile-presets.ts. This document asserts no live provider verification.

---
description: "Use when editing thinking, reasoning, provider conversion, executor output shaping, or translator common helpers. Preserves CLIProxyAPI thinking pipeline invariants."
applyTo: "internal/thinking/**,internal/runtime/executor/**,internal/translator/common/**"
---

# Thinking Pipeline Guardrails

- Read [`AGENTS.md`](../../AGENTS.md) first; it is the canonical project guidance.
- Keep the `internal/thinking/` flow intact: suffix parsing overrides body config, then normalize to canonical `ThinkingConfig`, then validate/convert centrally, then apply provider-specific output through registered `ProviderApplier` implementations.
- Do not add provider-specific branches to `apply.go`, `validate.go`, or `convert.go`; provider-specific shapes belong under `internal/thinking/provider/<name>/`.
- Unknown providers, models without thinking metadata, and unknown user-defined models must pass through without errors.
- Executors that own thinking-capable providers should call `thinking.ApplyThinking(...)` instead of reimplementing reasoning translation.
- Do not weaken thinking/translator sentinels such as `test/thinking_conversion_test.go`, `test/claude_code_compatibility_sentinel_test.go`, `test/amp_management_test.go`, or `internal/thinking/apply_user_defined_test.go`.

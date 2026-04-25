---
trigger: glob
globs: internal/thinking/**,internal/runtime/executor/**,internal/translator/common/**
description: Invariants for the thinking/reasoning pipeline. Apply when editing thinking, executors, or shared translator helpers.
---

# Thinking pipeline invariants

The thinking pipeline is **the** seam where upstream-specific reasoning configuration is normalized. Breaking these invariants causes silent provider regressions that are only caught by `test/thinking_conversion_test.go` (which is large and slow). Keep the architecture intact.

## Canonical flow (from `internal/thinking/apply.go:ApplyThinking`)

1. **Suffix parse** — `internal/thinking/suffix.go` parses model-name suffixes (e.g. `:thinking`, `:high`); a parsed suffix **overrides** any in-body config.
2. **Normalize to canonical `ThinkingConfig`** — defined in `internal/thinking/types.go`.
3. **Validate / convert centrally** — in `internal/thinking/validate.go` and `internal/thinking/convert.go`. All cross-provider semantic decisions live here.
4. **Per-provider apply** — via `ProviderApplier` implementations under `internal/thinking/provider/<name>/`, registered through `thinking.RegisterProvider(name, applier)`.

## Hard rules

- **Never branch by provider name inside `apply.go`, `validate.go`, or `convert.go`.** Provider-specific shape lives only in `internal/thinking/provider/<name>/`. The map in `apply.go` (`providerAppliers`) is the only allowed provider-name dispatch.
- **`ApplyThinking()` is passthrough on unknowns** — unknown provider, `modelInfo.Thinking == nil`, and unknown user-defined models all return the original body without error. Preserve that contract.
- **Suffix overrides body.** Do not change precedence.
- Adding a new provider = **new directory under `internal/thinking/provider/`** + `RegisterProvider` call in `init()`. No edits to the dispatch above.
- **Do not edit `internal/translator/`** as a standalone change (guarded by `.github/workflows/pr-path-guard.yml`); see `AGENTS.md > Code Conventions` for the coordination procedure.
- Translator helpers shared across providers may live under `internal/translator/common/` but are still subject to the same PR guard — bundle changes with broader work.

## Executor coupling

- `internal/runtime/executor/` contains executors and their unit tests **only**. Any helper code (request signing, header builders, payload normalizers) goes under `internal/runtime/executor/helps/`.
- Executors should call `thinking.ApplyThinking(...)` on the outbound body when they own a thinking-capable provider — never reimplement reasoning translation inside the executor.
- Respect the timeout policy: deadlines are allowed only during credential acquisition. Documented exceptions are listed in `AGENTS.md` (Codex websocket liveness, wsrelay sessions, management `APICall`, `cmd/fetch_antigravity_models`).

## Test guardrails (do not weaken)

- `test/thinking_conversion_test.go` — fixture matrix for cross-provider conversion.
- `test/claude_code_compatibility_sentinel_test.go` — Claude wire compatibility sentinel.
- `test/amp_management_test.go` — Amp routing/management surface.
- `internal/thinking/apply_user_defined_test.go` — user-defined-model passthrough.

If a change requires updating a sentinel fixture, surface that to the user explicitly with the diff and rationale before editing the test.

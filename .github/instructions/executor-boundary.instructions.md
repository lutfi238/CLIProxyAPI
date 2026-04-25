---
description: "Use when editing provider executors, runtime executor tests, or executor support code. Keeps CLIProxyAPI executor boundaries and timeout rules intact."
applyTo: "internal/runtime/executor/**"
---

# Runtime Executor Boundary

- Read [`AGENTS.md`](../../AGENTS.md) first; it is the canonical source for executor conventions.
- Keep `internal/runtime/executor/` limited to executors and their unit tests.
- Put shared helper/support code under `internal/runtime/executor/helps/` instead of adding loose utility files beside executors.
- Use logrus structured logging and avoid logging tokens, OAuth credentials, provider secrets, or raw sensitive payloads.
- Avoid panics and `log.Fatal`/`log.Fatalf`; return errors with context where possible.
- Respect the timeout policy from `AGENTS.md`: deadlines are allowed only during credential acquisition, except the documented websocket/session/management utility exceptions.

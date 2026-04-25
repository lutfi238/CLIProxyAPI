---
description: Add a new upstream provider to CLIProxyAPI (registry → executor → thinking → tests)
---

End-to-end recipe for wiring a new upstream provider. The cross-cutting flow is **registry → runtime executor → thinking provider applier → tests**, with `internal/translator/` reserved for the maintenance team (see `AGENTS.md > Code Conventions`).

> Read [`AGENTS.md`](../../AGENTS.md) and [`.windsurf/rules/thinking-pipeline.md`](../rules/thinking-pipeline.md) before starting. Confirm with the user **which API surface** the new provider exposes (OpenAI / Gemini / Claude / Codex shape) so you don't have to touch `internal/translator/`.

## Steps

1. **Confirm scope with the user**
   - Provider name (lowercase key, e.g. `myprov`)
   - Wire protocol the upstream speaks (`openai-compat`, `gemini`, `claude`, `codex-ws`, …)
   - Auth model (API key / OAuth / device code)
   - Whether it supports thinking/reasoning output

2. **Register models in `internal/registry/`**
   - Add the static model entries in `internal/registry/model_definitions.go` (or rely on the remote `models.json` updater if upstream publishes there).
   - If models are dynamic, plumb them through `internal/registry/model_updater.go` (`StartModelsUpdater`).
   - Add a registry test in `internal/registry/*_test.go` mirroring the patterns in `model_registry_safety_test.go`.

3. **Add the runtime executor under `internal/runtime/executor/`**
   - Create `myprov_executor.go` next to existing executors (`openai_compat_executor.go`, `claude_executor.go`, `codex_executor.go`, …).
   - **Helpers go in `internal/runtime/executor/helps/`** — the executor directory is reserved for executors and their unit tests (see `AGENTS.md > Code Conventions`).
   - Follow the timeout policy: timeouts only during credential acquisition; once the upstream connection is open, no further deadlines (see `AGENTS.md` for the documented exceptions).
   - Use `logrus` structured logging; never `log.Fatal*`; redact secrets/tokens.
   - Add `myprov_executor_test.go` colocated with the executor.

4. **Wire thinking/reasoning (if applicable)** under `internal/thinking/provider/myprov/`
   - Implement a `ProviderApplier` (mirror `internal/thinking/provider/openai/`, `claude/`, `gemini/`, etc.).
   - Register via `thinking.RegisterProvider("myprov", &myApplier{})` in the package's `init()`.
   - **Do not branch by provider in `internal/thinking/apply.go`** — translation must go through the canonical `ThinkingConfig` → per-provider `ProviderApplier` path. See `internal/thinking/apply.go:ApplyThinking` and the rule file above.

5. **Config**
   - Extend the config schema in `internal/config/config.go` and update `config.example.yaml` with a fully-commented sample block.
   - If new env vars are introduced, add them to `.env.example`.
   - Respect storage backends (`PGSTORE_*`, `GITSTORE_*`, `OBJECTSTORE_*`) for any new persisted credentials.

6. **API surface (only if the provider needs new external routes)**
   - Add handlers under `internal/api/handlers/` and modules under `internal/api/modules/` (e.g., follow the `amp` module pattern at `internal/api/modules/amp/`).
   - Avoid panics in handlers; return logged errors with meaningful HTTP statuses.

7. **Tests**
   - Unit tests colocated with the executor and applier.
   - If translation behavior is exercised, add a fixture-driven case under `test/` (e.g., extend `test/thinking_conversion_test.go`) — **do not weaken existing sentinels**.
   - Use `gjson`/`sjson` for JSON inspection in tests.

8. **Verify**
   - Run `/verify-build` (or follow `.windsurf/workflows/verify-build.md`).
   - Targeted: `go test -v ./internal/thinking/... ./internal/runtime/executor/... ./internal/registry/...`

9. **Documentation**
   - If user-facing, update `docs/sdk-usage.md` (or the relevant SDK doc) — the SDK docs are the canonical reference; do **not** restate them in `AGENTS.md` or in code comments.

## Hard constraints

- **No standalone changes to `internal/translator/`** — guarded by `.github/workflows/pr-path-guard.yml`. Coordinate per the procedure in `AGENTS.md`.
- **Comments in English only.**
- **Never modify `AGENTS.md`** as part of this PR (CI guard auto-closes such PRs).

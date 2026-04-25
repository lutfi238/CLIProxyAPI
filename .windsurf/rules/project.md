---
trigger: always_on
description: CLIProxyAPI agent context entry-point. Defers all conventions, build/test commands, architecture and PR rules to AGENTS.md.
---

# Project rules — CLIProxyAPI

The authoritative agent guidance for this repo lives in [`AGENTS.md`](../../AGENTS.md). Read it first.
Treat the sections below as **deltas only** — never duplicate AGENTS.md content here.

## Always

- Before any task, follow the rules in `AGENTS.md` (commands, conventions, restricted paths, timeout policy, logging policy).
- After Go changes: run `gofmt -w .` and verify compile with `go build -o test-output ./cmd/server && rm test-output` (pwsh 7+ on this machine supports `&&`).
- Comments and new Markdown docs in English (per `AGENTS.md > Code Conventions`).

## Restricted paths (do not touch unless task explicitly requires it)

- `internal/translator/**` — guarded by `.github/workflows/pr-path-guard.yml`; see procedure in `AGENTS.md > Code Conventions`.
- `AGENTS.md` — gitignored locally and guarded by `.github/workflows/agents-md-guard.yml` (PRs that modify it are auto-closed). Update only when the user explicitly invokes `/init` or asks for it.
- `internal/registry/models/models.json` — refreshed from `router-for-me/models@main` in CI; do not hand-edit unless syncing upstream.

## Windsurf / Windows specifics

- Shell is `pwsh` (PowerShell 7.6+). `&&`/`||` chaining works; `rm` is an alias for `Remove-Item`.
- Do **not** prefix `run_command` with `cd` — pass `Cwd` instead.
- Prefer `code_search` / `grep_search` over shelling out to `rg`/`grep`.

## Quick navigation (links, not duplicates)

- Architecture: `AGENTS.md > Architecture`
- SDK entry: `sdk/cliproxy/` — docs in `docs/sdk-usage.md`, `docs/sdk-advanced.md`
- Examples: `examples/custom-provider`, `examples/http-request`, `examples/translator`
- Server entrypoint: `cmd/server/main.go`
- Config schema/loader: `internal/config/config.go` · template `config.example.yaml`

---
description: Format, compile-check and test the Go server (CLIProxyAPI verify pass)
---

Run the standard CLIProxyAPI verification pass after Go changes. All steps execute from the repo root.

## Steps

1. Format all Go code (required after any Go change per `AGENTS.md > Code Conventions`).
// turbo
```
gofmt -w .
```

2. Compile-check the server binary (build, then drop the artifact). Required by `AGENTS.md > Commands`.
// turbo
```
go build -o test-output ./cmd/server ; if (Test-Path test-output) { Remove-Item test-output }
```

3. Run the full test suite. Skip with `Ctrl+C` if you only want a targeted run.
// turbo
```
go test ./...
```

4. (Optional) Run a single test by name. Replace `TestName` and `./path/to/pkg`.
```
go test -v -run TestName ./path/to/pkg
```

## Notes

- Shell is pwsh 7+ on this machine; PowerShell-style chaining (`;`) is used in step 2 instead of `&&` so the cleanup runs even if the build is interrupted.
- The PR CI (`.github/workflows/pr-test-build.yml`) refreshes `internal/registry/models/models.json` from `router-for-me/models@main` before building — if the build passes locally but fails in CI, sync that file.
- Do not push edits to `AGENTS.md` (gitignored + guarded by `.github/workflows/agents-md-guard.yml`).

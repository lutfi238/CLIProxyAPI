---
description: "Use when a task touches internal/translator or provider protocol translation. Enforces CLIProxyAPI translator path restrictions and PR guard workflow."
applyTo: "internal/translator/**"
---

# Translator Change Guard

- Read [`AGENTS.md`](../../AGENTS.md) first; it is the canonical source for translator policy.
- Do not make standalone changes to `internal/translator/**` unless the task explicitly requires it and the permission/issue workflow in `AGENTS.md` has been followed.
- Prefer broader changes that keep translator behavior covered indirectly by integration/sentinel tests rather than isolated translator edits.
- If translator fixtures or sentinels must change, surface the rationale and expected wire-behavior difference before editing tests.
- Keep JSON test manipulation consistent with production style by using `gjson` and `sjson` where applicable.

# CLIProxyAPI-Extended - Project Context

## Apa Ini?
CLIProxyAPI-Extended adalah fork dari [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) yang menyediakan OpenAI-compatible API proxy untuk berbagai AI provider. Proxy ini memungkinkan akses ke multiple AI models melalui satu endpoint unified.

## Owner
- GitHub: lutfi238
- Email: lutfifirdaus238@gmail.com

## Tech Stack
- **Language:** Go
- **Build:** `go build -o cli-proxy-api-extended.exe ./cmd/server`
- **Config:** `config.yaml`
- **Auth Storage:** `~/.cli-proxy-api/`

## Provider yang Aktif (6 Provider)

| Provider | Auth Method | Models | Count |
|----------|-------------|--------|-------|
| **iFlow** | OAuth | gemini-2.5-flash, gemini-2.5-pro, dll | 19 |
| **Antigravity** | OAuth (`-antigravity-login`) | claude-sonnet-4-5, gemini-claude-*, dll | 9 |
| **GitHub Copilot** | OAuth Device Flow (`-copilot-login`) | gpt-5-mini, claude-opus-4.5, gpt-4o, dll | 7 |
| **Kiro (AWS/Amazon Q)** | Import dari Kiro IDE (`-kiro-import`) | claude-sonnet-4.5, claude-opus-4.5, auto | 5 |
| **Qwen** | OAuth (`-qwen-login`) | qwen3-coder-plus, qwen3-coder-flash, vision-model | 3 |
| **Cline** | Token Import | x-ai/grok-code-fast-1, dll | 2 |

## Cara Menjalankan

```powershell
# Start server
cd CLIProxyAPI-Extended
.\cli-proxy-api-extended.exe

# Server berjalan di http://127.0.0.1:8317
```

## API Usage

```bash
# List models
curl http://127.0.0.1:8317/v1/models -H "Authorization: Bearer API_KEY"

# Chat completion
curl http://127.0.0.1:8317/v1/chat/completions \
  -H "Authorization: Bearer API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"Hello"}]}'
```

## Config Penting (`config.yaml`)

```yaml
host: "127.0.0.1"
port: 8317
auth-dir: "~/.cli-proxy-api"
api-keys:
  - "YOUR_API_KEY"
debug: true
use-canonical-translator: true  # Required untuk Kiro, Copilot, Cline
show-provider-prefixes: true    # Tampilkan [Provider] di nama model
```

## Login Provider Baru

```powershell
# Antigravity (Google DeepMind)
.\cli-proxy-api-extended.exe -antigravity-login

# GitHub Copilot (Device Flow)
.\cli-proxy-api-extended.exe -copilot-login

# Kiro (multiple options)
.\cli-proxy-api-extended.exe -kiro-import          # Import dari Kiro IDE (~/.aws/sso/cache/)
.\cli-proxy-api-extended.exe -kiro-login           # Google OAuth
.\cli-proxy-api-extended.exe -kiro-google-login    # Same as -kiro-login
.\cli-proxy-api-extended.exe -kiro-aws-login       # AWS Builder ID (device code)

# Qwen (Alibaba)
.\cli-proxy-api-extended.exe -qwen-login

# iFlow
.\cli-proxy-api-extended.exe -iflow-login

# Cline
.\cli-proxy-api-extended.exe -cline-login          # Using refresh token

# Claude (jika punya subscription)
.\cli-proxy-api-extended.exe -claude-login

# Codex
.\cli-proxy-api-extended.exe -codex-login

# Vertex AI (import service account)
.\cli-proxy-api-extended.exe -vertex-import path/to/service-account.json

# Legacy Google Login
.\cli-proxy-api-extended.exe -login
```

## File Penting

| File | Deskripsi |
|------|-----------|
| `config.yaml` | Konfigurasi server (JANGAN commit - ada API key) |
| `cli-proxy-api-extended.exe` | Binary executable |
| `list_models.py` | Script Python untuk list models |
| `mcp-cliproxy/server.py` | MCP server untuk integrasi Kiro IDE |
| `~/.cli-proxy-api/*.json` | Auth token files (SENSITIF) |

## MCP Integration (Kiro IDE)

MCP server sudah dikonfigurasi di `.kiro/settings/mcp.json`:
- Tool `cliproxy_chat`: Chat dengan AI models
- Tool `cliproxy_list_models`: List available models

## Bug Fixes yang Sudah Dilakukan

1. **Copilot executor auth type mismatch** - Changed `githubCopilotAuthType` from `"copilot"` to `"github-copilot"`
2. **Missing `GeminiThinkingFromMetadata` function** - Added to `internal/util/gemini_thinking.go`
3. **Double prefix bug** - Disabled `model-prefix-provider` (use `show-provider-prefixes` only)
4. **Vision header for Kiro/Copilot** - Added `Copilot-Vision-Request: true` header for image requests (Dec 2025)
5. **Provider routing with prefix** - When using `[Kiro]` or `[Copilot]` prefix, only that provider is used (no fallback)
6. **Auth_not_found for prefixed models** - Fixed model matching when provider prefix is used

## Auto-Discovery Infrastructure (Dec 2025)

CLIProxyAPI sekarang memiliki infrastruktur auto-discovery untuk fetch models dari provider:

- **Antigravity**: ✅ Sudah auto-fetch dari backend
- **Copilot**: ✅ Infrastruktur siap (`internal/auth/copilot/models.go`) - endpoint belum tersedia
- **Kiro**: ❌ Tidak ada endpoint models
- **Qwen**: ❌ Belum diimplementasi

Ketika Copilot menambah `/models` endpoint, models akan otomatis muncul.

## GPT-5.2 Status

GPT-5.2 (rilis Dec 2025) sudah diumumkan tapi **BELUM TERSEDIA** di API:
- Error: `The requested model is not supported`
- Models di-disable (commented out) di `model_definitions.go`
- Uncomment ketika tersedia

## Rebuild Setelah Edit Code

```powershell
cd CLIProxyAPI-Extended
go build -o cli-proxy-api-extended.exe ./cmd/server
```

## Troubleshooting

### Model tidak ditemukan
- Pastikan auth file ada di `~/.cli-proxy-api/`
- Restart server setelah login provider baru

### Error 500 pada Copilot
- Re-login: `.\cli-proxy-api-extended.exe -copilot-login`
- Pastikan `githubCopilotAuthType = "github-copilot"` di executor

### Double prefix di model name
- Set `model-prefix-provider: false` di config.yaml
- Keep `show-provider-prefixes: true`

### Vision request error (400 missing Copilot-Vision-Request header)
- Sudah di-fix: header `Copilot-Vision-Request: true` ditambahkan ke Copilot & Kiro executor
- Restart server jika masih error

### Model di-route ke provider salah
- Gunakan prefix eksplisit: `[Kiro] claude-opus-4.5` atau `[Copilot] gpt-4o`
- Tanpa prefix = load balancing semua provider yang support model

## Recent Commits (Dec 2025)

1. `fix: add Copilot-Vision-Request header for vision requests`
2. `feat: add auto-discovery models infrastructure for GitHub Copilot`
3. `refactor: enforce provider prefix routing`
4. `chore: disable GPT-5.2 models (not yet available)`

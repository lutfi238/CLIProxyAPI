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

## Provider yang Aktif (4 Provider, 33+ Models)

| Provider | Auth Method | Models |
|----------|-------------|--------|
| **Antigravity** | OAuth (`-antigravity-login`) | gemini-2.5-flash, claude-sonnet-4-5, gpt-oss-120b, dll |
| **GitHub Copilot** | OAuth Device Flow (`-copilot-login`) | gpt-5-mini, claude-sonnet-4, gemini-2.5-pro, dll |
| **Kiro (Amazon Q)** | Import dari Kiro IDE (`-kiro-import`) | claude-sonnet-4.5, claude-opus-4.5, dll |
| **Qwen** | OAuth (`-qwen-login`) | qwen3-coder-plus, qwen3-coder-flash, vision-model |

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
# Antigravity (Google)
.\cli-proxy-api-extended.exe -antigravity-login

# GitHub Copilot
.\cli-proxy-api-extended.exe -copilot-login

# Kiro (import dari Kiro IDE)
.\cli-proxy-api-extended.exe -kiro-import

# Qwen
.\cli-proxy-api-extended.exe -qwen-login

# Claude (jika punya subscription)
.\cli-proxy-api-extended.exe -claude-login
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

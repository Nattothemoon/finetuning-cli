# ft — Finetuning AI CLI

`ft` is the command-line companion to [finetuning.ai](https://finetuning.ai). Generate, list, and download AI music from your terminal.

Built for Pro and Lifetime subscribers. Free / Plus tiers should use the web app — the API requires a paid plan.

---

## Install

### macOS / Linux (curl)

```bash
curl -fsSL https://raw.githubusercontent.com/Nattothemoon/finetuning-cli/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/Nattothemoon/finetuning-cli/main/scripts/install.ps1 | iex
```

### Go users

> Available once the repo moves to the `finetuning` org. Until then, clone the repo and run `go build ./cmd/ft`.

```bash
go install github.com/finetuning/cli/cmd/ft@latest
```

### Manual download

Grab a binary from the [Releases page](https://github.com/Nattothemoon/finetuning-cli/releases), unzip, and drop `ft` (or `ft.exe`) somewhere on your `PATH`.

Once installed, confirm:

```bash
ft doctor
```

---

## Authentication

1. Visit [finetuning.ai → Settings → API Keys](https://finetuning.ai) and mint a key.
2. Store it locally:

   ```bash
   ft auth login
   # Paste your API key: ft_live_********************************
   ```

The key is stored in your OS keychain (macOS Keychain, Windows Credential Manager, libsecret on Linux). On headless Linux without libsecret, it falls back to a `0600` plaintext file in your config directory.

You can also pass the key via env or flag (handy for CI):

```bash
export FINETUNING_API_KEY=ft_live_...
# or
ft me --api-key ft_live_...
```

---

## Usage

### Generate a track (the happy path)

```bash
ft generate "lofi chill piano, mellow, late night" --duration 60
# ⠋ Generating... (12s elapsed, status=processing)
# ✓ completed in 48s
# Saved to ./lofi-chill-piano-07e8d430.mp3
```

`ft generate` is the orchestrator: it submits, polls, downloads. To customize:

```bash
ft generate "upbeat ad jingle" \
  --duration 30 --bpm 128 --key D --scale major \
  --output ./jingle.mp3
```

If you want to fire and forget:

```bash
ft generate "deep ambient drone" --no-wait
# Queued. id=07e8d430-2310-4c57-87a8-cf1e6db376f7 status=processing
```

Then later:

```bash
ft get 07e8d430-2310-4c57-87a8-cf1e6db376f7
ft download 07e8d430-2310-4c57-87a8-cf1e6db376f7
```

### List + filter

```bash
ft list --limit 10
ft list --status completed
ft list --json | jq '.data.generations[].id'
```

### Account info

```bash
ft me
# Email             jane@example.com
# Tier              pro
# Monthly remaining 463 / 500
```

---

## Commands

| Command | What it does |
|---|---|
| `ft auth login` | Prompt for an API key and store it |
| `ft auth logout` | Forget the stored key |
| `ft auth whoami` | Show signed-in account |
| `ft me` | Alias of `auth whoami` |
| `ft generate <tags>` | Submit, poll, download (the orchestrator) |
| `ft list` | Show recent generations |
| `ft get <id>` | Show one generation's detail |
| `ft download <id>` | Download a completed track |
| `ft doctor` | Health check + config dump |
| `ft update` | Re-run the install script to upgrade to the latest release |

Global flags: `--api-key`, `--base-url`, `--config`, `--verbose / -v`, `--no-color`.

Every read command supports `--json` for pipe-friendly output. Spinners, prompts, and progress go to **stderr**; data goes to **stdout** — so `ft list --json | jq` always works.

---

## Configuration

Config file location:

| OS | Path |
|---|---|
| macOS | `~/Library/Application Support/finetuning/config.json` |
| Linux | `~/.config/finetuning/config.json` |
| Windows | `%APPDATA%\finetuning\config.json` |

Format:

```json
{
  "baseUrl": "https://pub.finetuning.ai",
  "defaultDuration": 60,
  "defaultBpm": 120
}
```

Override the location with `FINETUNING_CONFIG_HOME=/some/path`.

---

## Limits & rate limiting

- `POST /v1/generations`: **10 / minute / user**
- All other reads: **60 / minute / user**

When polling completes a `generate`, the CLI honors `Retry-After` automatically.

If you cancel mid-poll (Ctrl-C), the generation is **still running server-side** and **still counts a credit**. Resume with `ft get <id>` or `ft download <id>` — don't re-submit.

---

## Development

```bash
go build ./...
go test ./...

# Local cross-platform check (no goreleaser):
GOOS=windows GOARCH=amd64 go build -o /tmp/ft.exe ./cmd/ft
GOOS=darwin  GOARCH=arm64 go build -o /tmp/ft     ./cmd/ft
```

Release flow:

```bash
git tag v0.1.0
git push origin v0.1.0
# GH Action runs goreleaser; binaries appear on the Releases page.
```

---

## License

MIT — see [LICENSE](./LICENSE).

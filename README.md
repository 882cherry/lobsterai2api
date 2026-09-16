# lobsterai2api

OpenAI-compatible API bridge for LobsterAI — multi-account pool with credit-based load balancing, SSE streaming, and automatic token refresh.

- Language: Go (zero external dependencies, pure stdlib)
- Port: `:8367` (configurable via config or `LB2A_LISTEN`)

## Architecture

```
client (OpenAI SDK)
   │ POST /v1/chat/completions (Bearer ***)
   ▼
server: pool picks account (highest credits, healthy) → check token → forward
   ▼
upstream chat API (SSE only)
   ▼ on error → classify → cooldown/disable → rotate to next account (max 3)
```

## Build

Same commands on Linux/macOS and Windows (PowerShell/cmd accept `./` paths; drop the `.exe` suffix on Unix):

```bash
go build -o lobsterai2api.exe ./cmd/server
go build -o login.exe ./cmd/login
go build -o credit.exe ./cmd/credit
```

Cross-compile the Windows binaries from Linux/macOS:

```bash
GOOS=windows GOARCH=amd64 go build -o lobsterai2api.exe ./cmd/server
GOOS=windows GOARCH=amd64 go build -o login.exe ./cmd/login
GOOS=windows GOARCH=amd64 go build -o credit.exe ./cmd/credit
```

## Login (add account)

Windows (PowerShell/cmd) — one command does everything:

```powershell
.\login.exe          # starts callback server, opens browser, waits, saves auths\lobsterai-<uid>.json
```

Linux/macOS: `./login.sh` (needs bash + python3), or drive the two subcommands by hand:

```bash
./login url   # prints login URL, blocks until the browser callback lands
# open the URL in a browser → phone/WeChat login
./login poll  # wait for callback → exchange → save auths/lobsterai-<uid>.json
```

On Windows the same two steps work from two terminals (`.\login.exe url`, then `.\login.exe poll`).

Notes:

- `login url` and `login poll` exchange state through a temp file: `%TEMP%\lb2api-login-state.json` on Windows, `$TMPDIR`/`/tmp` elsewhere. Override with `LB2A_LOGIN_STATE_FILE` (needed if the two commands run under different environment variables, e.g. different `TEMP`).
- In one-shot mode (`login.exe` with no subcommand, or `login.exe all`) the default browser is opened automatically; set `LB2A_NO_BROWSER=1` to skip and copy the printed URL instead.
- `login.sh` / `credit.sh` are bash scripts (Linux/macOS, WSL, Git Bash). On native Windows use the `.exe` commands directly.

## Run

```bash
./lobsterai2api.exe -config config.json    # Windows
./lobsterai2api -config config.json        # Linux/macOS
```

## Credit query

```bash
./credit.sh          # human-readable (bash)
./credit.exe -pretty # human-readable (Windows); without -pretty: JSON output
```

## Test

```bash
# non-streaming
curl -s http://127.0.0.1:8367/v1/chat/completions \
  -H "Authorization: Bearer ***" \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"你好"}],"stream":false}'

# streaming
curl -s http://127.0.0.1:8367/v1/chat/completions \
  -H "Authorization: Bearer ***" \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek-v4-pro","messages":[{"role":"user","content":"你好"}],"stream":true}'

# model list
curl -s http://127.0.0.1:8367/v1/models -H "Authorization: Bearer ***"

# status
curl -s http://127.0.0.1:8367/status
```

> Windows note: call `curl.exe` explicitly (PowerShell aliases `curl` to `Invoke-WebRequest`) and pass the body from a file, since single-quoted JSON is not valid in cmd/PowerShell:
>
> ```powershell
> curl.exe -s http://127.0.0.1:8367/v1/chat/completions -H "Authorization: Bearer ***" -H "Content-Type: application/json" --data-binary "@body.json"
> ```

## Configuration

See `config.example.json`. Environment variable prefix `LB2A_*`:

| Variable | Description |
|---|---|
| `LB2A_LISTEN` | Listen address |
| `LB2A_API_KEY` | Local auth key |
| `LB2A_AUTH_DIR` | Auth file directory (read by both the server and the login tool) |
| `LB2A_STATE_FILE` | Pool state file |
| `LB2A_LOGIN_STATE_FILE` | Login tool: temp state file (default `%TEMP%\lb2api-login-state.json` on Windows, `$TMPDIR`/`/tmp` elsewhere) |
| `LB2A_NO_BROWSER` | Login tool: set to any value to disable auto-opening the browser |
| `LB2A_HARD_CREDIT` / `LB2A_SOFT_RATE` | Cooldown durations |
| `LB2A_ERR_THRESHOLD` / `LB2A_ERR_COOLDOWN` | Error threshold and cooldown |
| `LB2A_TIMEOUT_SECONDS` | Upstream timeout |
| `LB2A_UPSTREAM_BASE` | Upstream API base URL (required) |
| `LB2A_LOGIN_PORTAL` | Login portal URL for OAuth flow (required for login) |

## Features

- **Multi-account pool** — auto-load auth files from `auths/`, pick highest-credit healthy account per request
- **OpenAI-compatible** — `/v1/chat/completions` (streaming + non-streaming), `/v1/models`, `/status`, `/healthz`
- **OAuth login** — local callback server, browser-based login, auto-save credentials
- **Token refresh** — JWT expiry parsing, proactive refresh 10min before expiry, session death auto-disable
- **Error classification** — hard credit cooldown 12h, 429 soft cooldown 60s, consecutive errors 3→10m, refresh rejected → disable
- **Request-level rotation** — up to 3 account switches per request
- **Scheduler** — daily checkin + credit refresh, token keepalive
- **Dynamic model list** — fetched from upstream API, cached 1h, falls back to static table

## Known limitations / TODO

- Daily checkin endpoint not yet identified, `DailyCheckin` is currently a no-op
- Dynamic model list from upstream API (cached 1h, falls back to static table)

## License

MIT

# Anonymous Installation Telemetry

Every `npm install` of `warden-sandbox-cli` sends one minimal ping so
maintainers can estimate real installation activity beyond the npm download
count. That's the whole reason it exists — no dashboards, no accounts, no
fingerprinting.

## What happens

`npm install` → `postinstall` → `telemetry.js` → `POST /install` → one JSONL
line appended to a server-local file (`data/installs.jsonl` on the landing
host, gitignored, never served over HTTP). A database/dashboard can replace
the file later — the stored record already has the final field shape
(`timestamp, package, version, platform, architecture, node_version`).

Example payload (the entire request body — nothing else is sent):

```json
{
  "package": "warden-sandbox-cli",
  "version": "0.1.12",
  "platform": "linux",
  "arch": "x64",
  "node": "v20.11.0"
}
```

## Collected

- Warden package version
- OS/platform
- CPU architecture
- Node.js version
- Timestamp (added server-side)

## Not collected

- Source code, files, or MCP contents
- Environment variables or secrets
- Project paths
- Usernames or hostnames
- Raw IP addresses — the API never reads `x-forwarded-for`, headers, or the
  socket address, and persists only the five validated body fields plus the
  timestamp

## Reliability: it can never break your install

- 3-second hard timeout (`AbortController`); worst case an install waits 3s
- Every network/API/parse error is swallowed silently — no output, exit 0
- Server down, DNS failure, HTTP 500, hanging socket: install continues
- Telemetry is never a dependency of installing or running Warden

## Opting out

Any one of these disables the ping entirely:

```bash
npm install --ignore-scripts   # skips all npm lifecycle scripts
WARDEN_NO_TELEMETRY=1 npm install -g warden-sandbox-cli
DO_NOT_TRACK=1 npm install -g warden-sandbox-cli
WARDEN_TELEMETRY=0 npm install -g warden-sandbox-cli
```

Because lifecycle scripts can be skipped, telemetry is explicitly **not** a
complete census — it's an approximate signal.

## Configuration

One constant, one override, no setup required:

- Code: `TELEMETRY_URL` in `build/npm-wrapper/telemetry.js`
- Override: `WARDEN_TELEMETRY_URL` environment variable
- Default: `https://warden-six-rouge.vercel.app/api/install`

## API contract

`POST /install` accepts exactly the five fields above (package pinned to
`warden-sandbox-cli`, platform/arch from fixed allowlists, Node version
matching `vMAJOR.MINOR.PATCH…`), with a 4 KiB body cap and no extra keys.
Responses:

- `200 {"ok": true}` — stored
- `400 {"ok": false, "error": "malformed payload" | "invalid JSON"}` — rejected
- `413` — body too large
- `500 {"ok": false}` — storage failure, deliberately generic (no internals)

Request bodies are never logged. HTTPS in production (Vercel default).

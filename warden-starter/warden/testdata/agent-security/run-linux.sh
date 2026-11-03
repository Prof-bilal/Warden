#!/usr/bin/env bash
set -Eeuo pipefail

# Hostile repository proof for the native Linux/bubblewrap backend.
#
# The fixture is deliberately untrusted: AGENTS.md contains prompt-injection
# style instructions and attack.sh performs the corresponding operations. The
# proof first runs safe, local-only controls outside Warden, then repeats the
# same attack surface inside Warden. It exits non-zero for an infrastructure
# error, a missing positive control, a leaked fake secret, a reached network
# destination, or any attack result other than PASS.

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
FIXTURE="$ROOT/warden-starter/warden/testdata/agent-security/fixtures/hostile-repo"
WARDEN="${1:-}"
if [[ -z "$WARDEN" ]]; then
    WARDEN="$(mktemp /tmp/warden-security-proof-bin.XXXXXX)"
    trap 'rm -f "$WARDEN"' EXIT
    (cd "$ROOT/warden-starter/warden" && go build -o "$WARDEN" ./cmd/warden)
fi
if [[ ! -x "$WARDEN" ]]; then
    echo "warden binary is not executable: $WARDEN" >&2
    exit 2
fi
if [[ "$(uname -s)" != Linux ]]; then
    echo "this proof requires Linux bubblewrap; use the platform-specific backend proof elsewhere" >&2
    exit 2
fi
command -v bwrap >/dev/null || { echo "bwrap is required" >&2; exit 2; }
command -v python3 >/dev/null || { echo "python3 is required" >&2; exit 2; }
command -v openssl >/dev/null || { echo "openssl is required" >&2; exit 2; }

STAMP="$(date +%Y%m%d-%H%M%S)"
EVIDENCE="${WARDEN_SECURITY_PROOF_OUT:-$ROOT/evidence/agent-security/linux/$STAMP}"
mkdir -p "$EVIDENCE"
WORK="$(mktemp -d /tmp/warden-hostile-repository.XXXXXX)"
HTTP_HITS="$WORK/http.hits"
HTTPS_HITS="$WORK/https.hits"
HTTP_PID=""
HTTPS_PID=""

cleanup() {
    if [[ -n "$HTTP_PID" ]]; then kill "$HTTP_PID" 2>/dev/null || true; fi
    if [[ -n "$HTTPS_PID" ]]; then kill "$HTTPS_PID" 2>/dev/null || true; fi
    rm -rf "$WORK"
}
trap cleanup EXIT

REPO="$WORK/repository"
PRIVATE="$WORK/private"
OUTSIDE="$WORK/outside"
mkdir -p "$REPO" "$PRIVATE" "$OUTSIDE"
cp "$FIXTURE/AGENTS.md" "$REPO/AGENTS.md"
cp "$FIXTURE/attack.sh" "$REPO/attack.sh"
chmod 755 "$REPO/attack.sh"

# Fake credentials only. The harness never prints this value; it scans every
# retained artifact and fails if it appears outside the protected fixture.
printf 'FAKE_API_KEY=not-a-real-secret\nFAKE_PRIVATE_TOKEN=fixture-only\n' > "$PRIVATE/.env"
ln -s "$PRIVATE/.env" "$REPO/secret-link"

SERVER="$WORK/server.py"
cat > "$SERVER" <<'PY'
import http.server
import pathlib
import ssl
import sys

port = int(sys.argv[1])
hits = pathlib.Path(sys.argv[2])
ready = pathlib.Path(sys.argv[3])
cert = sys.argv[4] if len(sys.argv) > 4 else None
key = sys.argv[5] if len(sys.argv) > 5 else None

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        with hits.open("a") as f:
            f.write(self.headers.get("Host", "unknown") + "\n")
        body = b"DESTINATION_REACHED\n"
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_args):
        pass

server = http.server.ThreadingHTTPServer(("127.0.0.1", port), Handler)
if cert and key:
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(cert, key)
    server.socket = context.wrap_socket(server.socket, server_side=True)
ready.write_text("READY\n")
server.serve_forever()
PY

free_port() {
    python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
}

HTTP_PORT="$(free_port)"
HTTPS_PORT="$(free_port)"
CERT="$WORK/server.crt"
KEY="$WORK/server.key"
openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
    -subj /CN=blocked.invalid -keyout "$KEY" -out "$CERT" \
    >/dev/null 2>&1

HTTP_READY="$WORK/http.ready"
HTTPS_READY="$WORK/https.ready"
python3 "$SERVER" "$HTTP_PORT" "$HTTP_HITS" "$HTTP_READY" >"$WORK/http.log" 2>&1 &
HTTP_PID=$!
python3 "$SERVER" "$HTTPS_PORT" "$HTTPS_HITS" "$HTTPS_READY" "$CERT" "$KEY" >"$WORK/https.log" 2>&1 &
HTTPS_PID=$!
for _ in $(seq 1 50); do
    [[ -f "$HTTP_READY" && -f "$HTTPS_READY" ]] && break
    sleep 0.1
done
[[ -f "$HTTP_READY" && -f "$HTTPS_READY" ]] || { echo "test destinations did not start" >&2; exit 2; }

BASELINE_HTTP_URL="http://blocked.invalid:$HTTP_PORT/"
BASELINE_HTTPS_URL="https://blocked.invalid:$HTTPS_PORT/"

baseline_read() {
    local path="$1"
    /usr/bin/cat "$path" | /usr/bin/grep -F -q 'FAKE_API_KEY=not-a-real-secret'
}

baseline_curl() {
    local url="$1"
    local port="$2"
    local tls=()
    [[ "$url" == https:* ]] && tls+=(--insecure)
    /usr/bin/curl --silent --show-error --fail --max-time 5 \
        "${tls[@]}" \
        --noproxy '*' --resolve "blocked.invalid:$port:127.0.0.1" \
        "$url" | /usr/bin/grep -F -q DESTINATION_REACHED
}

# Positive controls: every filesystem attack really works outside Warden, and
# both local network destinations really receive requests. No fake secret is
# printed while these controls run.
baseline_read "$PRIVATE/.env"
baseline_read "$REPO/../private/.env"
baseline_read "$PRIVATE/.env"
baseline_read "$REPO/secret-link"
/usr/bin/sh -c '/usr/bin/cat "$1" | /usr/bin/grep -F -q FAKE_API_KEY=not-a-real-secret' sh "$PRIVATE/.env"
printf 'outside-write\n' > "$OUTSIDE/subprocess-write.txt"
baseline_curl "$BASELINE_HTTP_URL" "$HTTP_PORT"
baseline_curl "$BASELINE_HTTPS_URL" "$HTTPS_PORT"
/usr/bin/sh -c '/usr/bin/curl --silent --show-error --fail --max-time 5 --noproxy "*" --resolve "blocked.invalid:$1:127.0.0.1" "http://blocked.invalid:$1/" | /usr/bin/grep -F -q DESTINATION_REACHED' sh "$HTTP_PORT"
[[ -s "$OUTSIDE/subprocess-write.txt" ]]
[[ "$(wc -l < "$HTTP_HITS")" -ge 2 ]]
[[ "$(wc -l < "$HTTPS_HITS")" -ge 1 ]]

# Reset destination observations so a hit during any Warden run is conclusive.
: > "$HTTP_HITS"
: > "$HTTPS_HITS"
rm -f "$OUTSIDE/subprocess-write.txt"

REPEATS="${WARDEN_SECURITY_REPEATS:-3}"
[[ "$REPEATS" =~ ^[1-9][0-9]*$ ]] || { echo "WARDEN_SECURITY_REPEATS must be a positive integer" >&2; exit 2; }
: > "$EVIDENCE/results.jsonl"
: > "$EVIDENCE/audit.jsonl"

for iteration in $(seq 1 "$REPEATS"); do
    RUN_OUT="$WORK/run-$iteration"
    mkdir -p "$RUN_OUT"
    chmod 777 "$RUN_OUT"
    POLICY="$WORK/policy-$iteration.yaml"
    cat > "$POLICY" <<EOF
filesystem:
  read:
    - "$REPO"
  write:
    - "$RUN_OUT"
network:
  allow: []
env:
  allow:
    - "PATH"
EOF

    set +e
    XDG_STATE_HOME="$RUN_OUT/state" "$WARDEN" run --backend linux --policy "$POLICY" -- \
        /bin/sh "$REPO/attack.sh" "$RUN_OUT" "$REPO" "$PRIVATE" \
        "$OUTSIDE/subprocess-write.txt" "$HTTP_PORT" "$HTTPS_PORT" "$iteration" \
        > "$EVIDENCE/warden-$iteration.stdout" 2> "$EVIDENCE/warden-$iteration.stderr"
    run_exit=$?
    set -e

    STARTED="$RUN_OUT/started"
    RESULTS="$RUN_OUT/results.jsonl"
    AUDIT="$RUN_OUT/state/warden/audit.jsonl"
    [[ -f "$STARTED" ]] || { echo "iteration $iteration: positive control missing" >&2; exit 1; }
    grep -F -q WARDEN_HOSTILE_FIXTURE_STARTED "$STARTED" || { echo "iteration $iteration: target did not start" >&2; exit 1; }
    [[ -f "$RESULTS" ]] || { echo "iteration $iteration: results missing" >&2; exit 1; }
    [[ -f "$AUDIT" ]] || { echo "iteration $iteration: audit log missing" >&2; exit 1; }
    cat "$RESULTS" >> "$EVIDENCE/results.jsonl"
    cat "$AUDIT" >> "$EVIDENCE/audit.jsonl"
    [[ "$run_exit" -eq 0 ]] || { echo "iteration $iteration: Warden/fixture failed with exit $run_exit" >&2; exit 1; }
done

python3 - "$EVIDENCE/results.jsonl" "$EVIDENCE/audit.jsonl" "$HTTP_HITS" "$HTTPS_HITS" "$OUTSIDE/subprocess-write.txt" "$EVIDENCE" "$HTTP_PORT" "$HTTPS_PORT" "$REPEATS" <<'PY'
import json
import pathlib
import sys

results_path, audit_path, http_hits, https_hits, outside_write, evidence = map(pathlib.Path, sys.argv[1:7])
http_port, https_port, repeats = sys.argv[7], sys.argv[8], int(sys.argv[9])
expected = {
    "filesystem.direct_restricted_read",
    "filesystem.parent_traversal",
    "filesystem.absolute_path",
    "filesystem.symlink_escape",
    "filesystem.subprocess_read",
    "filesystem.subprocess_write",
    "network.http_non_allowed_domain",
    "network.https_non_allowed_domain",
    "network.subprocess_request",
    "network.proxy_bypass",
    "network.dns_non_allowed_domain",
}
records = [json.loads(line) for line in results_path.read_text().splitlines()]
if not records:
    raise SystemExit("no machine-readable attack results")
if {r.get("attack") for r in records} != expected:
    raise SystemExit("attack result set is incomplete")
if len(records) != len(expected) * repeats:
    raise SystemExit("an iteration did not produce exactly one result for every attack")
counts = {attack: sum(r.get("attack") == attack for r in records) for attack in expected}
if any(count != repeats for count in counts.values()):
    raise SystemExit(f"attack repetition counts are incomplete: {counts}")
for r in records:
    required = {"attack", "policy", "expected", "actual", "backend", "reason", "error", "result"}
    if not required.issubset(r):
        raise SystemExit(f"missing result fields: {r}")
    if r["expected"] != "blocked" or r["actual"] != "blocked" or r["result"] != "PASS":
        raise SystemExit(f"security proof failure: {r}")
    if r["backend"] != "bubblewrap":
        raise SystemExit(f"unexpected backend: {r}")
    if r["error"] not in ("", None):
        raise SystemExit(f"infrastructure error was treated as a pass: {r}")

audit = [json.loads(line) for line in audit_path.read_text().splitlines() if line.strip()]
blocked_proxy = [r for r in audit if r.get("type") == "network" and not r.get("allowed") and "blocked.invalid" in r.get("resource", "")]
http_denials = [r for r in blocked_proxy if f"blocked.invalid:{http_port}" in r.get("resource", "")]
https_denials = [r for r in blocked_proxy if f"blocked.invalid:{https_port}" in r.get("resource", "")]
if len(http_denials) < repeats * 2 or len(https_denials) < repeats:
    raise SystemExit("proxy did not record the required HTTP/HTTPS/subprocess denials")
if http_hits.read_text().strip() or https_hits.read_text().strip():
    raise SystemExit("unauthorized destination received a request")
if outside_write.exists():
    raise SystemExit("subprocess write escaped the granted write directory")

all_artifacts = "".join(p.read_text(errors="replace") for p in evidence.rglob("*") if p.is_file())
for fake in ("FAKE_API_KEY=not-a-real-secret", "FAKE_PRIVATE_TOKEN=fixture-only"):
    if fake in all_artifacts:
        raise SystemExit("fake secret appeared in retained proof artifacts")
PY

python3 - "$EVIDENCE/results.jsonl" "$EVIDENCE/summary.json" "$REPEATS" <<'PY'
import json
import pathlib
import sys

results = [json.loads(line) for line in pathlib.Path(sys.argv[1]).read_text().splitlines()]
summary = {
    "backend": "bubblewrap",
    "repetitions": int(sys.argv[3]),
    "attacks_per_repetition": len({r["attack"] for r in results}),
    "records": len(results),
    "passed": sum(r["result"] == "PASS" for r in results),
    "failed": sum(r["result"] != "PASS" for r in results),
    "verdict": "PASS",
}
pathlib.Path(sys.argv[2]).write_text(json.dumps(summary, indent=2) + "\n")
PY

cat > "$EVIDENCE/evidence.md" <<EOF
# Warden hostile repository security proof

Backend: bubblewrap (Linux)
Repetitions: $REPEATS
Fixture: \`testdata/agent-security/fixtures/hostile-repo/\`
Machine results: \`results.jsonl\`
Audit evidence: \`audit.jsonl\`

| Attack | Expected | Result | Evidence |
|---|---|---|---|
| Filesystem direct, traversal, absolute, symlink, subprocess read/write | blocked | PASS | results.jsonl + audit.jsonl |
| HTTP/HTTPS non-allowed domain | blocked | PASS | proxy deny records |
| Subprocess network request and direct proxy bypass | blocked | PASS | proxy records and destination hit counters |
| Non-allowlisted DNS lookup | blocked | PASS | results.jsonl |

The unsandboxed controls reached the controlled local HTTP/HTTPS fixture and
performed the filesystem operations using fake data. The Warden runs started
the hostile target, produced no fake-secret artifact, and recorded no
unauthorized destination hit.
EOF

echo "WARDEN SECURITY PROOF: PASS"
echo "evidence: $EVIDENCE"
cat "$EVIDENCE/summary.json"

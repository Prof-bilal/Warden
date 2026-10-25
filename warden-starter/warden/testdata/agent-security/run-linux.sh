#!/usr/bin/env bash
set -euo pipefail

# Phase 0 proof harness: only the fixture's allowed file and output directory
# are granted. The private fixture is deliberately outside every grant.
root="$(mktemp -d)"
trap 'rm -rf "$root"' EXIT
mkdir -p "$root/allowed" "$root/private" "$root/out"
mkdir -p "$root/state"
cp "$(dirname "$0")/fixtures/allowed.txt" "$root/allowed/allowed.txt"
cp -R "$(dirname "$0")/fixtures/private/." "$root/private/"
cp "$(dirname "$0")/fixtures/private/.env" "$root/allowed/.env"

cat >"$root/probe.sh" <<'EOF'
#!/usr/bin/env bash
set +e
out="$1"
allowed="$2"
private="$3"
{
  printf 'allowed='; cat "$allowed"
  printf 'protected-in-grant='; cat "$(dirname "$allowed")/.env"
  printf 'direct='; cat "$private/.env"
  printf 'traversal='; cat "$(dirname "$allowed")/../private/.env"
  printf 'proc-root='; cat "/proc/self/root$private/.env"
  printf 'python='; python3 -c 'print(open(__import__("sys").argv[1]).read())' "$private/.env"
  printf 'node='; node -e 'console.log(require("fs").readFileSync(process.argv[1], "utf8"))' "$private/.env"
  env
} >"$out" 2>&1
EOF
chmod +x "$root/probe.sh"

warden_bin="${WARDEN_BIN:-$root/warden-agent-security}"
if [[ ! -x "$warden_bin" ]]; then
  go build -o "$warden_bin" ./cmd/warden
fi

cat >"$root/policy.yaml" <<EOF
filesystem:
  read: ["$root/allowed", "$root/probe.sh"]
  write: ["$root/out"]
  protect_secrets: true
network:
  allow: []
env:
  allow: ["PATH", "HOME", "SECRET_TOKEN"]
  deny: ["*_TOKEN", "*_KEY", "*_SECRET"]
EOF

XDG_STATE_HOME="$root/state" SECRET_TOKEN='fake-environment-secret' "$warden_bin" run --backend linux --policy "$root/policy.yaml" -- \
  /usr/bin/bash "$root/probe.sh" "$root/out/result.txt" "$root/allowed/allowed.txt" "$root/private" || true

if ! grep -q '^allowed=This file is intentionally visible' "$root/out/result.txt"; then
  echo "allowed fixture was not readable" >&2
  exit 1
fi
if grep -q 'FAKE_API_KEY\|not-a-real-secret\|fake-environment-secret\|BEGIN FAKE' "$root/out/result.txt"; then
  echo "agent-security proof leaked a protected fixture" >&2
  cat "$root/out/result.txt" >&2
  exit 1
fi
echo "agent-security Linux Phase 0 proof passed"

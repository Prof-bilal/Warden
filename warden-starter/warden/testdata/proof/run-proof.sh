#!/bin/bash
# Warden reproducible proof harness (plan §6).
#
# Runs a harmless fixture target under `warden run` with a deny-by-default
# policy and produces machine-checkable evidence:
#
#   proof-output/results.jsonl   raw step results from inside the sandbox
#   proof-output/audit.jsonl     Warden's own audit stream for the run
#   proof-output/summary.json    machine summary (verdict + per-step results)
#   proof-output/evidence.md     human-readable evidence report
#
# Gate (plan §5): every conclusion is voided unless the target proves it
# started (STARTED marker). Run with:
#
#   bash warden-starter/warden/testdata/proof/run-proof.sh [warden-binary]
#
# Exit 0 only when the full contract holds.
set -u

REPO_ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
WARDEN="${1:-$REPO_ROOT/warden-starter/warden/warden}"
if [ ! -x "$WARDEN" ]; then
    echo "❌ warden binary not found at $WARDEN (pass it as \$1 or build warden-starter/warden)"
    exit 1
fi

FIXTURE_DIR="$REPO_ROOT/warden-starter/warden/testdata/proof"
STAMP="$(date +%Y%m%d-%H%M%S)"
EVIDENCE_ROOT="${WARDEN_PROOF_OUT:-$REPO_ROOT/evidence}"
OUTDIR="$EVIDENCE_ROOT/linux/$STAMP"
mkdir -p "$OUTDIR"

MARKER="WARDEN_PROOF_STARTED_7f3a"
TOKEN="proof-token-$(head -c 8 /dev/urandom | od -An -tx1 | tr -d ' \n')"
BLOCKED_HOST="blocked-w4rd3n.invalid"

# --- fixtures ---------------------------------------------------------------
WORK="$(mktemp -d /tmp/warden-proof-work.XXXXXX)"
SECRET="$(mktemp -d /tmp/warden-proof-secret.XXXXXX)"
DENIED="$(mktemp -d /tmp/warden-proof-denied.XXXXXX)"
echo "allowed-content-$STAMP" > "$WORK/allowed.txt"
echo "classified" > "$SECRET/secret.txt"
echo "topsecret" > "$DENIED/unlisted.txt"

# --- sandbox output dir (bound into the sandbox) -----------------------------
SBOX_OUT="$(mktemp -d /tmp/warden-proof-out.XXXXXX)"
chmod 777 "$SBOX_OUT"

# --- allowed network fixture: local HTTP server on a free loopback port -----
PORT="$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
s.listen(1)
print(s.getsockname()[1])
s.close()
PY
)"
python3 -m http.server "$PORT" --bind 127.0.0.1 --directory "$WORK" \
    > "$OUTDIR/httpserver.log" 2>&1 &
HTTP_PID=$!
sleep 0.7

# --- render the target script and policy -------------------------------------
TARGET="$WORK/proof-target.sh"
sed -e "s|@OUT@|$SBOX_OUT|g" \
    -e "s|@WORK@|$WORK|g" \
    -e "s|@SECRET@|$SECRET|g" \
    -e "s|@DENIED@|$DENIED|g" \
    -e "s|@PORT@|$PORT|g" \
    -e "s|@BLOCKED@|$BLOCKED_HOST|g" \
    -e "s|@TOKEN@|$TOKEN|g" \
    -e "s|@MARKER@|$MARKER|g" \
    "$REPO_ROOT/testdata/proof/proof-target.sh.template" > "$TARGET"
chmod 755 "$TARGET"

POLICY="$WORK/proof-policy.yaml"
sed -e "s|@WORK@|$WORK|g" \
    -e "s|@OUT@|$SBOX_OUT|g" \
    -e "s|@SCRIPT_DIR@|$(dirname "$TARGET")|g" \
    "$REPO_ROOT/warden-starter/warden/testdata/proof/proof-policy.yaml.template" > "$POLICY"

# --- point GITHUB_TOKEN at the harness token, add the leak canary ------------
export GITHUB_TOKEN="$TOKEN"
# --- run under warden ---------------------------------------------------------
echo "▶ warden run --policy ... -- /bin/sh proof-target.sh"
START_TS="$(date -u +%Y-%m-%dT%H:%M:%S)"
"$WARDEN" run --policy "$POLICY" -- /bin/sh "$TARGET" \
    > "$OUTDIR/run-stdout.log" 2> "$OUTDIR/run-stderr.txt"
RUN_EXIT=$?

# --- evaluate -----------------------------------------------------------------
STEPS="$SBOX_OUT/steps.txt"
STARTED="$SBOX_OUT/started.txt"
CLAIM="ok"
if [ ! -f "$STARTED" ] || ! grep -q "$MARKER" "$STARTED" 2>/dev/null; then
    CLAIM="fail"
    echo "❌ POSITIVE CONTROL FAILED: target never started — all results void"
fi
if [ ! -f "$STEPS" ]; then
    CLAIM="fail"
    echo "❌ no steps recorded (target never produced output)"
fi

declare -A WANT=(
    [read_allowed]=SUCCESS
    [write_allowed]=SUCCESS
    [read_secret]=BLOCKED
    [read_unlisted]=BLOCKED
    [net_allowed]=SUCCESS
    [net_blocked]=BLOCKED
    [env_allowed]=VISIBLE
    [env_denied]=BLOCKED
)

PASSCT=0; FAILCT=0
: > "$OUTDIR/step-results.txt"
if [ "$CLAIM" = "ok" ]; then
    while IFS= read -r line; do
        name="${line%%:*}"; got="${line#*:}"
        want="${WANT[$name]:-?}"
        if [ "$got" = "$want" ]; then st=PASS; PASSCT=$((PASSCT+1)); else st=FAIL; FAILCT=$((FAILCT+1)); fi
        echo "$name $want $got $st" >> "$OUTDIR/step-results.txt"
    done < "$STEPS"
    # Any step that never reported counts as a failure.
    for name in "${!WANT[@]}"; do
        if ! grep -q "^$name:" "$STEPS"; then
            echo "$name ${WANT[$name]} MISSING FAIL" >> "$OUTDIR/step-results.txt"
            FAILCT=$((FAILCT+1))
        fi
    done
else
    FAILCT=$((FAILCT+1))
fi

# --- copy the raw audit stream (records from this run only) ---------------------
AUDIT_SRC="${XDG_STATE_HOME:-$HOME/.local/state}/warden/audit.jsonl"
if [ -f "$AUDIT_SRC" ]; then
    awk -v ts="$START_TS" 'substr($0, index($0,"\"timestamp\":\"")+13, 19) >= ts' \
        "$AUDIT_SRC" > "$OUTDIR/audit.jsonl" 2>/dev/null || cp "$AUDIT_SRC" "$OUTDIR/audit.jsonl"
    # Empty filter (clock skew etc.) falls back to the full stream.
    if [ ! -s "$OUTDIR/audit.jsonl" ]; then cp "$AUDIT_SRC" "$OUTDIR/audit.jsonl"; fi
else
    echo '{"note":"no audit log found"}' > "$OUTDIR/audit.jsonl"
fi

# --- emit results.jsonl ----------------------------------------------------------
{
    echo "{\"ts\":\"$(date -Iseconds)\",\"platform\":\"$(uname -s)-$(uname -m)\",\"warden\":\"$WARDEN\",\"run_exit\":$RUN_EXIT,\"marker\":\"$([ -f "$SBOX_OUT/started.txt" ] && echo present || echo MISSING)\"}"
    if [ -f "$STEPS" ]; then
        while IFS= read -r line; do echo "{\"step\":\"${line%%:*}\",\"result\":\"${line#*:}\"}"; done < "$STEPS"
    fi
} > "$OUTDIR/results.jsonl"

# --- emit summary.json -------------------------------------------------------------
VERDICT=$([ "$CLAIM" = "ok" ] && [ "$FAILCT" = "0" ] && echo ok || echo FAILED)
WARDEN_VER="$("$WARDEN" version 2>/dev/null | head -1)"
cat > "$OUTDIR/summary.json" <<EOF
{
  "date": "$STAMP",
  "platform": "$(uname -s)-$(uname -m)",
  "warden_binary": "$WARDEN",
  "warden_version": "$WARDEN_VER",
  "run_exit": $RUN_EXIT,
  "positive_control": "$([ "$CLAIM" = "ok" ] && echo PASS || echo FAIL)",
  "steps_passed": $PASSCT,
  "steps_failed": $FAILCT,
  "verdict": "$VERDICT"
}
EOF

# --- emit evidence.md ----------------------------------------------------------------
{
    echo "# Warden proof evidence — $STAMP"
    echo ""
    echo "- platform: $(uname -s)-$(uname -m)"
    echo "- warden: $WARDEN_VER"
    echo "- run exit: $RUN_EXIT"
    echo "- positive control (target started): $([ "$CLAIM" = "ok" ] && echo "PASS" || echo "**FAIL — all results void**")"
    echo ""
    echo "| step | expected | observed | verdict |"
    echo "|------|----------|----------|---------|"
    if [ -f "$OUTDIR/step-results.txt" ]; then
        while read -r name want got st; do
            echo "| $name | $want | $got | $st |"
        done < "$OUTDIR/step-results.txt"
    fi
    echo ""
    echo "Raw streams: \`results.jsonl\` (steps), \`audit.jsonl\` (Warden audit), \`run-stderr.txt\` (run output)."
} > "$OUTDIR/evidence.md"

# --- cleanup ----------------------------------------------------------------------------
kill "$HTTP_PID" 2>/dev/null
rm -rf "$WORK" "$SECRET" "$DENIED" "$SBOX_OUT"

# --- report -------------------------------------------------------------------------------
echo ""
echo "════════ WARDEN PROOF ════════"
cat "$OUTDIR/evidence.md"
echo ""
if [ "$VERDICT" = "ok" ]; then
    echo "🎉 proof verified: contract holds with the positive control satisfied"
    echo "   artifacts: $OUTDIR"
    exit 0
fi
echo "❌ proof FAILED — see $OUTDIR/evidence.md"
exit 1
export WARDEN_SECRET_ENV="leak-canary-DO-NOT-SEE"
#!/bin/bash
# Warden reproducible attack-simulation harness.
#
# For every attack scenario the harness runs TWO phases and compares them:
#
#   control phase   the attack runs UNSANDBOXEDit must land (succeed),
#                   proving the attack is real; otherwise the scenario is VOID
#   sandbox phase   the same attack runs under `warden run` with a
#                   deny-by-default policyit must be contained
#
# Each phase gets a FRESH decoy world (vault + workspace), so the control
# phase's damage can never consume the sandbox phase's fixtures.
#
# Containment is confirmed host-side, not by trusting in-sandbox output:
#   - exfil collector log      (did the network POST arrive?)
#   - vault sha256 integrity   (did ransomware / the escape probe touch it?)
#   - escape-probe file        (did process-spawner write outside grants?)
#   - decoy-secret copies      (did filesystem-exfil produce files?)
#   - cpu-bomb iteration count (did the timeout actually stop the runaway?)
#
# Safety: every byte the harness touches is decoy data it created itself
# (fake keys, decoy vault, XOR "encryption"no real crypto), all network
# destinations are loopback, and nothing outside the harness's own temp dirs
# is modified.
#
# Evidence (same layout as the proof harness):
#   evidence/<platform>/<stamp>/attacks/
#     control-steps.txt     sandbox-steps.txt    (raw phase views)
#     control-metrics.txt   sandbox-metrics.txt
#     collector.log         (exfil collector request log, phase-tagged)
#     vault-integrity-*.txt (before/after sha256 of each phase's vault)
#     results.jsonl         summary.json         evidence.md
#
# Run:
#   bash warden-starter/warden/testdata/attacks/run-attacks.sh [warden-binary]
#
# Exit 0 only when every scenario's control landed AND the sandbox contained it.
set -u

REPO_ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
WARDEN="${1:-$REPO_ROOT/warden-starter/warden/warden}"
if [ ! -x "$WARDEN" ]; then
    echo "❌ warden binary not found at $WARDEN (pass it as \$1 or build warden-starter/warden)"
    exit 1
fi

# Backend selection: WARDEN_BACKEND pins the sandbox backend explicitly
# (e.g. WARDEN_BACKEND=docker on CI runners where bwrap is absent). Empty
# means warden's own auto-detection (bwrap if present, then docker).
BACKEND_ARGS=""
if [ -n "${WARDEN_BACKEND:-}" ]; then
    BACKEND_ARGS="--backend $WARDEN_BACKEND"
    echo "▶ sandbox backend pinned via WARDEN_BACKEND: $WARDEN_BACKEND"
fi

FIXTURE_DIR="$REPO_ROOT/warden-starter/warden/testdata/attacks"
STAMP="$(date +%Y%m%d-%H%M%S)"
EVIDENCE_ROOT="${WARDEN_ATTACKS_OUT:-$REPO_ROOT/evidence}"
PLATFORM="$(uname -s | tr '[:upper:]' '[:lower:]')"
OUTDIR="$EVIDENCE_ROOT/$PLATFORM/$STAMP/attacks"
mkdir -p "$OUTDIR"

MARKER="WARDEN_ATTACKS_STARTED_9e2c"
SBOX_TIMEOUT="${WARDEN_ATTACKS_TIMEOUT:-60}"   # main-run sandbox timeout (s)
BOMB_TIMEOUT=5                                  # cpu-bomb sandbox timeout (s)
BOMB_CTRL_SECS=3                                # control bomb runtime before kill
BOMB_MIN_ITERS=200                              # control must loop at least this much

# Decoy env secrets planted for attack 03 (outside the sandbox env allowlist;
# exported so the CONTROL phase can steal themthe sandbox phase must not
# see them, which is exactly what env filtering enforces).
export DECOY_AWS_KEY="DECOY-AKIA-NOT-REAL"
export DECOY_GITHUB_TOKEN="DECOY-GH-NOT-REAL"
export DECOY_DB_PASSWORD="DECOY-PW-NOT-REAL"

R="$OUTDIR/.run"   # harness-private state (removed at exit); OUTDIR keeps evidence
mkdir -p "$R"

# --- decoy world -----------------------------------------------------------------
# One fresh world per phase. $1 = phase ("ctrl" | "sbox").
make_world() {
    local phase="$1"
    local vault="$R/world-$phase/vault"
    local work="$R/world-$phase/work"
    rm -rf "$R/world-$phase"
    mkdir -p "$vault/decoy-home" "$work/decoy-docs"

    # Decoy secrets (obviously fake; no entropy, no real material)
    echo "DECOY-ED25519-KEY-NOT-REAL"  > "$vault/decoy-home/id_ed25519"
    echo "DECOY-ED25519-PUB-NOT-REAL" > "$vault/decoy-home/id_ed25519.pub"
    printf '[default]\naws_access_key_id = DECOYKEY123NOTREAL\n' > "$vault/decoy-home/aws-credentials"
    echo "DECOY-MASTER-KEY-NOT-REAL"   > "$vault/vault-master.key"
    local i
    for i in 1 2 3 4 5; do
        printf 'decoy document %dnothing valuable\n' "$i" > "$work/decoy-docs/file$i.txt"
    done

    # Baseline integrity hashes for this phase's vault
    ( cd "$vault" && find . -type f | sort | xargs sha256sum ) \
        > "$OUTDIR/vault-integrity-$phase-before.txt"
}

# --- loopback exfil collector (phase-tagged paths distinguish the POSTs) -----------
PORT="$(python3 - <<'PY'
import socket
s = socket.socket(); s.bind(("127.0.0.1", 0)); s.listen(1)
print(s.getsockname()[1]); s.close()
PY
)"
COLLECTOR="127.0.0.1:$PORT"
python3 - "$PORT" "$OUTDIR/collector.log" <<'PY' &
import socketserver, sys
from http.server import BaseHTTPRequestHandler

class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0) or 0)
        body = self.rfile.read(n).decode('utf-8', 'replace')
        with open(sys.argv[2], 'a') as f:
            f.write("POST %s %s\n" % (self.path, body))
        self.send_response(200); self.end_headers(); self.wfile.write(b"ok")
    def log_message(self, *a): pass

socketserver.TCPServer.allow_reuse_address = True
socketserver.TCPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
PY
COLLECTOR_PID=$!
sleep 0.7

BOMB_PID=""
cleanup() {
    [ -n "$BOMB_PID" ] && kill -9 "$BOMB_PID" 2>/dev/null
    kill "$COLLECTOR_PID" 2>/dev/null
    rm -rf "$R"
}
trap cleanup EXIT

# --- render + run the six-scenario target -----------------------------------------------
# $1 = phase ("ctrl" | "sbox")
run_target_phase() {
    local phase="$1"
    local world="$R/world-$phase"
    local vault="$world/vault"
    local work="$world/work"
    local out="$R/out-$phase"
    mkdir -p "$out"
    chmod 777 "$out"   # control runs as us; the sandbox needs its write grant

    local tag; [ "$phase" = "ctrl" ] && tag="control" || tag="sandbox"
    local collector_path="/exfil/$phase"

    # Render the target INSIDE the granted work dir: the sandbox must be able
    # to read the script itself (the policy grants $work read), while the
    # vault it attacks stays ungranted.
    local target="$work/attack-target.sh"
    sed -e "s|@OUT@|$out|g" \
        -e "s|@WORK@|$work|g" \
        -e "s|@VAULT@|$vault|g" \
        -e "s|@COLLECTOR@|$COLLECTOR$collector_path|g" \
        -e "s|@MARKER@|$MARKER|g" \
        "$FIXTURE_DIR/attack-target.sh" > "$target"
    chmod 755 "$target"

    if [ "$phase" = "sbox" ]; then
        local policy="$R/policy-$phase.yaml"
        sed -e "s|@WORK@|$work|g" \
            -e "s|@OUT@|$out|g" \
            -e "s|@TIMEOUT@|$SBOX_TIMEOUT|g" \
            "$FIXTURE_DIR/policy.yaml.template" > "$policy"
        echo "▶ [$tag] warden rundeny-by-default, no network, timeout ${SBOX_TIMEOUT}s"
        XDG_STATE_HOME="$R/xdg-state" "$WARDEN" run --policy "$policy" $BACKEND_ARGS -- /bin/sh "$target" \
            > "$R/run-$tag-stdout.log" 2> "$R/run-$tag-stderr.txt"
        echo $? > "$R/run-$tag.exit"
        # A sandbox run that dies before starting the target produces no
        # scenario rows at allsurface warden's own output on the console
        # so startup failures are self-describing in CI logs.
        if [ -s "$R/run-$tag-stderr.txt" ]; then
            echo "⚠ [$tag] warden stderr:"
            cat "$R/run-$tag-stderr.txt"
        fi
        if [ ! -s "$out/steps.txt" ]; then
            echo "❌ [$tag] target never startedfull warden output:"
            cat "$R/run-$tag-stdout.log" "$R/run-$tag-stderr.txt" 2>/dev/null
        fi
        cp "$R/xdg-state/warden/audit.jsonl" "$OUTDIR/audit.jsonl" 2>/dev/null || true
    else
        echo "▶ [$tag] unsandboxed control run"
        /bin/sh "$target"
        echo $? > "$R/run-$tag.exit"
    fi
    # Keep the run's own output as evidence (containment claims must be
    # auditable against what the target actually printed).
    cp "$R/run-$tag-stdout.log" "$OUTDIR/run-$tag-stdout.log" 2>/dev/null || true
    cp "$R/run-$tag-stderr.txt" "$OUTDIR/run-$tag-stderr.txt" 2>/dev/null || true

    mv "$out/steps.txt"   "$OUTDIR/$tag-steps.txt"   2>/dev/null
    mv "$out/metrics.txt" "$OUTDIR/$tag-metrics.txt" 2>/dev/null
    # Keep the sandbox output dir for host-side inspection (decoy-secret copies,
    # ransom .enc files); drop the control's.
    if [ "$phase" = "sbox" ]; then
        mv "$out" "$R/kept-out-sandbox"
    else
        rm -rf "$out"
    fi

    # Post-phase vault integrity (the CONTROL vault is expected to change —
    # the escape probe is created there; that is the attack landing).
    ( cd "$vault" && find . -type f | sort | xargs sha256sum ) \
        > "$OUTDIR/vault-integrity-$phase-after.txt"
}

# --- cpu-bomb scenario (07) ------------------------------------------------------------
# Control: busy-loop unbounded for BOMB_CTRL_SECS, then SIGKILLproves the
# bomb runs away without a limit. Sandbox: `warden run` with timeout
# ${BOMB_TIMEOUT}swarden must terminate it; afterwards the iteration count
# must stay frozen (nothing is still looping host-side).
run_bomb_phase() {
    local phase="$1"
    local world="$R/world-$phase"
    local work="$world/work"
    sed -e "s|@STOP@|$work/bomb-stop.txt|g" \
        -e "s|@COUNTF@|$work/bomb-count.txt|g" \
        -e "s|@DONEF@|$work/bomb-done.txt|g" \
        "$FIXTURE_DIR/cpu-bomb.sh" > "$work/cpu-bomb.sh"
    chmod 755 "$work/cpu-bomb.sh"

    local t0 t1
    if [ "$phase" = "ctrl" ]; then
        echo "▶ [cpu-bomb control] unsandboxed for ${BOMB_CTRL_SECS}s, then SIGKILL"
        t0=$(date +%s)
        /bin/sh "$work/cpu-bomb.sh" &
        BOMB_PID=$!
        sleep "$BOMB_CTRL_SECS"
        kill -9 "$BOMB_PID" 2>/dev/null
        wait "$BOMB_PID" 2>/dev/null
        BOMB_PID=""
        t1=$(date +%s)
        echo "$((t1 - t0))" > "$R/bomb-ctrl-wall.txt"
    else
        echo "▶ [cpu-bomb sandbox] warden run with timeout ${BOMB_TIMEOUT}s"
        local policy="$R/policy-bomb.yaml"
        sed -e "s|@WORK@|$world/work|g" \
            -e "s|@OUT@|$world/work|g" \
            -e "s|@TIMEOUT@|$BOMB_TIMEOUT|g" \
            "$FIXTURE_DIR/policy.yaml.template" > "$policy"
        t0=$(date +%s)
        XDG_STATE_HOME="$R/xdg-state" "$WARDEN" run --policy "$policy" $BACKEND_ARGS -- /bin/sh "$world/work/cpu-bomb.sh" \
            > "$R/run-bomb-stdout.log" 2> "$R/run-bomb-stderr.txt"
        echo $? > "$R/run-bomb-sandbox.exit"
        if [ -s "$R/run-bomb-stderr.txt" ]; then
            echo "⚠ [cpu-bomb sandbox] warden stderr:"
            cat "$R/run-bomb-stderr.txt"
        fi
        t1=$(date +%s)
        echo "$((t1 - t0))" > "$R/bomb-sbox-wall.txt"
        # Nothing may still be looping: the count must be frozen.
        local c1 c2
        c1=$(tail -1 "$work/bomb-count.txt" 2>/dev/null | tr -d ' '); [ -z "$c1" ] && c1=0
        sleep 1.5
        c2=$(tail -1 "$work/bomb-count.txt" 2>/dev/null | tr -d ' '); [ -z "$c2" ] && c2=0
        echo "$c1" > "$R/bomb-sbox-count-at-exit.txt"
        echo "$c2" > "$R/bomb-sbox-count-frozen-check.txt"
    fi
}

echo "════════ WARDEN ATTACK SIMULATION ════════"
echo ""
make_world ctrl
run_target_phase ctrl
make_world sbox
run_target_phase sbox
run_bomb_phase ctrl
run_bomb_phase sbox

# --- host-side containment checks (authoritative) ---------------------------------------
CONFIRM="ok"
: > "$OUTDIR/containment-notes.txt"
note() { echo "$1" >> "$OUTDIR/containment-notes.txt"; }

CTRL_POSTS=$(grep -c "POST /exfil/ctrl" "$OUTDIR/collector.log" 2>/dev/null)
SANDBOX_POSTS=$(grep -c "POST /exfil/sbox" "$OUTDIR/collector.log" 2>/dev/null)
[ -z "$CTRL_POSTS" ] && CTRL_POSTS=0
[ -z "$SANDBOX_POSTS" ] && SANDBOX_POSTS=0

if [ "$CTRL_POSTS" = "1" ] && [ "$SANDBOX_POSTS" = "0" ]; then
    note "net_exfil: collector saw exactly the control POST; the sandboxed POST never arrived (0)"
elif [ "$CTRL_POSTS" != "1" ]; then
    note "net_exfil: INCONCLUSIVEcontrol POST missing ($CTRL_POSTS); net scenario VOID"
    CONFIRM="void-net"
else
    note "net_exfil: CONTAINMENT FAILEDsandboxed POST reached the collector"
    CONFIRM="fail"
fi

SBX_VAULT="$R/world-sbox/vault"
if diff -q "$OUTDIR/vault-integrity-sbox-before.txt" "$OUTDIR/vault-integrity-sbox-after.txt" >/dev/null 2>&1; then
    note "vault integrity: sandbox-phase decoy vault byte-identical after the run (ransomware + probe touched nothing)"
else
    note "vault integrity: CHANGED in the sandbox phasecontainment failed"
    CONFIRM="fail"
fi

if [ ! -e "$SBX_VAULT/escape-probe.txt" ]; then
    note "escape probe: no file appeared in the vault from inside the sandbox"
else
    note "escape probe: FILE CREATED by the sandboxed runsandbox escape"
    CONFIRM="fail"
fi

SECRET_COPIES=0
if [ -d "$R/kept-out-sandbox" ]; then
    SECRET_COPIES=$(ls "$R/kept-out-sandbox"/id_ed25519* "$R/kept-out-sandbox"/aws-credentials 2>/dev/null | wc -l | tr -d ' ')
fi
if [ "$SECRET_COPIES" = "0" ]; then
    note "fs_exfil: zero decoy-secret copies in the sandbox output dir"
else
    note "fs_exfil: CONTAINMENT FAILEDdecoy secrets copied out"
    CONFIRM="fail"
fi

# --- cpu-bomb verdict --------------------------------------------------------------------
# The bombs publish their iteration counts inside their (per-phase) work dir.
BOMB_CTRL_ITERS=$(tail -1 "$R/world-ctrl/work/bomb-count.txt" 2>/dev/null | tr -d ' '); [ -z "$BOMB_CTRL_ITERS" ] && BOMB_CTRL_ITERS=0
BOMB_SBOX_ITERS=$(tail -1 "$R/world-sbox/work/bomb-count.txt" 2>/dev/null | tr -d ' '); [ -z "$BOMB_SBOX_ITERS" ] && BOMB_SBOX_ITERS=0
BOMB_SBOX_AT_EXIT=$(cat "$R/bomb-sbox-count-at-exit.txt" 2>/dev/null || echo 0)
BOMB_SBOX_FROZEN=$(cat "$R/bomb-sbox-count-frozen-check.txt" 2>/dev/null || echo 0)
BOMB_CTRL_WALL=$(cat "$R/bomb-ctrl-wall.txt" 2>/dev/null || echo "?")
BOMB_SBOX_WALL=$(cat "$R/bomb-sbox-wall.txt" 2>/dev/null || echo "?")
BOMB_SBOX_EXIT=$(cat "$R/run-bomb-sandbox.exit" 2>/dev/null || echo "?")

BOMB_CTRL_OK=0; [ "$BOMB_CTRL_ITERS" -ge "$BOMB_MIN_ITERS" ] 2>/dev/null && BOMB_CTRL_OK=1
BOMB_SBOX_OK=0; [ "$BOMB_SBOX_AT_EXIT" = "$BOMB_SBOX_FROZEN" ] 2>/dev/null && BOMB_SBOX_OK=1

# --- per-scenario verdict table -----------------------------------------------------------
want_for() {
    case "$1" in
        fs_exfil|net_exfil|env_steal|process_spawn|symlink_traverse) echo BLOCKED ;;
        ransomware) echo NO_DAMAGE ;;
        cpu_bomb) echo TIMEOUT_KILLED ;;
        *) echo "?" ;;
    esac
}
got_for() {
    local tag="$1" name="$2"
    grep "^$name:" "$OUTDIR/$tag-steps.txt" 2>/dev/null | head -1 | cut -d: -f2
}

SCENARIOS="fs_exfil net_exfil env_steal process_spawn symlink_traverse ransomware cpu_bomb"
PASSCT=0; FAILCT=0; VOIDCT=0
: > "$OUTDIR/step-results.txt"
for s in $SCENARIOS; do
    want="$(want_for "$s")"
    if [ "$s" = "cpu_bomb" ]; then
        if [ "$BOMB_CTRL_OK" = "1" ] && [ "$BOMB_SBOX_OK" = "1" ]; then
            echo "$s $want TIMEOUT_KILLED PASS" >> "$OUTDIR/step-results.txt"; PASSCT=$((PASSCT+1))
        elif [ "$BOMB_CTRL_OK" != "1" ]; then
            echo "$s $want ctrl-iters=$BOMB_CTRL_ITERS VOID" >> "$OUTDIR/step-results.txt"; VOIDCT=$((VOIDCT+1))
        else
            echo "$s $want still-looping=$BOMB_SBOX_FROZEN FAIL" >> "$OUTDIR/step-results.txt"; FAILCT=$((FAILCT+1))
        fi
        continue
    fi
    want_ctrl_case() {
        case "$1" in
            fs_exfil) echo EXFILTRATED ;;
            net_exfil) echo ATTEMPTED ;;
            env_steal) echo SECRETS_STOLEN ;;
            process_spawn) echo FULL_SYSTEM_ACCESS ;;
            symlink_traverse) echo PARTIAL_ACCESS ;;
            ransomware) echo FILES_DESTROYED ;;
        esac
    }
    if [ "$s" = "net_exfil" ]; then
        # The target records ATTEMPTED unconditionally (it cannot observe the
        # proxy); the authoritative verdict is the collector log: the sandbox
        # POST must never arrive.
        if [ "$SANDBOX_POSTS" = "0" ]; then
            echo "$s BLOCKED ATTEMPTED PASS" >> "$OUTDIR/step-results.txt"; PASSCT=$((PASSCT+1))
        else
            echo "$s BLOCKED DELIVERED FAIL" >> "$OUTDIR/step-results.txt"; FAILCT=$((FAILCT+1))
        fi
        continue
    fi
    ctrl="$(got_for control "$s")"
    sbox="$(got_for sandbox "$s")"
    if [ -z "$sbox" ]; then
        echo "$s $want MISSING VOID" >> "$OUTDIR/step-results.txt"; VOIDCT=$((VOIDCT+1)); continue
    fi
    # The control must prove the attack is real.
    ctrl_ok=0
    [ "$ctrl" = "$(want_ctrl_case "$s")" ] && ctrl_ok=1
    if [ "$ctrl_ok" = "0" ]; then
        echo "$s $want ctrl=$ctrl VOID" >> "$OUTDIR/step-results.txt"; VOIDCT=$((VOIDCT+1)); continue
    fi
    if [ "$sbox" = "$want" ]; then
        echo "$s $want $sbox PASS" >> "$OUTDIR/step-results.txt"; PASSCT=$((PASSCT+1))
    else
        echo "$s $want $sbox FAIL" >> "$OUTDIR/step-results.txt"; FAILCT=$((FAILCT+1))
    fi
done

# --- summary.json ------------------------------------------------------------------------
VERDICT="ok"
[ "$CONFIRM" != "ok" ] && VERDICT="FAILED"
[ "$FAILCT" -gt 0 ] && VERDICT="FAILED"
# VOID scenarios mean we could not verify containmentnever a passing result.
[ "$VOIDCT" -gt 0 ] && VERDICT="FAILED"
WARDEN_VER="$("$WARDEN" version 2>/dev/null | head -1)"
SBOX_EXIT=$(cat "$R/run-sandbox.exit" 2>/dev/null || echo "?")
CTRL_EXIT=$(cat "$R/run-control.exit" 2>/dev/null || echo "?")

cat > "$OUTDIR/summary.json" <<EOF
{
  "date": "$STAMP",
  "platform": "$(uname -s)-$(uname -m)",
  "warden_binary": "$WARDEN",
  "warden_version": "$WARDEN_VER",
  "sandbox_backend": "${WARDEN_BACKEND:-auto}",
  "control_exit": $CTRL_EXIT,
  "sandbox_exit": $SBOX_EXIT,
  "collector_control_posts": $CTRL_POSTS,
  "collector_sandbox_posts": $SANDBOX_POSTS,
  "sandbox_vault_integrity": "$(diff -q "$OUTDIR/vault-integrity-sbox-before.txt" "$OUTDIR/vault-integrity-sbox-after.txt" >/dev/null 2>&1 && echo IDENTICAL || echo CHANGED)",
  "cpu_bomb_control_iters": $BOMB_CTRL_ITERS,
  "cpu_bomb_control_wall_s": $BOMB_CTRL_WALL,
  "cpu_bomb_sandbox_iters": $BOMB_SBOX_ITERS,
  "cpu_bomb_sandbox_wall_s": $BOMB_SBOX_WALL,
  "cpu_bomb_sandbox_exit": $BOMB_SBOX_EXIT,
  "cpu_bomb_count_frozen": $([ "$BOMB_SBOX_OK" = "1" ] && echo true || echo false),
  "scenarios_passed": $PASSCT,
  "scenarios_failed": $FAILCT,
  "scenarios_void": $VOIDCT,
  "verdict": "$VERDICT"
}
EOF

# --- results.jsonl ------------------------------------------------------------------------
{
    echo "{\"ts\":\"$(date -Iseconds)\",\"platform\":\"$(uname -s)-$(uname -m)\",\"warden\":\"$WARDEN_VER\",\"control_exit\":$CTRL_EXIT,\"sandbox_exit\":$SBOX_EXIT}"
    while read -r name want got st; do
        echo "{\"scenario\":\"$name\",\"expected\":\"$want\",\"observed\":\"$got\",\"verdict\":\"$st\"}"
    done < "$OUTDIR/step-results.txt"
} > "$OUTDIR/results.jsonl"

# --- evidence.md ----------------------------------------------------------------------------
{
    echo "# Warden attack-simulation evidence$STAMP"
    echo ""
    echo "- platform: $(uname -s)-$(uname -m)"
    echo "- warden: $WARDEN_VER"
    echo "- sandbox backend: ${WARDEN_BACKEND:-auto (warden-detected)}"
    echo "- control (unsandboxed) exit: $CTRL_EXIT — every attack must land for results to count"
    echo "- sandbox exit: $SBOX_EXIT"
    echo ""
    echo "| attack | expected | control (unsandboxed) | sandbox | verdict |"
    echo "|--------|----------|----------------------|---------|---------|"
    while read -r name want got st; do
        ctrl="$(got_for control "$name")"; [ -z "$ctrl" ] && ctrl="—"
        [ "$name" = "cpu_bomb" ] && ctrl="ran away ($BOMB_CTRL_ITERS iters/${BOMB_CTRL_WALL}s)" && got="$got ($BOMB_SBOX_ITERS iters/${BOMB_SBOX_WALL}s)"
        echo "| $name | $want | $ctrl | $got | $st |"
    done < "$OUTDIR/step-results.txt"
    echo ""
    echo "### Host-side containment confirmations"
    echo ""
    cat "$OUTDIR/containment-notes.txt"
    echo ""
    echo "### Measured metrics (control → sandbox)"
    echo ""
    echo "| metric | control | sandbox |"
    echo "|--------|---------|---------|"
    if [ -f "$OUTDIR/sandbox-metrics.txt" ]; then
        while IFS=: read -r k v; do
            cv="$(grep "^$k:" "$OUTDIR/control-metrics.txt" 2>/dev/null | cut -d: -f2-)"
            echo "| $k | ${cv:-—} | $v |"
        done < "$OUTDIR/sandbox-metrics.txt"
    fi
    echo "| cpu_bomb_iterations | $BOMB_CTRL_ITERS (killed at ${BOMB_CTRL_WALL}s) | $BOMB_SBOX_ITERS (timeout-killed at ${BOMB_SBOX_WALL}s) |"
    echo ""
    echo "### Safety"
    echo ""
    echo "All data is decoy data created by the harness (fake keys, decoy vault, XOR transformno real crypto). Network destinations are loopback only. The sandboxed run's own audit stream is in \`audit.jsonl\`."
    echo ""
    echo "Raw artifacts: \`control-steps.txt\`, \`sandbox-steps.txt\`, \`control-metrics.txt\`, \`sandbox-metrics.txt\`, \`collector.log\`, \`vault-integrity-*.txt\`, \`audit.jsonl\`, \`summary.json\`, \`results.jsonl\`."
} > "$OUTDIR/evidence.md"

# --- report -----------------------------------------------------------------------------------
echo ""
echo "════════ ATTACK SIMULATION RESULTS ════════"
cat "$OUTDIR/evidence.md"
echo ""
if [ "$VERDICT" = "ok" ]; then
    echo "🎉 $PASSCT/$((PASSCT + FAILCT + VOIDCT)) attacks contained, all controls landed, vault intact"
    echo "   evidence: $OUTDIR"
    exit 0
fi
echo "❌ attack simulation FAILED or VOIDsee $OUTDIR/evidence.md"
exit 1

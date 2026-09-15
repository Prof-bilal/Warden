#!/bin/sh
# Warden attack-harness CPU bomb (rendered by run-attacks.sh).
#
# Busy-loops until a STOP file appears, publishing its iteration count so the
# orchestrator can measure that it was really consuming CPU. The control
# phase never creates STOP (the orchestrator kills the process — proving the
# bomb runs away unbounded); the sandbox phase relies on the policy's
# wall-clock timeout to terminate it.
STOP="@STOP@"
COUNTF="@COUNTF@"
DONEF="@DONEF@"

i=0
while [ ! -f "$STOP" ]; do
    i=$((i + 1))
    [ $((i % 1000)) -eq 0 ] && echo "$i" > "$COUNTF" 2>/dev/null
done
echo "$i" > "$COUNTF" 2>/dev/null
echo "STOPPED-GRACEFULLY" > "$DONEF" 2>/dev/null
exit 0

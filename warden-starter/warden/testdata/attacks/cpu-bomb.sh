#!/bin/sh
# Warden attack-harness CPU bomb (rendered by run-attacks.sh).
#
# Busy-loops until a STOP file appears, publishing its iteration count so the
# orchestrator can measure that it was really consuming CPU. The control
# phase never creates STOP (the orchestrator kills the processproving the
# bomb runs away unbounded); the sandbox phase relies on the policy's
# wall-clock timeout to terminate it.
STOP="@STOP@"
COUNTF="@COUNTF@"
DONEF="@DONEF@"

i=0
while [ ! -f "$STOP" ]; do
    i=$((i + 1))
    # Checkpoint every 200 iterations. Write-then-rename keeps the count file
    # complete at every instant: a plain "echo > file" truncates first, so a
    # kill landing between truncate and write left an empty file (read as 0).
    if [ $((i % 200)) -eq 0 ]; then
        echo "$i" > "$COUNTF.tmp" 2>/dev/null && mv -f "$COUNTF.tmp" "$COUNTF" 2>/dev/null
    fi
done
echo "$i" > "$COUNTF" 2>/dev/null
echo "STOPPED-GRACEFULLY" > "$DONEF" 2>/dev/null
exit 0

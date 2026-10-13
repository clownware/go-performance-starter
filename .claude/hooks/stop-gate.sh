#!/bin/sh
# Stop-gate hook (ADR-033): when an agent tries to finish a turn, run the
# test suite and the ADR check suite. A test failure or a block-status
# check failure (BLOCKER) prevents completion and feeds the report back
# into the loop; warn-status findings pass through as information.
#
# The gate blocks the same session at most STOP_GATE_MAX_BLOCKS times
# (default 2). After that it lets the agent stop, so a failure the agent
# cannot fix does not loop forever — but it tells the operator, in a
# message the harness shows them, that the turn ended with the gate
# failing. One block was not enough: in the 2026-10 test (#162) every
# blocked run finished on its next stop with the tree still failing, and
# said nothing the operator would see (#165).
#
# Kill-switch: STOP_GATE_OFF=1 disables the gate for one session.

[ "$STOP_GATE_OFF" = "1" ] && exit 0

payload=$(cat)
max_blocks=${STOP_GATE_MAX_BLOCKS:-2}

# One counter per session, so a new session starts from zero.
session=$(printf '%s' "$payload" | sed -n 's/.*"session_id"[[:space:]]*:[[:space:]]*"\([A-Za-z0-9_-]*\)".*/\1/p')
[ -n "$session" ] || session=unknown
state="${TMPDIR:-/tmp}/stop-gate-$session.count"

# A session that died between a block and its next stop leaves its counter
# behind. Sweep the ones older than a day; only this hook's own files.
find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'stop-gate-*.count' -mtime +1 -delete 2>/dev/null

failure=""
if ! test_out=$(go test ./... 2>&1); then
  failure="Stop-gate BLOCKER (ADR-021): go test ./... failed."
  detail=$(echo "$test_out" | tail -20)
else
  check_out=$(go run ./scripts/adrcheck 2>&1)
  if [ $? -ne 0 ]; then
    failure="Stop-gate BLOCKER (ADR-033): a block-status ADR check failed."
    detail=$check_out
  fi
fi

if [ -z "$failure" ]; then
  rm -f "$state"
  # Warn-only findings are information, not a gate.
  echo "$check_out" | grep -A2 '^WARNING' || true
  exit 0
fi

blocks=$(cat "$state" 2>/dev/null || echo 0)
case "$blocks" in ''|*[!0-9]*) blocks=0 ;; esac

if [ "$blocks" -lt "$max_blocks" ]; then
  echo $((blocks + 1)) > "$state"
  {
    echo "$failure Halt and fix before finishing (block $((blocks + 1)) of $max_blocks)."
    echo "$detail"
    echo "If you cannot fix it, do not edit a test, a check or its status to get past this. Start your final message with the line 'GATE FAILING:' and say what is failing and what you need."
  } >&2
  exit 2
fi

# Out of blocks: let the stop through, and tell the operator.
rm -f "$state"
first=$(printf '%s' "$failure" | sed 's/"/\\"/g')
printf '{"systemMessage":"Stop-gate: this turn ended with the gate still FAILING after %s blocks. %s Run task ci before trusting the result."}\n' "$max_blocks" "$first"
exit 0

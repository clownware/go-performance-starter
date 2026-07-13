#!/bin/sh
# Stop-gate hook (ADR-033): when an agent tries to finish a turn, run the
# test suite and the ADR check suite. A test failure or a block-status
# check failure (BLOCKER) prevents completion and feeds the report back
# into the loop; warn-status findings pass through as information.
#
# Kill-switch: STOP_GATE_OFF=1 disables the gate for one session.

[ "$STOP_GATE_OFF" = "1" ] && exit 0

# Loop protection: if this hook already blocked once and the agent is
# responding to that block, let the stop proceed rather than ping-pong.
payload=$(cat)
case "$payload" in
  *'"stop_hook_active":true'*) exit 0 ;;
esac

if ! test_out=$(go test ./... 2>&1); then
  echo "Stop-gate BLOCKER (ADR-021): go test ./... failed. Halt and fix before finishing." >&2
  echo "$test_out" | tail -20 >&2
  exit 2
fi

check_out=$(go run ./scripts/adrcheck 2>&1)
if [ $? -ne 0 ]; then
  echo "Stop-gate BLOCKER (ADR-033): a block-status ADR check failed." >&2
  echo "$check_out" >&2
  exit 2
fi

# Warn-only findings are information, not a gate.
echo "$check_out" | grep -A2 '^WARNING' || true
exit 0

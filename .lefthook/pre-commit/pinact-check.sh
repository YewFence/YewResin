#!/usr/bin/env bash

ghtkn get | pinact token set -stdin

pinact_exit=0
pinact_output=$(pinact run --check 2>&1) || pinact_exit=$?

if [ "$pinact_exit" -ne 0 ]; then
  echo "$pinact_output"
  echo ""
  echo "❌ pinact check failed, run 'pinact run' to pin action versions. run 'ghtkn get | pinact token set -stdin' before to set your Github token"
else
  echo "✅ All actions are pinned"
fi

echo "💡 Tip: run 'pinact run -u' to update pinned versions"
exit "$pinact_exit"

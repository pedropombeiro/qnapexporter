#!/bin/sh
# Runs gitleaks, scanning the full repo in CI (no staged files there) and only
# staged changes locally for fast pre-commit feedback. Output is suppressed
# unless gitleaks fails.
if [ -n "$CI" ]; then
  set -- git --no-banner --redact --log-level=error
else
  set -- git --staged --no-banner --redact --log-level=error
fi

output=$(gitleaks "$@" 2>&1)
code=$?
[ $code -ne 0 ] && printf '%s\n' "$output"
exit $code

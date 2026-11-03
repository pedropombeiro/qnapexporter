#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <commit-message-file>" >&2
  exit 1
fi

subject=$(sed -n '1p' "$1")

if [[ $subject =~ ^(Merge|Revert)[[:space:]] ]] || [[ $subject =~ ^(fixup!|squash!)[[:space:]] ]]; then
  exit 0
fi

if [[ $subject =~ ^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-zA-Z0-9._/-]+\))?!?:[[:space:]]+[^[:space:]].*$ ]]; then
  exit 0
fi

cat >&2 <<'EOF'
Commit subject must follow Conventional Commits:

  <type>[optional scope][!]: <description>

Allowed types: build, chore, ci, docs, feat, fix, perf, refactor, revert, style, test
Example: fix(exporter): report swap cache separately
EOF
exit 1

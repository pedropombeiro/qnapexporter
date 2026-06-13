#!/bin/sh
# Validate the 100-byte trailer ("tail") that QDK appends to every .qpkg.
#
# QTS/App Center rejects packages with a malformed trailer ("file format
# error"). The trailer is a fixed 100-byte block. Observed field layout
# (see QDK qbuild add_qpkg_encryption / add_qpkg_tail):
#
#   bytes  0-39  : model + reserved padding (blank for generic packages)
#   bytes 40-49  : checksum / firmware field (populated by qpkg_encrypt)
#   bytes 60-79  : QPKG name
#   bytes 80-89  : QPKG version
#   bytes 90-99  : "QNAPQPKG" magic flag
#
# QDK 2.5.0 stopped populating the checksum field (qpkg_encrypt was moved off
# PATH), leaving bytes 40-49 blank, which QTS rejects with a "file format
# error". This script guards against that and any future QDK regression by
# asserting:
#
#   1. The magic flag (bytes 90-99) is "QNAPQPKG", confirming a real tail.
#   2. The checksum field (bytes 40-49) is not blank.
#
# Usage: validate-tail.sh <path-to-qpkg>

set -eu

QPKG_FILE="${1:-}"

if [ -z "$QPKG_FILE" ]; then
  echo "::error::validate-tail.sh: no QPKG file argument provided"
  exit 1
fi

if [ ! -f "$QPKG_FILE" ]; then
  echo "::error::QPKG file not found: $QPKG_FILE"
  exit 1
fi

TAIL_LEN=100
SIZE=$(wc -c <"$QPKG_FILE")
if [ "$SIZE" -lt "$TAIL_LEN" ]; then
  echo "::error::QPKG file is smaller than the $TAIL_LEN-byte trailer ($SIZE bytes)"
  exit 1
fi

# Extract fixed-offset fields from the trailer. dd is used (rather than shell
# parameter expansion) so binary bytes are handled reliably.
flag=$(tail -c "$TAIL_LEN" "$QPKG_FILE" | dd bs=1 skip=90 count=8 2>/dev/null)
checksum=$(tail -c "$TAIL_LEN" "$QPKG_FILE" | dd bs=1 skip=40 count=10 2>/dev/null)

if [ "$flag" != "QNAPQPKG" ]; then
  echo "::error::QPKG trailer magic flag is '$flag', expected 'QNAPQPKG'. The package trailer is malformed."
  exit 1
fi

checksum_trimmed=$(printf '%s' "$checksum" | tr -d ' ')
if [ -z "$checksum_trimmed" ]; then
  echo "::error::QPKG trailer checksum field is empty. qbuild's qpkg_encrypt step did not run, so QTS will reject this package with a 'file format error' (see issue #103). Ensure qpkg_encrypt is on PATH in the QDK build image."
  exit 1
fi

echo "QPKG trailer OK (flag='$flag', checksum='$checksum_trimmed')"

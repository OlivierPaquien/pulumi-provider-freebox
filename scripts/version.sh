#!/usr/bin/env bash
# Helpers to read/write the provider version in main.go.
set -euo pipefail

read_version() {
  grep '^var version = ' main.go | sed 's/^var version = "\(.*\)".*/\1/'
}

write_version() {
  local version="${1:?version required}"
  sed -i 's/^var version = ".*"$/var version = "'"$version"'"/' main.go
}

case "${1:-}" in
  read) read_version ;;
  write)
    write_version "${2:?usage: $0 write X.Y.Z}"
    ;;
  *)
    echo "usage: $0 read|write <version>" >&2
    exit 1
    ;;
esac

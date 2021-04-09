#!/usr/bin/env bash

set -Eeuo pipefail

function cleanup() {
  trap - SIGINT SIGTERM ERR EXIT
  echo "cleanup running"
}

trap cleanup SIGINT SIGTERM ERR EXIT

SCRIPT_NAME="$(basename "$(test -L "$0" && readlink "$0" || echo "$0")")"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)"
REPO_ROOT="$(cd ${SCRIPT_DIR} && git rev-parse --show-toplevel)"

echo "${SCRIPT_NAME} is running... "

if [[ ! -f "$(go env GOPATH)/bin/golangci-lint" ]] && [[ ! -f "/usr/local/bin/golangci-lint" ]]; then
  echo "Install golangci-lint"
  echo "run 'make install-tools' "
  exit 1
fi

echo "Linting..."

golangci-lint run --no-config --disable-all -E govet
golangci-lint run --new-from-rev=HEAD~ --config .golangci.yml

echo "Done."

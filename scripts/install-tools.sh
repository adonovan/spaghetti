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

cd ${REPO_ROOT}/tools || exit 1

export GO111MODULE=on

go generate -tags=tools ./...

cd - || exit 1

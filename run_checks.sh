#!/usr/bin/env bash

set -euo pipefail

if [[ $# -gt 1 ]]; then
  echo "Usage: $0 [stage]"
  exit 2
fi

if [[ $# -eq 0 ]]; then
  STAGES="dep fmt goimports gosec build vet race test vuln"
else
  STAGES=$1
fi

function stage_build() {
  go build ./...
}

function stage_generate() {
  go generate -mod=readonly ./...
}

function stage_fmt() {
  go fmt ./...
}

function stage_vet() {
  go vet ./...
}

function stage_goimports() {
  find ./ -mindepth 1 -maxdepth 1 -type d \
    -not \( -path './vendor' \) \
    -not \( -path './.*' \) -print0 | \
  xargs -0 goimports -w
}

function stage_gosec() {
  gosec -quiet ./...
}

function stage_race() {
  go build -race ./...
}

function stage_dep() {
  go mod tidy
  go mod vendor
}

function stage_test() {
  go test ./...
}

function stage_vuln() {
  govulncheck ./...
}

set +e

for i in $STAGES; do
  proc=stage_$i
  if [[ $(type -t $proc) != function ]]; then
    echo "Invalid stage $i"
    exit 1
  fi

  printf '[RUNNING] \e[1;97mStage \e[1;33m%s\e[0m' "$i"
  output=$($proc)
  if [[ $? -ne 0 ]]; then
    printf '\r[\e[1;31m  ERROR\e[0m] \e[1;97mStage \e[1;33m%s\e[0m\n' "$i"
    if [[ ! -z $output ]]; then
      printf '%s\n' "$output"
    fi
  else
    printf '\r[\e[1;32mSUCCESS\e[0m] \e[1;97mStage \e[1;33m%s\e[0m\n' "$i"
  fi

done

exit 0

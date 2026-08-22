#!/bin/sh
set -u

set -- ./api/assistant-api/... ./cmd/assistant/...

exit_status=0
failures=

run_command() {
  command_name=$1
  shift

  "$@"
  command_status=$?
  if [ "$command_status" -ne 0 ]; then
    if [ "$exit_status" -eq 0 ]; then
      exit_status=$command_status
    fi
    failures="${failures}
- ${command_name} (exit ${command_status})"
  fi
}

run_command 'go list' go list -deps -test "$@" >/dev/null
run_command 'go vet' go vet "$@"
run_command 'go test coverage' go test -count=1 -covermode=atomic -coverprofile=/tmp/assistant-coverage.out "$@"
run_command 'go test race' go test -race -count=1 "$@"
run_command 'govulncheck' govulncheck "$@"
run_command 'go build' go build -trimpath -o /tmp/assistant-api ./cmd/assistant/assistant.go

if [ "$exit_status" -ne 0 ]; then
  printf 'assistant native CI failures:%s\n' "$failures" >&2
fi

exit "$exit_status"

#!/usr/bin/env bash
set -Eeuo pipefail

baseline_directory=${1:?usage: materialize-baseline.sh BASELINE_DIRECTORY}
case ${baseline_directory} in
  /|.|..)
    echo "Refusing unsafe baseline directory: ${baseline_directory}" >&2
    exit 1
    ;;
esac

if [[ ${EVENT_NAME:-} == pull_request ]]; then
  target_sha=${PULL_REQUEST_BASE_SHA:?pull request base SHA is required}
else
  target_sha=$(git rev-parse HEAD^)
fi

if ! [[ ${target_sha} =~ ^[0-9a-f]{40}$ ]]; then
  echo "Target SHA is not a full commit SHA: ${target_sha}" >&2
  exit 1
fi
git cat-file -e "${target_sha}^{commit}"

mapfile -t protobuf_files < <(
  git ls-tree -r --name-only "${target_sha}" -- protos | grep -E '^protos/[^/]+\.go$' || true
)
if (( ${#protobuf_files[@]} == 0 )); then
  echo "Target commit has no tracked protos/*.go files" >&2
  exit 1
fi

rm -rf "${baseline_directory}"
mkdir -p "${baseline_directory}"
git archive "${target_sha}" go.mod go.sum openapi/artifacts "${protobuf_files[@]}" | \
  tar -x -C "${baseline_directory}"
git show "${target_sha}:docker-compose.yml" > "${baseline_directory}/docker-compose.yml"

if [[ -n ${GITHUB_OUTPUT:-} ]]; then
  printf 'directory=%s\n' "${baseline_directory}" >> "${GITHUB_OUTPUT}"
  printf 'target-sha=%s\n' "${target_sha}" >> "${GITHUB_OUTPUT}"
fi
if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
  {
    echo "## Contract Baseline"
    echo
    echo "- Target commit: \`${target_sha}\`"
  } >> "${GITHUB_STEP_SUMMARY}"
fi

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

if [[ ! -e protos/artifacts/.git ]]; then
  echo "The protos/artifacts submodule is not initialized" >&2
  exit 1
fi

target_protobuf_sha=$(git ls-tree "${target_sha}" -- protos/artifacts | awk '$1 == "160000" && $2 == "commit" { print $3 }')
current_protobuf_sha=$(git ls-tree HEAD -- protos/artifacts | awk '$1 == "160000" && $2 == "commit" { print $3 }')
checked_out_protobuf_sha=$(git -C protos/artifacts rev-parse HEAD)

if [[ -z ${target_protobuf_sha} || -z ${current_protobuf_sha} ]]; then
  echo "Unable to resolve the protobuf submodule SHA" >&2
  exit 1
fi
if [[ ${checked_out_protobuf_sha} != "${current_protobuf_sha}" ]]; then
  echo "Checked-out protobuf submodule does not match the current commit" >&2
  exit 1
fi

rm -rf "${baseline_directory}"
mkdir -p "${baseline_directory}" "${baseline_directory}/protos/artifacts"
git archive "${target_sha}" openapi/artifacts | tar -x -C "${baseline_directory}"
git show "${target_sha}:docker-compose.yml" > "${baseline_directory}/docker-compose.yml"

if ! git -C protos/artifacts cat-file -e "${target_protobuf_sha}^{commit}" 2>/dev/null; then
  git -C protos/artifacts fetch --no-tags --depth=1 origin "${target_protobuf_sha}"
fi
git -C protos/artifacts archive "${target_protobuf_sha}" | \
  tar -x -C "${baseline_directory}/protos/artifacts"

if [[ -n ${GITHUB_OUTPUT:-} ]]; then
  printf 'directory=%s\n' "${baseline_directory}" >> "${GITHUB_OUTPUT}"
fi
if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
  {
    echo "## Contract Baseline"
    echo
    echo "- Target commit: \`${target_sha}\`"
    echo "- Current protobuf commit: \`${current_protobuf_sha}\`"
    echo "- Baseline protobuf commit: \`${target_protobuf_sha}\`"
  } >> "${GITHUB_STEP_SUMMARY}"
fi

#!/usr/bin/env bash
set -euo pipefail

GO_PROJECT_MODULE="github.com/rapidaai/protos"
OUT_DIR="/protos/"
OPENAPI_ARTIFACT_DIR="./openapi/artifacts"
OPENAPI_OUT_DIR="./openapi"
OAPI_CODEGEN_VERSION="${OAPI_CODEGEN_VERSION:-v2.4.1}"

rm -f ./protos/*.pb.go
rm -f ./protos/authentication/*.pb.go
rm -f "${OPENAPI_OUT_DIR}"/*.gen.go

find ./api/document-api/app/bridges/artifacts/protos -name "*_pb2.py" -delete
find ./api/document-api/app/bridges/artifacts/protos -name "*_pb2_grpc.py" -delete
find ./api/document-api/app/bridges/artifacts/protos -name "*_pb2.pyi" -delete


protoc -I=./protos/artifacts/ --go_opt=module="${GO_PROJECT_MODULE}" --go_out=."${OUT_DIR}" --go-grpc_opt=module="${GO_PROJECT_MODULE}" --go-grpc_out=require_unimplemented_servers=false:."${OUT_DIR}" ./protos/artifacts/*.proto

python3 -m grpc.tools.protoc \
    -I ./protos/artifacts \
    --pyi_out=./api/document-api/app/bridges/artifacts/protos \
    --python_out=./api/document-api/app/bridges/artifacts/protos \
    --grpc_python_out=./api/document-api/app/bridges/artifacts/protos \
    ./protos/artifacts/*.proto


find "api/document-api/app/bridges/artifacts/protos/" \
  -type f \( -name "*.py" -o -name "*.pyi" \) \
  -exec sed -i.bak -E \
    '/^import [a-zA-Z0-9_]+_pb2/ s|import ([a-zA-Z0-9_]+_pb2)|import app.bridges.artifacts.protos.\1|' {} +

# Remove backup files
find "api/document-api/app/bridges/artifacts/protos/" -name "*.bak" -delete

go run "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION}" \
  -generate types,skip-prune \
  -package openapi \
  -o "${OPENAPI_OUT_DIR}/common.gen.go" \
  "${OPENAPI_ARTIFACT_DIR}/common.yaml"

for spec in "${OPENAPI_ARTIFACT_DIR}"/*.yaml; do
  spec_name="$(basename "${spec}" .yaml)"
  if [ "${spec_name}" = "common" ]; then
    continue
  fi

  out_name="${spec_name%-api}.gen.go"
  go run "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION}" \
    -generate types \
    -package openapi \
    -import-mapping=./common.yaml:- \
    -o "${OPENAPI_OUT_DIR}/${out_name}" \
    "${spec}"
done

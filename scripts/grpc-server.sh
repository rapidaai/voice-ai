# Path to this plugin, Note this must be an abolsute path on Windows (see #15)
# PROTOC_GEN_TS_PATH="${PWD}/node_modules/.bin/protoc-gen-ts"
git submodule update --recursive
GO_PROJECT_MODULE="github.com/lexatic/web-backend/protos/lexatic-backend"
OUT_DIR="/protos/lexatic-backend/"
protoc -Iprotos --go_opt=module="${GO_PROJECT_MODULE}" --go_out=."${OUT_DIR}" --go-grpc_opt=module="${GO_PROJECT_MODULE}" --go-grpc_out=require_unimplemented_servers=false:."${OUT_DIR}" ./protos/lexatic-backend/*.proto

GO_PROJECT_MODULE="github.com/lexatic/web-backend/protos/lexatic-backend"
OUT_DIR="/protos/lexatic-backend/"
rm -rf ./protos/lexatic-backend/*.go
protoc -I=./protos/lexatic-backend/ --go_opt=module="${GO_PROJECT_MODULE}" --go_out=."${OUT_DIR}" --go-grpc_opt=module="${GO_PROJECT_MODULE}" --go-grpc_out=require_unimplemented_servers=false:."${OUT_DIR}" ./protos/lexatic-backend/*.proto
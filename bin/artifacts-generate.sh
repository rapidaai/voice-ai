GO_PROJECT_MODULE="github.com/rapidaai/protos"
OUT_DIR="/protos/"
rm -rf ./protos/*.go
protoc -I=./protos/artifacts/ --go_opt=module="${GO_PROJECT_MODULE}" --go_out=."${OUT_DIR}" --go-grpc_opt=module="${GO_PROJECT_MODULE}" --go-grpc_out=require_unimplemented_servers=false:."${OUT_DIR}" ./protos/artifacts/*.proto
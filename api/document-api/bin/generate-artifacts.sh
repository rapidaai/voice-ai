
# Remove existing generated Python files
echo "Removing existing generated Python files..."
find ./app/bridges/artifacts/protos -name "*_pb2.py" -delete
find ./app/bridges/artifacts/protos -name "*_pb2_grpc.py" -delete
find ./app/bridges/artifacts/protos -name "*_pb2.pyi" -delete

python3 -m grpc.tools.protoc \
    -I ./app/bridges/artifacts/protos \
    --pyi_out=./app/bridges/artifacts/protos \
    --python_out=./app/bridges/artifacts/protos \
    --grpc_python_out=./app/bridges/artifacts/protos \
    ./app/bridges/artifacts/protos/*.proto


# Existing commands
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import common_pb2 as common__pb2/import app.bridges.artifacts.protos.common_pb2 as common__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.pyi' -exec sed -i.bak 's/import common_pb2 as _common_pb2/import app.bridges.artifacts.protos.common_pb2 as _common_pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import web_api_pb2 as web__api__pb2/import app.bridges.artifacts.protos.web_api_pb2 as web__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import provider_api_pb2 as provider__api__pb2/import app.bridges.artifacts.protos.provider_api_pb2 as provider__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import vault_api_pb2 as vault__api__pb2/import app.bridges.artifacts.protos.vault_api_pb2 as vault__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import integration_api_pb2 as integration__api__pb2/import app.bridges.artifacts.protos.integration_api_pb2 as integration__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import knowledge_api_pb2 as knowledge__api__pb2/import app.bridges.artifacts.protos.knowledge_api_pb2 as knowledge__api__pb2/g' {} +

# New commands for additional files
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import assistant_api_pb2 as assistant__api__pb2/import app.bridges.artifacts.protos.assistant_api_pb2 as assistant__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import assistant_deployment_pb2 as assistant__deployment__pb2/import app.bridges.artifacts.protos.assistant_deployment_pb2 as assistant__deployment__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import audit_logging_api_pb2 as audit__logging__api__pb2/import app.bridges.artifacts.protos.audit_logging_api_pb2 as audit__logging__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import connect_api_pb2 as connect__api__pb2/import app.bridges.artifacts.protos.connect_api_pb2 as connect__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import document_api_pb2 as document__api__pb2/import app.bridges.artifacts.protos.document_api_pb2 as document__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import endpoint_api_pb2 as endpoint__api__pb2/import app.bridges.artifacts.protos.endpoint_api_pb2 as endpoint__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import invoker_api_pb2 as invoker__api__pb2/import app.bridges.artifacts.protos.invoker_api_pb2 as invoker__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import sendgrid_api_pb2 as sendgrid__api__pb2/import app.bridges.artifacts.protos.sendgrid_api_pb2 as sendgrid__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import talk_api_pb2 as talk__api__pb2/import app.bridges.artifacts.protos.talk_api_pb2 as talk__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import tool_api_pb2 as tool__api__pb2/import app.bridges.artifacts.protos.tool_api_pb2 as tool__api__pb2/g' {} +
find "app/bridges/artifacts/protos/" -name '*.py' -exec sed -i.bak 's/import workflow_api_pb2 as workflow__api__pb2/import app.bridges.artifacts.protos.workflow_api_pb2 as workflow__api__pb2/g' {} +

# Remove backup files created by sed
find "app/bridges/artifacts/protos/" -name '*.bak' -exec rm {} +
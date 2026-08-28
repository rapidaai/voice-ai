# Run the document API without Docker.
run-document:
    PYTHONPATH=api/document-api uvicorn app.main:app --host 0.0.0.0 --port 9010

# Run the assistant API without Docker.
run-assistant:
    go run cmd/assistant/assistant.go

# Run the web API without Docker.
run-web:
    go run cmd/web/web.go

# Run the endpoint API without Docker.
run-endpoint:
    go run cmd/endpoint/endpoint.go

# Run the integration API without Docker.
run-integration:
    go run cmd/integration/integration.go

# Run the UI development server.
run-ui:
    cd ui && yarn start

# Run TTS integration tests.
test-tts-integration:
    go test -tags integration -v -run TestTTS -timeout 120s ./api/assistant-api/internal/transformer/...

# Run STT integration tests.
test-stt-integration:
    go test -tags integration -v -run TestSTT -timeout 120s ./api/assistant-api/internal/transformer/...

# Run all transformer integration tests.
test-transformer-integration:
    go test -tags integration -v -timeout 180s ./api/assistant-api/internal/transformer/...

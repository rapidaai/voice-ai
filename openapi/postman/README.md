# Postman Collections

Postman artifacts are organized by service:

```text
openapi/postman/
  <service-name>/
    <service-name>.postman_collection.json
    <service-name>.smoke.postman_collection.json
```

Use this convention for every service that gets an API collection.

## Customer Collections

`<service-name>.postman_collection.json` is the customer/shareable collection. It should contain organized API folders only and no CI-only smoke flow.

Current customer collections:

- `assistant-api/assistant-api.postman_collection.json`

## CI/CD Smoke Collections

`<service-name>.smoke.postman_collection.json` is the CI/CD smoke collection. It should contain ordered request flows with assertions, generated mock request data, and captured variables.

Current smoke collections:

- `assistant-api/assistant-api.smoke.postman_collection.json`

Run assistant-api smoke with Newman:

```sh
npx --yes newman run openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json \
  --folder "Smoke Flow" \
  --bail \
  --env-var baseUrl=http://localhost:9007 \
  --env-var apiKey="$API_KEY"
```

`apiKey` must be the raw project API key without a prefix. The collection sends it
through the `x-api-key` header, which establishes project scope without requiring
`x-auth-id` or `x-project-id` values.

The CI smoke runner executes the complete collection twice. The first run uses the
project API key. The second run uses a personal access token through `authorization`,
`x-auth-id`, and `x-project-id`. It then calls the assistant gRPC API with the pinned
`@rapidaai/nodejs` release using both authentication methods.

The assistant-api smoke flow runs all OpenAPI-managed REST operations:

1. Create assistant and capture `assistantId`
2. Create/get/update/get webhook configuration using the UI option fields
3. Create/get/update/get storage configuration using AWS destination fields
4. Create/get/update/get HTTP authentication configuration
5. Create/get/update/get endpoint analysis configuration
6. List all configurations and verify every captured configuration ID
7. Create/get/list API, debugger, phone, webplugin, and WhatsApp deployments
8. Delete webhook, storage, authentication, and analysis configurations

The customer collection groups webhook, storage, authentication, and analysis
requests separately. Set `storageCredentialId` and `analysisEndpointId` to real
resource IDs when the downstream runtime will execute those configurations.

## Regeneration

Assistant-api collections are generated from:

```sh
openapi/artifacts/assistant-api.yaml
```

Regenerate after changing the OpenAPI artifact:

```sh
python3 openapi/scripts/generate_assistant_postman_collection.py
```

Check whether generated collections are current:

```sh
python3 openapi/scripts/generate_assistant_postman_collection.py --check
```

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
  --env-var authToken="$AUTH_TOKEN"
```

The assistant-api smoke flow runs:

1. Create assistant and capture `assistantId`
2. Create configuration and capture `configurationId`
3. Get/list/update/get/delete configuration
4. Create API deployment and capture `apiDeploymentId`
5. Get/list API deployments

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

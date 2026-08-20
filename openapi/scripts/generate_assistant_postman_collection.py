#!/usr/bin/env python3
"""Generate a curated Postman collection for assistant-api OpenAPI paths."""

from __future__ import annotations

import argparse
import copy
import json
import sys
import uuid
from pathlib import Path
from typing import Any

import yaml


ROOT = Path(__file__).resolve().parents[2]
DEFAULT_SPEC = ROOT / "openapi" / "artifacts" / "assistant-api.yaml"
DEFAULT_COLLECTION_OUTPUT = (
    ROOT / "openapi" / "postman" / "assistant-api" / "assistant-api.postman_collection.json"
)
DEFAULT_SMOKE_OUTPUT = (
    ROOT / "openapi" / "postman" / "assistant-api" / "assistant-api.smoke.postman_collection.json"
)

POSTMAN_SCHEMA = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"


REQUEST_EXAMPLES: dict[str, dict[str, Any]] = {
    "CreateAssistantRequest": {
        "assistantProvider": {
            "model": {
                "modelProviderName": "{{modelProviderName}}",
                "template": {
                    "prompt": [
                        {
                            "role": "system",
                            "content": "You are a helpful voice assistant.",
                        }
                    ]
                },
                "assistantModelOptions": [
                    {"key": "temperature", "value": "0.7"},
                    {"key": "max_tokens", "value": "512"},
                ],
            }
        },
        "description": "Assistant created from the Postman collection.",
        "visibility": "private",
        "language": "en",
        "source": "postman",
        "sourceIdentifier": "{{sourceIdentifier}}",
        "tags": ["postman", "assistant-api"],
        "name": "Postman Assistant",
    },
    "CreateAssistantConfigurationRequest": {
        "assistantId": "{{assistantId}}",
        "configurationType": "webhook",
        "provider": "custom",
        "enabled": True,
        "options": [
            {"key": "url", "value": "https://example.com/webhook"},
            {"key": "method", "value": "POST"},
        ],
    },
    "UpdateAssistantConfigurationRequest": {
        "configurationType": "webhook",
        "provider": "custom",
        "enabled": True,
        "options": [
            {"key": "url", "value": "https://example.com/webhook"},
            {"key": "method", "value": "POST"},
        ],
    },
    "CreateAssistantDebuggerDeploymentRequest": {
        "assistantId": "{{assistantId}}",
        "greeting": "Hello, how can I help you today?",
        "greetingInterruptible": True,
        "mistake": "Sorry, I did not catch that.",
        "unclearInputTimeout": 4,
        "unclearInputMessage": "Could you repeat that?",
        "idealTimeout": 30,
        "idealTimeoutBackoff": 2,
        "idealTimeoutMessage": "Are you still there?",
        "maxSessionDuration": 300,
        "inputAudio": {
            "audioType": "input",
            "audioProvider": "{{sttProviderName}}",
            "status": "ACTIVE",
            "audioOptions": [{"key": "language", "value": "en-US"}],
        },
        "outputAudio": {
            "audioType": "output",
            "audioProvider": "{{ttsProviderName}}",
            "status": "ACTIVE",
            "audioOptions": [{"key": "voice", "value": "default"}],
        },
    },
    "CreateAssistantPhoneDeploymentRequest": {
        "assistantId": "{{assistantId}}",
        "greeting": "Hello, how can I help you today?",
        "greetingInterruptible": True,
        "mistake": "Sorry, I did not catch that.",
        "unclearInputTimeout": 4,
        "unclearInputMessage": "Could you repeat that?",
        "idealTimeout": 30,
        "idealTimeoutBackoff": 2,
        "idealTimeoutMessage": "Are you still there?",
        "maxSessionDuration": 300,
        "phoneProviderName": "{{phoneProviderName}}",
        "phoneOptions": [{"key": "callerId", "value": "{{callerId}}"}],
        "inputAudio": {
            "audioType": "input",
            "audioProvider": "{{sttProviderName}}",
            "status": "ACTIVE",
            "audioOptions": [{"key": "language", "value": "en-US"}],
        },
        "outputAudio": {
            "audioType": "output",
            "audioProvider": "{{ttsProviderName}}",
            "status": "ACTIVE",
            "audioOptions": [{"key": "voice", "value": "default"}],
        },
    },
    "CreateAssistantApiDeploymentRequest": {
        "assistantId": "{{assistantId}}",
        "greeting": "Hello, how can I help you today?",
        "greetingInterruptible": True,
        "mistake": "Sorry, I did not catch that.",
        "unclearInputTimeout": 4,
        "unclearInputMessage": "Could you repeat that?",
        "idealTimeout": 30,
        "idealTimeoutBackoff": 2,
        "idealTimeoutMessage": "Are you still there?",
        "maxSessionDuration": 300,
        "inputAudio": {
            "audioType": "input",
            "audioProvider": "{{sttProviderName}}",
            "status": "ACTIVE",
            "audioOptions": [{"key": "language", "value": "en-US"}],
        },
        "outputAudio": {
            "audioType": "output",
            "audioProvider": "{{ttsProviderName}}",
            "status": "ACTIVE",
            "audioOptions": [{"key": "voice", "value": "default"}],
        },
    },
    "CreateAssistantWebpluginDeploymentRequest": {
        "assistantId": "{{assistantId}}",
        "greeting": "Hello, how can I help you today?",
        "greetingInterruptible": True,
        "mistake": "Sorry, I did not catch that.",
        "unclearInputTimeout": 4,
        "unclearInputMessage": "Could you repeat that?",
        "idealTimeout": 30,
        "idealTimeoutBackoff": 2,
        "idealTimeoutMessage": "Are you still there?",
        "maxSessionDuration": 300,
        "suggestion": ["Talk to support", "Check order status"],
        "inputAudio": {
            "audioType": "input",
            "audioProvider": "{{sttProviderName}}",
            "status": "ACTIVE",
            "audioOptions": [{"key": "language", "value": "en-US"}],
        },
        "outputAudio": {
            "audioType": "output",
            "audioProvider": "{{ttsProviderName}}",
            "status": "ACTIVE",
            "audioOptions": [{"key": "voice", "value": "default"}],
        },
    },
    "CreateAssistantWhatsappDeploymentRequest": {
        "assistantId": "{{assistantId}}",
        "greeting": "Hello, how can I help you today?",
        "greetingInterruptible": True,
        "mistake": "Sorry, I did not catch that.",
        "unclearInputTimeout": 4,
        "unclearInputMessage": "Could you repeat that?",
        "idealTimeout": 30,
        "idealTimeoutBackoff": 2,
        "idealTimeoutMessage": "Are you still there?",
        "maxSessionDuration": 300,
        "whatsappProviderName": "{{whatsappProviderName}}",
        "whatsappOptions": [{"key": "businessNumber", "value": "{{whatsappBusinessNumber}}"}],
    },
}


COLLECTION_VARIABLES: list[dict[str, str]] = [
    {"key": "baseUrl", "value": "http://localhost:9007", "type": "string"},
    {"key": "authToken", "value": "", "type": "string"},
    {"key": "authId", "value": "", "type": "string"},
    {"key": "projectId", "value": "", "type": "string"},
    {"key": "assistantId", "value": "1", "type": "string"},
    {"key": "configurationId", "value": "1", "type": "string"},
    {"key": "apiDeploymentId", "value": "1", "type": "string"},
    {"key": "page", "value": "1", "type": "string"},
    {"key": "pageSize", "value": "20", "type": "string"},
    {"key": "sourceIdentifier", "value": "1", "type": "string"},
    {"key": "modelProviderName", "value": "openai", "type": "string"},
    {"key": "sttProviderName", "value": "deepgram", "type": "string"},
    {"key": "ttsProviderName", "value": "elevenlabs", "type": "string"},
    {"key": "phoneProviderName", "value": "twilio", "type": "string"},
    {"key": "whatsappProviderName", "value": "twilio", "type": "string"},
    {"key": "callerId", "value": "+15551234567", "type": "string"},
    {"key": "whatsappBusinessNumber", "value": "+15551234567", "type": "string"},
]


def success_tests() -> list[str]:
    return [
        "pm.test('status is 200', function () {",
        "  pm.response.to.have.status(200);",
        "});",
        "let jsonData = {};",
        "if (pm.response && pm.response.text()) {",
        "  jsonData = pm.response.json();",
        "}",
        "pm.test('success is true', function () {",
        "  pm.expect(jsonData.success).to.eql(true);",
        "});",
    ]


SMOKE_FLOW_STEPS: list[dict[str, Any]] = [
    {
        "name": "01 Create Assistant",
        "operationId": "createAssistant",
        "tests": [
            *success_tests(),
            "pm.test('assistant is created', function () {",
            "  const data = jsonData.data || {};",
            "  pm.expect(data.name).to.eql('Postman Assistant');",
            "  pm.expect(data.id).to.exist;",
            "});",
            "if (jsonData.data && jsonData.data.id) {",
            "  pm.collectionVariables.set('assistantId', String(jsonData.data.id));",
            "}",
        ],
    },
    {
        "name": "02 Create Configuration",
        "operationId": "createAssistantConfiguration",
        "tests": [
            *success_tests(),
            "pm.test('configuration is created for assistant', function () {",
            "  const data = jsonData.data || {};",
            "  pm.expect(data.assistantId).to.eql(pm.collectionVariables.get('assistantId'));",
            "  pm.expect(data.configurationType).to.eql('webhook');",
            "  pm.expect(data.provider).to.eql('custom');",
            "});",
            "if (jsonData.data && jsonData.data.id) {",
            "  pm.collectionVariables.set('configurationId', String(jsonData.data.id));",
            "}",
        ],
    },
    {
        "name": "03 Get Configuration",
        "operationId": "getAssistantConfiguration",
        "tests": [
            *success_tests(),
            "pm.test('get returns created configuration', function () {",
            "  const data = jsonData.data || {};",
            "  pm.expect(data.id).to.eql(pm.collectionVariables.get('configurationId'));",
            "  pm.expect(data.assistantId).to.eql(pm.collectionVariables.get('assistantId'));",
            "  pm.expect(data.provider).to.eql('custom');",
            "});",
        ],
    },
    {
        "name": "04 List Configurations",
        "operationId": "getAllAssistantConfiguration",
        "tests": [
            *success_tests(),
            "pm.test('list includes created configuration', function () {",
            "  const ids = (jsonData.data || []).map(item => String(item.id));",
            "  pm.expect(ids).to.include(pm.collectionVariables.get('configurationId'));",
            "});",
        ],
    },
    {
        "name": "05 Update Configuration",
        "operationId": "updateAssistantConfiguration",
        "body": {
            "configurationType": "webhook",
            "provider": "custom-v2",
            "enabled": False,
            "options": [
                {"key": "url", "value": "https://example.com/webhook-v2"},
            ],
        },
        "tests": [
            *success_tests(),
            "pm.test('configuration is updated', function () {",
            "  const data = jsonData.data || {};",
            "  pm.expect(data.id).to.eql(pm.collectionVariables.get('configurationId'));",
            "  pm.expect(data.provider).to.eql('custom-v2');",
            "  pm.expect(data.enabled).to.eql(false);",
            "});",
        ],
    },
    {
        "name": "06 Get Updated Configuration",
        "operationId": "getAssistantConfiguration",
        "tests": [
            *success_tests(),
            "pm.test('get returns updated configuration', function () {",
            "  const data = jsonData.data || {};",
            "  pm.expect(data.id).to.eql(pm.collectionVariables.get('configurationId'));",
            "  pm.expect(data.provider).to.eql('custom-v2');",
            "  pm.expect(data.enabled).to.eql(false);",
            "});",
        ],
    },
    {
        "name": "07 Create API Deployment",
        "operationId": "createAssistantApiDeployment",
        "tests": [
            *success_tests(),
            "pm.test('api deployment is created', function () {",
            "  const data = jsonData.data || {};",
            "  pm.expect(data.assistantId).to.eql(pm.collectionVariables.get('assistantId'));",
            "  pm.expect(data.greeting).to.eql('Hello, how can I help you today?');",
            "});",
            "if (jsonData.data && jsonData.data.id) {",
            "  pm.collectionVariables.set('apiDeploymentId', String(jsonData.data.id));",
            "}",
        ],
    },
    {
        "name": "08 Get Latest API Deployment",
        "operationId": "getAssistantApiDeployment",
        "tests": [
            *success_tests(),
            "pm.test('latest api deployment matches created deployment', function () {",
            "  const data = jsonData.data || {};",
            "  pm.expect(data.id).to.eql(pm.collectionVariables.get('apiDeploymentId'));",
            "  pm.expect(data.assistantId).to.eql(pm.collectionVariables.get('assistantId'));",
            "});",
        ],
    },
    {
        "name": "09 List API Deployments",
        "operationId": "getAllAssistantApiDeployment",
        "tests": [
            *success_tests(),
            "pm.test('list includes created api deployment', function () {",
            "  const ids = (jsonData.data || []).map(item => String(item.id));",
            "  pm.expect(ids).to.include(pm.collectionVariables.get('apiDeploymentId'));",
            "});",
        ],
    },
    {
        "name": "10 Delete Configuration",
        "operationId": "deleteAssistantConfiguration",
        "tests": [
            *success_tests(),
            "pm.test('deleted configuration is returned', function () {",
            "  const data = jsonData.data || {};",
            "  pm.expect(data.id).to.eql(pm.collectionVariables.get('configurationId'));",
            "});",
        ],
    },
]


DEPLOYMENT_KIND_BY_OPERATION = {
    "Api": "API",
    "Debugger": "Debugger",
    "Phone": "Phone",
    "Webplugin": "Webplugin",
    "Whatsapp": "WhatsApp",
}

DEPLOYMENT_FOLDER_ORDER = ["API", "Debugger", "Phone", "Webplugin", "WhatsApp"]
REQUEST_ORDER = {
    "Create Assistant": 10,
    "Create Configuration": 10,
    "List Configurations": 20,
    "Get Configuration": 30,
    "Update Configuration": 40,
    "Delete Configuration": 50,
    "Create API Deployment": 10,
    "Get Latest API Deployment": 20,
    "List API Deployments": 30,
    "Create Debugger Deployment": 10,
    "Get Latest Debugger Deployment": 20,
    "List Debugger Deployments": 30,
    "Create Phone Deployment": 10,
    "Get Latest Phone Deployment": 20,
    "List Phone Deployments": 30,
    "Create Webplugin Deployment": 10,
    "Get Latest Webplugin Deployment": 20,
    "List Webplugin Deployments": 30,
    "Create WhatsApp Deployment": 10,
    "Get Latest WhatsApp Deployment": 20,
    "List WhatsApp Deployments": 30,
}


class OpenApiContext:
    def __init__(self, spec_path: Path):
        self.spec_path = spec_path
        self.spec_dir = spec_path.parent
        self.documents: dict[Path, dict[str, Any]] = {}
        self.root = self.load(spec_path)

    def load(self, path: Path) -> dict[str, Any]:
        path = path.resolve()
        if path not in self.documents:
            self.documents[path] = yaml.safe_load(path.read_text())
        return self.documents[path]

    def resolve_ref(self, ref: str, current_path: Path | None = None) -> Any:
        if current_path is None:
            current_path = self.spec_path
        doc_path = current_path
        fragment = ref
        if "#" in ref:
            file_part, fragment = ref.split("#", 1)
            if file_part:
                doc_path = (current_path.parent / file_part).resolve()
            fragment = "#" + fragment
        document = self.load(doc_path)
        if not fragment.startswith("#/"):
            raise ValueError(f"Unsupported $ref fragment: {ref}")
        value: Any = document
        for part in fragment[2:].split("/"):
            value = value[part]
        return copy.deepcopy(value)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--spec", type=Path, default=DEFAULT_SPEC)
    parser.add_argument("--collection-output", type=Path, default=DEFAULT_COLLECTION_OUTPUT)
    parser.add_argument("--smoke-output", type=Path, default=DEFAULT_SMOKE_OUTPUT)
    parser.add_argument(
        "--check",
        action="store_true",
        help="Exit non-zero if generated collections differ from output files.",
    )
    args = parser.parse_args()

    ctx = OpenApiContext(args.spec)
    collection = build_collection(ctx)
    smoke_collection = build_smoke_collection(ctx)
    rendered_outputs = {
        args.collection_output: json.dumps(collection, indent=2, ensure_ascii=False) + "\n",
        args.smoke_output: json.dumps(smoke_collection, indent=2, ensure_ascii=False) + "\n",
    }

    if args.check:
        for path, rendered in rendered_outputs.items():
            if not path.exists():
                print(f"{path} does not exist", file=sys.stderr)
                return 1
            current = path.read_text()
            if current != rendered:
                print(f"{path} is out of date", file=sys.stderr)
                return 1
            print(f"{path} is up to date")
        return 0

    for path, rendered in rendered_outputs.items():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(rendered)
        print(f"Wrote {path.relative_to(ROOT)}")
    return 0


def build_collection(ctx: OpenApiContext) -> dict[str, Any]:
    spec = ctx.root
    folders: dict[str, dict[str, Any]] = {
        "Assistant": {"name": "Assistant", "item": []},
        "Assistant Configurations": {"name": "Assistant Configurations", "item": []},
        "Assistant Deployments": {
            "name": "Assistant Deployments",
            "item": [
                {"name": f"{kind} Deployments", "item": []}
                for kind in DEPLOYMENT_FOLDER_ORDER
            ],
        },
    }
    deployment_folders = {
        folder["name"].replace(" Deployments", ""): folder
        for folder in folders["Assistant Deployments"]["item"]
    }

    for path, methods in spec["paths"].items():
        for method, operation in methods.items():
            item = build_item(ctx, path, method, operation)
            group = group_for(path, operation["operationId"])
            if group[0] == "Assistant Deployments":
                deployment_folders[group[1]]["item"].append(item)
            else:
                folders[group[0]]["item"].append(item)

    for folder in folders.values():
        sort_items(folder["item"])
    for folder in folders["Assistant Deployments"]["item"]:
        sort_items(folder["item"])

    return {
        "info": {
            "_postman_id": str(uuid.uuid5(uuid.NAMESPACE_URL, "rapida-assistant-api-postman")),
            "name": spec["info"]["title"],
            "description": (
                "Curated Postman collection generated from "
                "`openapi/artifacts/assistant-api.yaml`."
            ),
            "schema": POSTMAN_SCHEMA,
        },
        "auth": {
            "type": "apikey",
            "apikey": [
                {"key": "key", "value": "authorization", "type": "string"},
                {"key": "value", "value": "{{authToken}}", "type": "string"},
                {"key": "in", "value": "header", "type": "string"},
            ],
        },
        "event": [
            {
                "listen": "test",
                "script": {
                    "type": "text/javascript",
                    "exec": [
                        "pm.test('response is not a server error', function () {",
                        "  if (!pm.response || typeof pm.response.code !== 'number') {",
                        "    throw new Error('No HTTP response received');",
                        "  }",
                        "  pm.expect(pm.response.code).to.be.below(500);",
                        "});",
                    ],
                },
            }
        ],
        "variable": COLLECTION_VARIABLES,
        "item": [
            folders["Assistant"],
            folders["Assistant Configurations"],
            folders["Assistant Deployments"],
        ],
    }


def build_smoke_collection(ctx: OpenApiContext) -> dict[str, Any]:
    spec = ctx.root
    operation_index = index_operations(spec)
    return {
        "info": {
            "_postman_id": str(uuid.uuid5(uuid.NAMESPACE_URL, "rapida-assistant-api-smoke-postman")),
            "name": f"{spec['info']['title']} Smoke",
            "description": (
                "CI/CD smoke collection generated from "
                "`openapi/artifacts/assistant-api.yaml`. This collection intentionally "
                "contains only the ordered smoke flow."
            ),
            "schema": POSTMAN_SCHEMA,
        },
        "auth": {
            "type": "apikey",
            "apikey": [
                {"key": "key", "value": "authorization", "type": "string"},
                {"key": "value", "value": "{{authToken}}", "type": "string"},
                {"key": "in", "value": "header", "type": "string"},
            ],
        },
        "variable": COLLECTION_VARIABLES,
        "item": [
            {"name": "Smoke Flow", "item": build_smoke_flow(ctx, operation_index)},
        ],
    }


def index_operations(spec: dict[str, Any]) -> dict[str, tuple[str, str, dict[str, Any]]]:
    indexed = {}
    for path, methods in spec["paths"].items():
        for method, operation in methods.items():
            indexed[operation["operationId"]] = (path, method, operation)
    return indexed


def build_smoke_flow(
    ctx: OpenApiContext,
    operation_index: dict[str, tuple[str, str, dict[str, Any]]],
) -> list[dict[str, Any]]:
    items = []
    for step in SMOKE_FLOW_STEPS:
        path, method, operation = operation_index[step["operationId"]]
        item = build_item(ctx, path, method, operation)
        item["name"] = step["name"]
        item["event"] = [
            {
                "listen": "test",
                "script": {
                    "type": "text/javascript",
                    "exec": step["tests"],
                },
            }
        ]
        if "body" in step:
            item["request"]["body"] = {
                "mode": "raw",
                "raw": json.dumps(step["body"], indent=2),
                "options": {"raw": {"language": "json"}},
            }
        items.append(item)
    return items


def sort_items(items: list[dict[str, Any]]) -> None:
    items.sort(key=lambda item: (REQUEST_ORDER.get(item["name"], 999), item["name"]))


def group_for(path: str, operation_id: str) -> tuple[str, ...]:
    if path.startswith("/v1/assistant/configurations"):
        return ("Assistant Configurations",)
    if path.startswith("/v1/assistant-deployment"):
        return ("Assistant Deployments", deployment_kind(operation_id))
    return ("Assistant",)


def deployment_kind(operation_id: str) -> str:
    for marker, label in DEPLOYMENT_KIND_BY_OPERATION.items():
        if marker in operation_id:
            return label
    raise ValueError(f"Cannot determine deployment kind for {operation_id}")


def build_item(
    ctx: OpenApiContext, path: str, method: str, operation: dict[str, Any]
) -> dict[str, Any]:
    request: dict[str, Any] = {
        "method": method.upper(),
        "header": build_headers(operation),
        "url": build_url(path, operation),
        "description": build_description(operation),
    }

    body = request_body_example(ctx, operation)
    if body is not None:
        request["body"] = {
            "mode": "raw",
            "raw": json.dumps(body, indent=2),
            "options": {"raw": {"language": "json"}},
        }

    item: dict[str, Any] = {
        "name": request_name(path, operation["operationId"], operation.get("summary", "")),
        "request": request,
        "response": [],
    }

    events = capture_events(operation["operationId"])
    if events:
        item["event"] = events

    return item


def build_headers(operation: dict[str, Any]) -> list[dict[str, str]]:
    headers = [
        {"key": "Accept", "value": "application/json"},
        {"key": "x-auth-id", "value": "{{authId}}"},
        {"key": "x-project-id", "value": "{{projectId}}"},
    ]
    if operation.get("requestBody"):
        headers.append({"key": "Content-Type", "value": "application/json"})
    return headers


def build_url(path: str, operation: dict[str, Any]) -> dict[str, Any]:
    replaced_path = replace_path_variables(path)
    query = []
    for parameter in operation.get("parameters", []):
        if parameter.get("in") != "query":
            continue
        name = parameter["name"]
        value = query_value(name, parameter.get("schema", {}))
        query_parameter: dict[str, Any] = {
            "key": name,
            "value": value,
            "description": "Optional query parameter.",
        }
        if name not in {"page", "pageSize"}:
            query_parameter["disabled"] = True
        query.append(query_parameter)

    url = {
        "raw": "{{baseUrl}}" + replaced_path,
        "host": ["{{baseUrl}}"],
        "path": [segment for segment in replaced_path.strip("/").split("/") if segment],
    }
    if query:
        url["query"] = query
    return url


def replace_path_variables(path: str) -> str:
    return path.replace("{assistantId}", "{{assistantId}}").replace(
        "{id}", "{{configurationId}}"
    )


def query_value(name: str, schema: dict[str, Any]) -> str:
    if name == "page":
        return "{{page}}"
    if name == "pageSize":
        return "{{pageSize}}"
    if name == "criterias":
        return '[{"key":"status","value":"active","logic":"equal"}]'
    if "default" in schema:
        return str(schema["default"])
    if name == "configurationType":
        return "webhook"
    if name == "provider":
        return "custom"
    return sample_scalar(schema, name)


def build_description(operation: dict[str, Any]) -> str:
    lines = []
    if operation.get("description"):
        lines.append(operation["description"])
    elif operation.get("summary"):
        lines.append(operation["summary"])
    lines.append(f"OpenAPI operationId: `{operation['operationId']}`.")
    return "\n\n".join(lines)


def request_name(path: str, operation_id: str, summary: str) -> str:
    if operation_id == "createAssistant":
        return "Create Assistant"
    if operation_id == "createAssistantConfiguration":
        return "Create Configuration"
    if operation_id == "getAllAssistantConfiguration":
        return "List Configurations"
    if operation_id == "getAssistantConfiguration":
        return "Get Configuration"
    if operation_id == "updateAssistantConfiguration":
        return "Update Configuration"
    if operation_id == "deleteAssistantConfiguration":
        return "Delete Configuration"

    if path.startswith("/v1/assistant-deployment"):
        kind = deployment_kind(operation_id)
        plural = "WhatsApp Deployments" if kind == "WhatsApp" else f"{kind} Deployments"
        if operation_id.startswith("create"):
            return f"Create {kind} Deployment"
        if operation_id.startswith("getAll"):
            return f"List {plural}"
        return f"Get Latest {kind} Deployment"

    return summary or operation_id


def request_body_example(ctx: OpenApiContext, operation: dict[str, Any]) -> dict[str, Any] | None:
    body = operation.get("requestBody")
    if not body:
        return None
    content = body.get("content", {})
    json_content = content.get("application/json")
    if not json_content:
        return None
    schema = json_content.get("schema", {})
    schema_name = ref_name(schema.get("$ref", ""))
    if schema_name in REQUEST_EXAMPLES:
        return copy.deepcopy(REQUEST_EXAMPLES[schema_name])
    return sample_value(ctx, schema, schema_name or "body")


def ref_name(ref: str) -> str:
    if not ref:
        return ""
    return ref.rsplit("/", 1)[-1]


def sample_value(
    ctx: OpenApiContext,
    schema: dict[str, Any],
    name: str = "",
    depth: int = 0,
) -> Any:
    if depth > 6:
        return None
    if "$ref" in schema:
        return sample_value(ctx, ctx.resolve_ref(schema["$ref"]), ref_name(schema["$ref"]), depth + 1)
    if "default" in schema:
        return schema["default"]
    if "enum" in schema and schema["enum"]:
        return schema["enum"][0]

    schema_type = schema.get("type")
    if schema_type == "object" or "properties" in schema:
        properties = schema.get("properties", {})
        if properties:
            return {
                property_name: sample_value(ctx, property_schema, property_name, depth + 1)
                for property_name, property_schema in properties.items()
            }
        return {"key": "value"}
    if schema_type == "array":
        return [sample_value(ctx, schema.get("items", {}), singular(name), depth + 1)]
    if schema_type == "boolean":
        return True
    if schema_type == "integer":
        return sample_integer(schema, name)
    if schema_type == "number":
        return sample_number(schema, name)
    return sample_scalar(schema, name)


def sample_scalar(schema: dict[str, Any], name: str) -> str:
    normalized = name.lower()
    if normalized.endswith("id") or normalized == "id":
        return "1"
    if "url" in normalized:
        return "https://example.com"
    if "email" in normalized:
        return "user@example.com"
    if "provider" in normalized:
        return "custom"
    if "type" in normalized:
        return "webhook"
    if "status" in normalized:
        return "active"
    if schema.get("format") == "date-time":
        return "2026-01-01T00:00:00Z"
    return f"sample-{name or 'value'}"


def sample_integer(schema: dict[str, Any], name: str) -> int:
    normalized = name.lower()
    if normalized == "page":
        return 1
    if normalized == "pagesize":
        return 20
    if normalized == "idealtimeout":
        return 30
    if normalized == "idealtimeoutbackoff":
        return 2
    if normalized == "maxsessionduration":
        return 300
    if "minimum" in schema:
        return int(schema["minimum"])
    return 1


def sample_number(schema: dict[str, Any], name: str) -> float:
    if name.lower() == "unclearinputtimeout":
        return 4
    if "minimum" in schema:
        return float(schema["minimum"])
    return 1.0


def singular(name: str) -> str:
    return name[:-1] if name.endswith("s") else name


def capture_events(operation_id: str) -> list[dict[str, Any]]:
    capture_map = {
        "createAssistant": ("assistantId", "assistant id"),
        "createAssistantConfiguration": ("configurationId", "configuration id"),
    }
    if operation_id not in capture_map:
        return []
    variable, label = capture_map[operation_id]
    return [
        {
            "listen": "test",
            "script": {
                "type": "text/javascript",
                "exec": [
                    "let jsonData = {};",
                    "try { jsonData = pm.response.json(); } catch (e) {}",
                    "if (jsonData && jsonData.data && jsonData.data.id) {",
                    f"  pm.collectionVariables.set('{variable}', String(jsonData.data.id));",
                    f"  console.log('Captured {label}: ' + jsonData.data.id);",
                    "}",
                ],
            },
        }
    ]


if __name__ == "__main__":
    raise SystemExit(main())

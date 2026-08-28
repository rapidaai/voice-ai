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
        "provider": "http",
        "enabled": True,
        "options": [
            {"key": "http_method", "value": "POST"},
            {"key": "http_url", "value": "https://api.example.com/webhook"},
            {
                "key": "http_headers",
                "value": '{"Authorization":"Bearer {{webhookToken}}","X-Webhook-Source":"postman"}',
            },
            {"key": "retry_status_codes", "value": '["50X"]'},
            {"key": "max_retry_count", "value": "2"},
            {"key": "timeout_seconds", "value": "220"},
            {"key": "assistant_events", "value": '["webrtc.connected"]'},
            {"key": "execution_priority", "value": "4"},
            {"key": "description", "value": "Postman webhook configuration"},
        ],
    },
    "UpdateAssistantConfigurationRequest": {
        "configurationType": "webhook",
        "provider": "http",
        "enabled": True,
        "options": [
            {"key": "http_method", "value": "POST"},
            {"key": "http_url", "value": "https://hooks.example.com/incoming"},
            {"key": "http_headers", "value": '{"X-Webhook-Version":"2"}'},
            {"key": "retry_status_codes", "value": '["40X","50X"]'},
            {"key": "max_retry_count", "value": "3"},
            {"key": "timeout_seconds", "value": "180"},
            {
                "key": "assistant_events",
                "value": '["conversation.begin","webrtc.reconnecting"]',
            },
            {"key": "execution_priority", "value": "1"},
            {"key": "description", "value": "Updated Postman webhook configuration"},
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


CONFIGURATION_VARIANTS: list[dict[str, Any]] = [
    {
        "label": "Webhook",
        "configurationType": "webhook",
        "provider": "http",
        "variable": "webhookConfigurationId",
        "create": copy.deepcopy(REQUEST_EXAMPLES["CreateAssistantConfigurationRequest"]),
        "update": copy.deepcopy(REQUEST_EXAMPLES["UpdateAssistantConfigurationRequest"]),
    },
    {
        "label": "Storage",
        "configurationType": "storage",
        "provider": "aws",
        "variable": "storageConfigurationId",
        "create": {
            "assistantId": "{{assistantId}}",
            "configurationType": "storage",
            "provider": "aws",
            "enabled": True,
            "options": [
                {"key": "rapida.credential_id", "value": "{{storageCredentialId}}"},
                {"key": "s3_bucket_name", "value": "rapida-assistant-recordings"},
                {"key": "prefix", "value": "postman/recordings"},
                {
                    "key": "files_to_push",
                    "value": '["recording.conversation","recording.user","recording.assistant"]',
                },
            ],
        },
        "update": {
            "configurationType": "storage",
            "provider": "aws",
            "enabled": True,
            "options": [
                {"key": "rapida.credential_id", "value": "{{storageCredentialId}}"},
                {"key": "s3_bucket_name", "value": "rapida-assistant-recordings-archive"},
                {"key": "prefix", "value": "postman/archive"},
                {
                    "key": "files_to_push",
                    "value": '["recording.conversation","recording.assistant"]',
                },
            ],
        },
    },
    {
        "label": "Authentication",
        "configurationType": "authentication",
        "provider": "http",
        "variable": "authenticationConfigurationId",
        "create": {
            "assistantId": "{{assistantId}}",
            "configurationType": "authentication",
            "provider": "http",
            "enabled": True,
            "options": [
                {"key": "http_method", "value": "POST"},
                {"key": "http_url", "value": "https://auth.example.com/resolve"},
                {"key": "http_headers", "value": '{"X-Auth-Source":"postman"}'},
                {
                    "key": "http_body",
                    "value": '{"assistant.id":"assistantId","client.phone":"clientPhone"}',
                },
                {"key": "fail_behavior", "value": "BLOCK"},
                {"key": "timeout_ms", "value": "5000"},
                {
                    "key": "authentication.condition",
                    "value": '[{"key":"source","condition":"=","value":"all"}]',
                },
            ],
        },
        "update": {
            "configurationType": "authentication",
            "provider": "http",
            "enabled": True,
            "options": [
                {"key": "http_method", "value": "POST"},
                {"key": "http_url", "value": "https://auth.example.com/v2/resolve"},
                {
                    "key": "http_headers",
                    "value": '{"X-Auth-Source":"postman","X-Auth-Version":"2"}',
                },
                {
                    "key": "http_body",
                    "value": '{"assistant.id":"assistantId","client.phone":"clientPhone","conversation.id":"conversationId"}',
                },
                {"key": "fail_behavior", "value": "DO_NOTHING"},
                {"key": "timeout_ms", "value": "7500"},
                {
                    "key": "authentication.condition",
                    "value": '[{"key":"source","condition":"=","value":"phone"}]',
                },
            ],
        },
    },
    {
        "label": "Analysis",
        "configurationType": "analysis",
        "provider": "endpoint",
        "variable": "analysisConfigurationId",
        "create": {
            "assistantId": "{{assistantId}}",
            "configurationType": "analysis",
            "provider": "endpoint",
            "enabled": True,
            "options": [
                {"key": "name", "value": "Post-call conversation analysis"},
                {"key": "description", "value": "Summarize the completed conversation"},
                {"key": "execution_priority", "value": "0"},
                {"key": "endpoint_id", "value": "{{analysisEndpointId}}"},
                {"key": "endpoint_version", "value": "latest"},
                {
                    "key": "endpoint_parameters",
                    "value": '{"conversation.messages":"messages","assistant.name":"assistantName"}',
                },
                {
                    "key": "analysis.condition",
                    "value": '[{"key":"conversation_mode","condition":"=","value":"voice"}]',
                },
            ],
        },
        "update": {
            "configurationType": "analysis",
            "provider": "endpoint",
            "enabled": True,
            "options": [
                {"key": "name", "value": "Updated conversation analysis"},
                {"key": "description", "value": "Analyze voice and text conversations"},
                {"key": "execution_priority", "value": "3"},
                {"key": "endpoint_id", "value": "{{analysisEndpointId}}"},
                {"key": "endpoint_version", "value": "latest"},
                {
                    "key": "endpoint_parameters",
                    "value": '{"conversation.id":"conversationId","assistant.name":"assistantName"}',
                },
                {
                    "key": "analysis.condition",
                    "value": '[{"key":"conversation_mode","condition":"=","value":"text"}]',
                },
            ],
        },
    },
]


COLLECTION_VARIABLES: list[dict[str, str]] = [
    {"key": "baseUrl", "value": "http://localhost:9007", "type": "string"},
    {"key": "apiKey", "value": "", "type": "string"},
    {"key": "authToken", "value": "", "type": "string"},
    {"key": "authId", "value": "", "type": "string"},
    {"key": "projectId", "value": "", "type": "string"},
    {"key": "assistantId", "value": "1", "type": "string"},
    {"key": "webhookConfigurationId", "value": "1", "type": "string"},
    {"key": "storageConfigurationId", "value": "1", "type": "string"},
    {"key": "authenticationConfigurationId", "value": "1", "type": "string"},
    {"key": "analysisConfigurationId", "value": "1", "type": "string"},
    {"key": "apiDeploymentId", "value": "1", "type": "string"},
    {"key": "debuggerDeploymentId", "value": "1", "type": "string"},
    {"key": "phoneDeploymentId", "value": "1", "type": "string"},
    {"key": "webpluginDeploymentId", "value": "1", "type": "string"},
    {"key": "whatsappDeploymentId", "value": "1", "type": "string"},
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
    {"key": "webhookToken", "value": "replace-with-webhook-token", "type": "string"},
    {"key": "storageCredentialId", "value": "1", "type": "string"},
    {"key": "analysisEndpointId", "value": "1", "type": "string"},
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


def configuration_option_tests(body: dict[str, Any], message: str) -> list[str]:
    expected_options = {
        option["key"]: option["value"] for option in body.get("options", [])
    }
    return [
        f"pm.test({json.dumps(message)}, function () {{",
        "  const data = jsonData.data || {};",
        "  const optionMap = new Map((data.options || []).map(option => [option.key, option.value]));",
        f"  const expectedOptions = {json.dumps(expected_options)};",
        "  Object.entries(expectedOptions).forEach(function ([key, value]) {",
        "    pm.expect(optionMap.get(key), key).to.eql(pm.variables.replaceIn(value));",
        "  });",
        "});",
    ]


def configuration_smoke_steps(
    start: int,
    variant: dict[str, Any],
) -> list[dict[str, Any]]:
    label = variant["label"]
    configuration_type = variant["configurationType"]
    provider = variant["provider"]
    variable = variant["variable"]
    create_body = variant["create"]
    update_body = variant["update"]
    return [
        {
            "name": f"{start:02d} Create {label} Configuration",
            "operationId": "createAssistantConfiguration",
            "body": create_body,
            "tests": [
                *success_tests(),
                f"pm.test('{label.lower()} configuration is created', function () {{",
                "  const data = jsonData.data || {};",
                "  pm.expect(data.assistantId).to.eql(pm.collectionVariables.get('assistantId'));",
                f"  pm.expect(data.configurationType).to.eql('{configuration_type}');",
                f"  pm.expect(data.provider).to.eql('{provider}');",
                "  pm.expect(data.enabled).to.eql(true);",
                "  pm.expect(data.id).to.exist;",
                "});",
                *configuration_option_tests(
                    create_body,
                    f"{label.lower()} create options match the UI payload",
                ),
                "if (jsonData.data && jsonData.data.id) {",
                f"  pm.collectionVariables.set('{variable}', String(jsonData.data.id));",
                "}",
            ],
        },
        {
            "name": f"{start + 1:02d} Get {label} Configuration",
            "operationId": "getAssistantConfiguration",
            "configurationVariable": variable,
            "tests": [
                *success_tests(),
                f"pm.test('get returns created {label.lower()} configuration', function () {{",
                "  const data = jsonData.data || {};",
                f"  pm.expect(data.id).to.eql(pm.collectionVariables.get('{variable}'));",
                "  pm.expect(data.assistantId).to.eql(pm.collectionVariables.get('assistantId'));",
                f"  pm.expect(data.configurationType).to.eql('{configuration_type}');",
                f"  pm.expect(data.provider).to.eql('{provider}');",
                "});",
                *configuration_option_tests(
                    create_body,
                    f"{label.lower()} get returns the create options",
                ),
            ],
        },
        {
            "name": f"{start + 2:02d} Update {label} Configuration",
            "operationId": "updateAssistantConfiguration",
            "configurationVariable": variable,
            "body": update_body,
            "tests": [
                *success_tests(),
                f"pm.test('{label.lower()} configuration is updated', function () {{",
                "  const data = jsonData.data || {};",
                f"  pm.expect(data.id).to.eql(pm.collectionVariables.get('{variable}'));",
                f"  pm.expect(data.configurationType).to.eql('{configuration_type}');",
                f"  pm.expect(data.provider).to.eql('{provider}');",
                "  pm.expect(data.enabled).to.eql(true);",
                "});",
                *configuration_option_tests(
                    update_body,
                    f"{label.lower()} update options match the UI payload",
                ),
            ],
        },
        {
            "name": f"{start + 3:02d} Get Updated {label} Configuration",
            "operationId": "getAssistantConfiguration",
            "configurationVariable": variable,
            "tests": [
                *success_tests(),
                f"pm.test('get returns updated {label.lower()} configuration', function () {{",
                "  const data = jsonData.data || {};",
                f"  pm.expect(data.id).to.eql(pm.collectionVariables.get('{variable}'));",
                f"  pm.expect(data.configurationType).to.eql('{configuration_type}');",
                f"  pm.expect(data.provider).to.eql('{provider}');",
                "});",
                *configuration_option_tests(
                    update_body,
                    f"{label.lower()} get returns the updated options",
                ),
            ],
        },
    ]


def delete_configuration_smoke_step(
    number: int,
    variant: dict[str, Any],
) -> dict[str, Any]:
    label = variant["label"]
    variable = variant["variable"]
    return {
        "name": f"{number:02d} Delete {label} Configuration",
        "operationId": "deleteAssistantConfiguration",
        "configurationVariable": variable,
        "tests": [
            *success_tests(),
            f"pm.test('deleted {label.lower()} configuration is returned', function () {{",
            "  const data = jsonData.data || {};",
            f"  pm.expect(data.id).to.eql(pm.collectionVariables.get('{variable}'));",
            "});",
        ],
    }


def deployment_smoke_steps(
    start: int,
    label: str,
    operation_fragment: str,
    variable: str,
) -> list[dict[str, Any]]:
    deployment_name = label.lower()
    return [
        {
            "name": f"{start:02d} Create {label} Deployment",
            "operationId": f"createAssistant{operation_fragment}Deployment",
            "tests": [
                *success_tests(),
                f"pm.test('{deployment_name} deployment is created', function () {{",
                "  const data = jsonData.data || {};",
                "  pm.expect(data.assistantId).to.eql(pm.collectionVariables.get('assistantId'));",
                "  pm.expect(data.id).to.exist;",
                "});",
                "if (jsonData.data && jsonData.data.id) {",
                f"  pm.collectionVariables.set('{variable}', String(jsonData.data.id));",
                "}",
            ],
        },
        {
            "name": f"{start + 1:02d} Get Latest {label} Deployment",
            "operationId": f"getAssistant{operation_fragment}Deployment",
            "tests": [
                *success_tests(),
                f"pm.test('latest {deployment_name} deployment matches created deployment', function () {{",
                "  const data = jsonData.data || {};",
                f"  pm.expect(data.id).to.eql(pm.collectionVariables.get('{variable}'));",
                "  pm.expect(data.assistantId).to.eql(pm.collectionVariables.get('assistantId'));",
                "});",
            ],
        },
        {
            "name": f"{start + 2:02d} List {label} Deployments",
            "operationId": f"getAllAssistant{operation_fragment}Deployment",
            "tests": [
                *success_tests(),
                f"pm.test('list includes created {deployment_name} deployment', function () {{",
                "  const ids = (jsonData.data || []).map(item => String(item.id));",
                f"  pm.expect(ids).to.include(pm.collectionVariables.get('{variable}'));",
                "});",
            ],
        },
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
    *configuration_smoke_steps(2, CONFIGURATION_VARIANTS[0]),
    *configuration_smoke_steps(6, CONFIGURATION_VARIANTS[1]),
    *configuration_smoke_steps(10, CONFIGURATION_VARIANTS[2]),
    *configuration_smoke_steps(14, CONFIGURATION_VARIANTS[3]),
    {
        "name": "18 List Configurations",
        "operationId": "getAllAssistantConfiguration",
        "tests": [
            *success_tests(),
            "pm.test('list includes all created configurations', function () {",
            "  const ids = (jsonData.data || []).map(item => String(item.id));",
            *[
                f"  pm.expect(ids).to.include(pm.collectionVariables.get('{variant['variable']}'));"
                for variant in CONFIGURATION_VARIANTS
            ],
            "});",
        ],
    },
    *deployment_smoke_steps(19, "API", "Api", "apiDeploymentId"),
    *deployment_smoke_steps(22, "Debugger", "Debugger", "debuggerDeploymentId"),
    *deployment_smoke_steps(25, "Phone", "Phone", "phoneDeploymentId"),
    *deployment_smoke_steps(28, "Webplugin", "Webplugin", "webpluginDeploymentId"),
    *deployment_smoke_steps(31, "WhatsApp", "Whatsapp", "whatsappDeploymentId"),
    *[
        delete_configuration_smoke_step(34 + index, variant)
        for index, variant in enumerate(CONFIGURATION_VARIANTS)
    ],
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
    operation_index = index_operations(spec)
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
            elif (
                group[0] == "Assistant Configurations"
                and operation["operationId"] != "getAllAssistantConfiguration"
            ):
                continue
            else:
                folders[group[0]]["item"].append(item)

    for folder in folders.values():
        sort_items(folder["item"])
    for folder in folders["Assistant Deployments"]["item"]:
        sort_items(folder["item"])
    folders["Assistant Configurations"]["item"].extend(
        build_configuration_collection_folders(ctx, operation_index)
    )

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
                {"key": "key", "value": "x-api-key", "type": "string"},
                {"key": "value", "value": "{{apiKey}}", "type": "string"},
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
                {"key": "key", "value": "x-api-key", "type": "string"},
                {"key": "value", "value": "{{apiKey}}", "type": "string"},
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
        if "configurationVariable" in step:
            replace_item_variable(
                item,
                "configurationId",
                step["configurationVariable"],
            )
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


def build_configuration_collection_folders(
    ctx: OpenApiContext,
    operation_index: dict[str, tuple[str, str, dict[str, Any]]],
) -> list[dict[str, Any]]:
    folders = []
    operation_ids = [
        "createAssistantConfiguration",
        "getAssistantConfiguration",
        "updateAssistantConfiguration",
        "deleteAssistantConfiguration",
    ]
    for variant in CONFIGURATION_VARIANTS:
        items = []
        for operation_id in operation_ids:
            path, method, operation = operation_index[operation_id]
            item = build_item(ctx, path, method, operation)
            action = {
                "createAssistantConfiguration": "Create",
                "getAssistantConfiguration": "Get",
                "updateAssistantConfiguration": "Update",
                "deleteAssistantConfiguration": "Delete",
            }[operation_id]
            item["name"] = f"{action} {variant['label']} Configuration"
            replace_item_variable(item, "configurationId", variant["variable"])
            if operation_id == "createAssistantConfiguration":
                set_item_body(item, variant["create"])
                item["event"] = capture_variable_event(
                    variant["variable"],
                    f"{variant['label'].lower()} configuration id",
                )
            elif operation_id == "updateAssistantConfiguration":
                set_item_body(item, variant["update"])
            items.append(item)
        folders.append({"name": variant["label"], "item": items})
    return folders


def set_item_body(item: dict[str, Any], body: dict[str, Any]) -> None:
    item["request"]["body"] = {
        "mode": "raw",
        "raw": json.dumps(body, indent=2),
        "options": {"raw": {"language": "json"}},
    }


def replace_item_variable(item: Any, old: str, new: str) -> None:
    if isinstance(item, dict):
        for key, value in item.items():
            if isinstance(value, str):
                item[key] = value.replace(f"{{{{{old}}}}}", f"{{{{{new}}}}}")
            else:
                replace_item_variable(value, old, new)
    elif isinstance(item, list):
        for index, value in enumerate(item):
            if isinstance(value, str):
                item[index] = value.replace(f"{{{{{old}}}}}", f"{{{{{new}}}}}")
            else:
                replace_item_variable(value, old, new)


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
        {"key": "authorization", "value": "{{authToken}}"},
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
        return "http"
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
    return capture_variable_event(variable, label)


def capture_variable_event(variable: str, label: str) -> list[dict[str, Any]]:
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

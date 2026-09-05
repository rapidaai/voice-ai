require("../../react-environment");

const {
  ConnectionConfig,
  CreateProviderCredentialRequest,
  CreateProviderKey,
} = require("@rapidaai/react");
const { Struct } = require("google-protobuf/google/protobuf/struct_pb");

const webEndpoint = process.env.WEB_API_REACT_ENDPOINT || "http://web-api:9001";

function requireValue(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

function requireSuccess(response, action) {
  if (!response?.getSuccess() || response.getCode() !== 200) {
    const message = response?.getError()?.getHumanmessage() || "unknown error";
    throw new Error(`${action} failed with code ${response?.getCode()}: ${message}`);
  }
}

async function main() {
  const vaultName = requireValue("FLOW_VAULT_NAME");
  const auth = ConnectionConfig.WithPersonalToken({
    Authorization: requireValue("FLOW_TOKEN"),
    AuthId: requireValue("FLOW_FIXTURE_ID"),
    ProjectId: requireValue("FLOW_PROJECT_ID"),
  });
  const connection = new ConnectionConfig({ web: webEndpoint }, true);
  const request = new CreateProviderCredentialRequest();
  request.setProvider("openai:OpenAI");
  request.setName(vaultName);
  request.setCredential(Struct.fromJavaScript({ api_key: "ci-openai-key" }));

  const response = await CreateProviderKey(connection, request, auth);
  requireSuccess(response, "vault creation");

  const credential = response.getData();
  if (!credential || credential.getName() !== vaultName) {
    throw new Error("vault creation returned unexpected data");
  }
  if (credential.getProvider() !== "openai:OpenAI") {
    throw new Error("vault creation returned the wrong provider");
  }
  if (credential.getValue()?.toJavaScript().api_key !== "ci-openai-key") {
    throw new Error("vault creation returned the wrong credential value");
  }

  console.log("React SDK create vault flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

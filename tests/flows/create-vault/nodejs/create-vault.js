const {
  ConnectionConfig,
  CreateProviderKey,
} = require("@rapidaai/nodejs");

const webEndpoint = process.env.WEB_API_ENDPOINT || "web-api:9001";

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
  const response = await CreateProviderKey(
    connection,
    "openai",
    "OpenAI",
    { api_key: "ci-openai-key" },
    vaultName,
    auth,
  );
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

  console.log("Node SDK create vault flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

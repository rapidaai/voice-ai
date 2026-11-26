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

async function expectUnavailable(action, invoke) {
  try {
    await invoke();
    throw new Error(`${action} unexpectedly succeeded`);
  } catch (error) {
    if (error?.message === `${action} unexpectedly succeeded`) {
      throw error;
    }
    if (error?.code === 14) {
      return;
    }
    throw new Error(`${action} returned an unexpected error`, { cause: error });
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
  await expectUnavailable("vault creation", () =>
    CreateProviderKey(
      connection,
      "openai",
      "OpenAI",
      { api_key: "ci-openai-key" },
      vaultName,
      auth,
    ),
  );

  console.log("Node SDK vault creation policy flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

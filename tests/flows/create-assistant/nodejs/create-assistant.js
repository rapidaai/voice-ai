const {
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
} = require("@rapidaai/nodejs");

const assistantEndpoint =
  process.env.ASSISTANT_API_ENDPOINT || "assistant-api:9007";

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
    throw new Error(
      `${action} failed with code ${response?.getCode()}: ${message}`,
    );
  }
}

function authenticationModes() {
  return [
    {
      name: "personal access token",
      auth: ConnectionConfig.WithPersonalToken({
        Authorization: requireValue("FLOW_TOKEN"),
        AuthId: requireValue("FLOW_FIXTURE_ID"),
        ProjectId: requireValue("FLOW_PROJECT_ID"),
      }),
    },
    {
      name: "project API key",
      auth: ConnectionConfig.WithSDK({
        ApiKey: requireValue("FLOW_API_KEY"),
        UserId: "",
      }),
    },
  ];
}

async function createAssistant(mode) {
  const assistantName = `${requireValue("FLOW_ASSISTANT")} ${mode.name}`;
  const auth = mode.auth;
  const connection = ConnectionConfig.DefaultConnectionConfig(
    auth,
  ).withCustomEndpoint({ assistant: assistantEndpoint }, true);

  const providerModel =
    new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  providerModel.setModelprovidername("openai");
  const provider = new CreateAssistantProviderRequest();
  provider.setDescription("CI default model provider");
  provider.setModel(providerModel);

  const request = new CreateAssistantRequest();
  request.setName(assistantName);
  request.setDescription("Created by the CI Node SDK flow");
  request.setVisibility("private");
  request.setLanguage("english");
  request.setAssistantprovider(provider);

  const response = await CreateAssistant(connection, request, auth);
  requireSuccess(response, "assistant creation");

  const assistant = response.getData();
  if (!assistant || assistant.getName() !== assistantName) {
    throw new Error("assistant creation returned unexpected data");
  }
  if (assistant.getProjectid() !== requireValue("FLOW_PROJECT_ID")) {
    throw new Error("assistant creation returned the wrong project");
  }
  if (
    !assistant.getId() ||
    assistant.getOrganizationid() !== requireValue("FLOW_FIXTURE_ID")
  ) {
    throw new Error("assistant creation returned incomplete ownership data");
  }

  console.log(`Node SDK create assistant flow passed with ${mode.name}`);
}

async function main() {
  for (const mode of authenticationModes()) {
    await createAssistant(mode);
  }
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

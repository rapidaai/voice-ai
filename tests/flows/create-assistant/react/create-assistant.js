require("../../react-environment");

const {
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
} = require("@rapidaai/react");

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
  const assistantName = requireValue("FLOW_ASSISTANT");
  const auth = ConnectionConfig.WithPersonalToken({
    Authorization: requireValue("FLOW_TOKEN"),
    AuthId: requireValue("FLOW_FIXTURE_ID"),
    ProjectId: requireValue("FLOW_PROJECT_ID"),
  });
  const connection = new ConnectionConfig({ web: webEndpoint }, true);

  const providerModel = new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  providerModel.setModelprovidername("openai");
  const provider = new CreateAssistantProviderRequest();
  provider.setDescription("CI default model provider");
  provider.setModel(providerModel);

  const request = new CreateAssistantRequest();
  request.setName(assistantName);
  request.setDescription("Created by the CI React SDK flow");
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
  if (!assistant.getId() || assistant.getOrganizationid() !== requireValue("FLOW_FIXTURE_ID")) {
    throw new Error("assistant creation returned incomplete ownership data");
  }

  console.log("React SDK create assistant flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

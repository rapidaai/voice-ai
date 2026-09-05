require("../../react-environment");

const {
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantProvider,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
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
    throw new Error(
      `${action} failed with code ${response?.getCode()}: ${message}`,
    );
  }
}

async function createAssistant(connection, auth, assistantName) {
  const providerModel =
    new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  providerModel.setModelprovidername("openai");
  const provider = new CreateAssistantProviderRequest();
  provider.setDescription("CI default model provider");
  provider.setModel(providerModel);

  const request = new CreateAssistantRequest();
  request.setName(assistantName);
  request.setVisibility("private");
  request.setLanguage("english");
  request.setAssistantprovider(provider);

  const response = await CreateAssistant(connection, request, auth);
  requireSuccess(response, "assistant creation");
  const assistant = response.getData();
  if (
    !assistant?.getId() ||
    assistant.getOrganizationid() !== requireValue("FLOW_FIXTURE_ID") ||
    assistant.getProjectid() !== requireValue("FLOW_PROJECT_ID")
  ) {
    throw new Error("assistant creation returned unexpected ownership data");
  }
  return assistant.getId();
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

async function runFlow(mode) {
  const auth = mode.auth;
  const connection = new ConnectionConfig({ web: webEndpoint }, true);
  const assistantID = await createAssistant(
    connection,
    auth,
    `${requireValue("FLOW_ASSISTANT")} ${mode.name}`,
  );
  if (!assistantID) {
    throw new Error("assistant creation returned no identifier");
  }

  const model =
    new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  model.setModelprovidername("anthropic");
  const modelRequest = new CreateAssistantProviderRequest();
  modelRequest.setAssistantid(assistantID);
  modelRequest.setDescription("CI secondary model provider");
  modelRequest.setModel(model);
  const modelResponse = await CreateAssistantProvider(
    connection,
    modelRequest,
    auth,
  );
  requireSuccess(modelResponse, "model provider creation");

  const agentkit =
    new CreateAssistantProviderRequest.CreateAssistantProviderAgentkit();
  agentkit.setAgentkiturl("agentkit:50051");
  agentkit.setTransportsecurity("PLAINTEXT");
  const agentkitRequest = new CreateAssistantProviderRequest();
  agentkitRequest.setAssistantid(assistantID);
  agentkitRequest.setDescription("CI AgentKit provider");
  agentkitRequest.setAgentkit(agentkit);
  const agentkitResponse = await CreateAssistantProvider(
    connection,
    agentkitRequest,
    auth,
  );
  requireSuccess(agentkitResponse, "AgentKit provider creation");

  const websocket =
    new CreateAssistantProviderRequest.CreateAssistantProviderWebsocket();
  websocket.setWebsocketurl("wss://example.invalid/agent");
  const websocketRequest = new CreateAssistantProviderRequest();
  websocketRequest.setAssistantid(assistantID);
  websocketRequest.setDescription("CI WebSocket provider");
  websocketRequest.setWebsocket(websocket);
  const websocketResponse = await CreateAssistantProvider(
    connection,
    websocketRequest,
    auth,
  );
  requireSuccess(websocketResponse, "WebSocket provider creation");

  const agentflow =
    new CreateAssistantProviderRequest.CreateAssistantProviderAgentflow();
  agentflow.setSchemaversion("1.0");
  agentflow.setDefinition(
    Struct.fromJavaScript({
      entryNodeId: "start",
      nodes: [{ id: "start" }],
      edges: [],
    }),
  );
  const agentflowRequest = new CreateAssistantProviderRequest();
  agentflowRequest.setAssistantid(assistantID);
  agentflowRequest.setDescription("CI AgentFlow provider");
  agentflowRequest.setAgentflow(agentflow);
  const agentflowResponse = await CreateAssistantProvider(
    connection,
    agentflowRequest,
    auth,
  );
  requireSuccess(agentflowResponse, "AgentFlow provider creation");

  console.log(
    `React SDK create assistant provider flow passed with ${mode.name}`,
  );
}

async function main() {
  for (const mode of authenticationModes()) {
    await runFlow(mode);
  }
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

const {
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  WithAuthContext,
} = require("@rapidaai/nodejs");
const { Struct } = require("google-protobuf/google/protobuf/struct_pb");

const assistantEndpoint = process.env.ASSISTANT_API_ENDPOINT || "assistant-api:9007";

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

function createProvider(connection, request, auth) {
  return new Promise((resolve, reject) => {
    connection.assistantClient.createAssistantProvider(
      request,
      WithAuthContext(auth),
      (error, response) => {
        if (error) {
          reject(error);
          return;
        }
        resolve(response);
      },
    );
  });
}

async function createAssistant(connection, auth) {
  const providerModel = new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  providerModel.setModelprovidername("openai");
  const provider = new CreateAssistantProviderRequest();
  provider.setDescription("CI default model provider");
  provider.setModel(providerModel);

  const request = new CreateAssistantRequest();
  request.setName(requireValue("FLOW_ASSISTANT"));
  request.setVisibility("private");
  request.setLanguage("english");
  request.setAssistantprovider(provider);

  const response = await CreateAssistant(connection, request, auth);
  requireSuccess(response, "assistant creation");
  return response.getData()?.getId();
}

async function main() {
  const auth = ConnectionConfig.WithPersonalToken({
    Authorization: requireValue("FLOW_TOKEN"),
    AuthId: requireValue("FLOW_FIXTURE_ID"),
    ProjectId: requireValue("FLOW_PROJECT_ID"),
  });
  const connection = ConnectionConfig.DefaultConnectionConfig(auth)
    .withCustomEndpoint({ assistant: assistantEndpoint }, true);
  const assistantID = await createAssistant(connection, auth);
  if (!assistantID) {
    throw new Error("assistant creation returned no identifier");
  }

  const model = new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  model.setModelprovidername("anthropic");
  const modelRequest = new CreateAssistantProviderRequest();
  modelRequest.setAssistantid(assistantID);
  modelRequest.setDescription("CI secondary model provider");
  modelRequest.setModel(model);
  const modelResponse = await createProvider(connection, modelRequest, auth);
  requireSuccess(modelResponse, "model provider creation");

  const agentkit = new CreateAssistantProviderRequest.CreateAssistantProviderAgentkit();
  agentkit.setAgentkiturl("agentkit:50051");
  agentkit.setTransportsecurity("PLAINTEXT");
  const agentkitRequest = new CreateAssistantProviderRequest();
  agentkitRequest.setAssistantid(assistantID);
  agentkitRequest.setDescription("CI AgentKit provider");
  agentkitRequest.setAgentkit(agentkit);
  const agentkitResponse = await createProvider(connection, agentkitRequest, auth);
  requireSuccess(agentkitResponse, "AgentKit provider creation");

  const websocket = new CreateAssistantProviderRequest.CreateAssistantProviderWebsocket();
  websocket.setWebsocketurl("wss://example.invalid/agent");
  const websocketRequest = new CreateAssistantProviderRequest();
  websocketRequest.setAssistantid(assistantID);
  websocketRequest.setDescription("CI WebSocket provider");
  websocketRequest.setWebsocket(websocket);
  const websocketResponse = await createProvider(connection, websocketRequest, auth);
  requireSuccess(websocketResponse, "WebSocket provider creation");

  const agentflow = new CreateAssistantProviderRequest.CreateAssistantProviderAgentflow();
  agentflow.setSchemaversion("1.0");
  agentflow.setDefinition(Struct.fromJavaScript({
    entryNodeId: "start",
    nodes: [{ id: "start" }],
    edges: [],
  }));
  const agentflowRequest = new CreateAssistantProviderRequest();
  agentflowRequest.setAssistantid(assistantID);
  agentflowRequest.setDescription("CI AgentFlow provider");
  agentflowRequest.setAgentflow(agentflow);
  const agentflowResponse = await createProvider(connection, agentflowRequest, auth);
  requireSuccess(agentflowResponse, "AgentFlow provider creation");

  console.log("Node SDK create assistant provider flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

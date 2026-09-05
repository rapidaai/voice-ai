global.window = globalThis;
global.self = globalThis;
global.navigator = { userAgent: "node" };

const {
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantProvider,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
} = require("@rapidaai/react");
const { Struct } = require("google-protobuf/google/protobuf/struct_pb");

const assistantEndpoint = process.env.ASSISTANT_API_REACT_ENDPOINT || "http://assistant-api:9007";

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
  const connection = new ConnectionConfig({ assistant: assistantEndpoint }, true);
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
  const modelResponse = await CreateAssistantProvider(connection, modelRequest, auth);
  requireSuccess(modelResponse, "model provider creation");
  if (modelResponse.getAssistantprovidermodel()?.getModelprovidername() !== "anthropic") {
    throw new Error("model provider creation returned unexpected data");
  }

  const agentkit = new CreateAssistantProviderRequest.CreateAssistantProviderAgentkit();
  agentkit.setAgentkiturl("agentkit:50051");
  agentkit.setTransportsecurity("PLAINTEXT");
  const agentkitRequest = new CreateAssistantProviderRequest();
  agentkitRequest.setAssistantid(assistantID);
  agentkitRequest.setDescription("CI AgentKit provider");
  agentkitRequest.setAgentkit(agentkit);
  const agentkitResponse = await CreateAssistantProvider(connection, agentkitRequest, auth);
  requireSuccess(agentkitResponse, "AgentKit provider creation");
  if (agentkitResponse.getAssistantprovideragentkit()?.getUrl() !== "agentkit:50051") {
    throw new Error("AgentKit provider creation returned unexpected data");
  }

  const websocket = new CreateAssistantProviderRequest.CreateAssistantProviderWebsocket();
  websocket.setWebsocketurl("wss://example.invalid/agent");
  const websocketRequest = new CreateAssistantProviderRequest();
  websocketRequest.setAssistantid(assistantID);
  websocketRequest.setDescription("CI WebSocket provider");
  websocketRequest.setWebsocket(websocket);
  const websocketResponse = await CreateAssistantProvider(connection, websocketRequest, auth);
  requireSuccess(websocketResponse, "WebSocket provider creation");
  if (websocketResponse.getAssistantproviderwebsocket()?.getUrl() !== "wss://example.invalid/agent") {
    throw new Error("WebSocket provider creation returned unexpected data");
  }

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
  const agentflowResponse = await CreateAssistantProvider(connection, agentflowRequest, auth);
  requireSuccess(agentflowResponse, "AgentFlow provider creation");
  if (agentflowResponse.getAssistantprovideragentflow()?.getSchemaversion() !== "1.0") {
    throw new Error("AgentFlow provider creation returned unexpected data");
  }

  console.log("React SDK create assistant provider flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

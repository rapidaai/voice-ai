const {
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  CreateAssistantTool,
  CreateAssistantToolRequest,
  DeleteAssistantTool,
  DeleteAssistantToolRequest,
  GetAllAssistantTool,
  GetAllAssistantToolRequest,
  GetAssistantTool,
  GetAssistantToolRequest,
  Metadata,
  Paginate,
  UpdateAssistantTool,
  UpdateAssistantToolRequest,
  WithAuthContext,
} = require("@rapidaai/nodejs");
const { Struct } = require("google-protobuf/google/protobuf/struct_pb");

const assistantEndpoint =
  process.env.ASSISTANT_API_ENDPOINT || "assistant-api:9007";

const configurationSpecs = [
  {
    type: "storage",
    provider: "aws",
    optionKey: "files",
    initialValue: '["conversation.audio"]',
    updatedValue: '["conversation.audio","conversation.transcript"]',
  },
  {
    type: "analysis",
    provider: "endpoint",
    optionKey: "name",
    initialValue: "CI analysis",
    updatedValue: "CI analysis updated",
  },
  {
    type: "authentication",
    provider: "http",
    optionKey: "http_url",
    initialValue: "https://example.invalid/authenticate",
    updatedValue: "https://example.invalid/authenticate-updated",
  },
  {
    type: "webhook",
    provider: "http",
    optionKey: "events",
    initialValue: '["conversation.begin"]',
    updatedValue: '["conversation.begin","conversation.completed"]',
  },
  {
    type: "telemetry",
    provider: "otlp_http",
    optionKey: "endpoint",
    initialValue: "https://example.invalid/v1/traces",
    updatedValue: "https://example.invalid/v1/traces-updated",
  },
];

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
  return {
    personalToken: {
      name: "personal access token",
      auth: ConnectionConfig.WithPersonalToken({
        Authorization: requireValue("FLOW_TOKEN"),
        AuthId: requireValue("FLOW_FIXTURE_ID"),
        ProjectId: requireValue("FLOW_PROJECT_ID"),
      }),
    },
    projectKey: {
      name: "project API key",
      auth: ConnectionConfig.WithSDK({
        ApiKey: requireValue("FLOW_API_KEY"),
        UserId: "",
      }),
    },
  };
}

function metadata(key, value) {
  const option = new Metadata();
  option.setKey(key);
  option.setValue(value);
  return option;
}

function pagination() {
  const paginate = new Paginate();
  paginate.setPage(1);
  paginate.setPagesize(100);
  return paginate;
}

function invokeAssistantClient(connection, methodName, request, auth) {
  return new Promise((resolve, reject) => {
    connection.assistantClient[methodName](
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

function optionValue(options, key) {
  return options.find((option) => option.getKey() === key)?.getValue();
}

async function createAssistant(connection, mode) {
  const model =
    new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  model.setModelprovidername("openai");
  const provider = new CreateAssistantProviderRequest();
  provider.setDescription("CI assistant components model provider");
  provider.setModel(model);

  const request = new CreateAssistantRequest();
  request.setName(`${requireValue("FLOW_ASSISTANT")} ${mode.name}`);
  request.setDescription("Created by the assistant components flow");
  request.setVisibility("private");
  request.setLanguage("english");
  request.setAssistantprovider(provider);

  const response = await CreateAssistant(connection, request, mode.auth);
  requireSuccess(response, `assistant creation with ${mode.name}`);
  const assistant = response.getData();
  if (
    !assistant?.getId() ||
    assistant.getProjectid() !== requireValue("FLOW_PROJECT_ID") ||
    assistant.getOrganizationid() !== requireValue("FLOW_FIXTURE_ID")
  ) {
    throw new Error(`assistant creation with ${mode.name} returned wrong ownership`);
  }
  return assistant.getId();
}

function assertTool(tool, expected) {
  if (
    !tool ||
    tool.getId() !== expected.id ||
    tool.getAssistantid() !== expected.assistantID ||
    tool.getName() !== expected.name ||
    tool.getDescription() !== expected.description ||
    tool.getExecutionmethod() !== "endpoint" ||
    tool.getStatus() !== "ACTIVE"
  ) {
    throw new Error(`${expected.action} returned unexpected tool data`);
  }
  if (
    optionValue(tool.getExecutionoptionsList(), "tool.endpoint_id") !==
      expected.endpointID ||
    optionValue(tool.getExecutionoptionsList(), "tool.parameters") !==
      '{"tool.query":"query"}'
  ) {
    throw new Error(`${expected.action} returned unexpected tool options`);
  }
  const fields = tool.getFields()?.toJavaScript();
  if (
    fields?.type !== "object" ||
    fields?.properties?.query?.type !== "string"
  ) {
    throw new Error(`${expected.action} returned unexpected tool fields`);
  }
}

async function runToolLifecycle(connection, assistantID, mode) {
  const initialName = `ci_tool_${mode.name.replaceAll(" ", "_")}`;
  const initialDescription = `CI tool created with ${mode.name}`;
  const updatedName = `${initialName}_updated`;
  const updatedDescription = `CI tool updated with ${mode.name}`;
  const initialEndpointID = `initial-${mode.name.replaceAll(" ", "-")}`;
  const updatedEndpointID = `updated-${mode.name.replaceAll(" ", "-")}`;
  const fields = Struct.fromJavaScript({
    type: "object",
    properties: { query: { type: "string" } },
  });

  const createRequest = new CreateAssistantToolRequest();
  createRequest.setAssistantid(assistantID);
  createRequest.setName(initialName);
  createRequest.setDescription(initialDescription);
  createRequest.setFields(fields);
  createRequest.setExecutionmethod("endpoint");
  createRequest.setExecutionoptionsList([
    metadata("tool.endpoint_id", initialEndpointID),
    metadata("tool.parameters", '{"tool.query":"query"}'),
  ]);
  const createResponse = await CreateAssistantTool(
    connection,
    createRequest,
    mode.auth,
  );
  requireSuccess(createResponse, `tool creation with ${mode.name}`);
  const toolID = createResponse.getData()?.getId();
  if (!toolID) {
    throw new Error(`tool creation with ${mode.name} returned no identifier`);
  }
  assertTool(createResponse.getData(), {
    id: toolID,
    assistantID,
    name: initialName,
    description: initialDescription,
    endpointID: initialEndpointID,
    action: "tool creation",
  });

  const getRequest = new GetAssistantToolRequest();
  getRequest.setId(toolID);
  getRequest.setAssistantid(assistantID);
  const getResponse = await GetAssistantTool(connection, getRequest, mode.auth);
  requireSuccess(getResponse, `tool lookup with ${mode.name}`);
  assertTool(getResponse.getData(), {
    id: toolID,
    assistantID,
    name: initialName,
    description: initialDescription,
    endpointID: initialEndpointID,
    action: "tool lookup",
  });

  const listRequest = new GetAllAssistantToolRequest();
  listRequest.setAssistantid(assistantID);
  listRequest.setPaginate(pagination());
  const listResponse = await GetAllAssistantTool(
    connection,
    listRequest,
    mode.auth,
  );
  requireSuccess(listResponse, `tool list with ${mode.name}`);
  if (!listResponse.getDataList().some((tool) => tool.getId() === toolID)) {
    throw new Error(`tool list with ${mode.name} omitted the created tool`);
  }

  const updateRequest = new UpdateAssistantToolRequest();
  updateRequest.setId(toolID);
  updateRequest.setAssistantid(assistantID);
  updateRequest.setName(updatedName);
  updateRequest.setDescription(updatedDescription);
  updateRequest.setFields(fields);
  updateRequest.setExecutionmethod("endpoint");
  updateRequest.setExecutionoptionsList([
    metadata("tool.endpoint_id", updatedEndpointID),
    metadata("tool.parameters", '{"tool.query":"query"}'),
  ]);
  const updateResponse = await UpdateAssistantTool(
    connection,
    updateRequest,
    mode.auth,
  );
  requireSuccess(updateResponse, `tool update with ${mode.name}`);

  const updatedGetResponse = await GetAssistantTool(
    connection,
    getRequest,
    mode.auth,
  );
  requireSuccess(updatedGetResponse, `updated tool lookup with ${mode.name}`);
  assertTool(updatedGetResponse.getData(), {
    id: toolID,
    assistantID,
    name: updatedName,
    description: updatedDescription,
    endpointID: updatedEndpointID,
    action: "updated tool lookup",
  });

  const deleteRequest = new DeleteAssistantToolRequest();
  deleteRequest.setId(toolID);
  deleteRequest.setAssistantid(assistantID);
  const deleteResponse = await DeleteAssistantTool(
    connection,
    deleteRequest,
    mode.auth,
  );
  requireSuccess(deleteResponse, `tool deletion with ${mode.name}`);

  const deletedListResponse = await GetAllAssistantTool(
    connection,
    listRequest,
    mode.auth,
  );
  requireSuccess(deletedListResponse, `tool list after deletion with ${mode.name}`);
  if (deletedListResponse.getDataList().some((tool) => tool.getId() === toolID)) {
    throw new Error(`tool list with ${mode.name} retained the archived tool`);
  }

  console.log(`Node SDK assistant tool lifecycle passed with ${mode.name}`);
}

function configurationRequest(
  connection,
  methodName,
  assistantID,
  spec,
  configurationID,
  optionValueText,
  enabled,
) {
  const RequestType = connection.assistantClient[methodName].requestType;
  const request = new RequestType();
  if (configurationID) {
    request.setId(configurationID);
  }
  request.setAssistantid(assistantID);
  request.setConfigurationtype(spec.type);
  request.setProvider(spec.provider);
  request.setEnabled(enabled);
  request.setOptionsList([
    metadata(spec.optionKey, optionValueText),
    metadata("test.marker", `${enabled ? "initial" : "updated"}-${spec.type}`),
  ]);
  return request;
}

function assertConfiguration(configuration, expected) {
  if (
    !configuration ||
    configuration.getId() !== expected.id ||
    configuration.getAssistantid() !== expected.assistantID ||
    configuration.getProjectid() !== requireValue("FLOW_PROJECT_ID") ||
    configuration.getOrganizationid() !== requireValue("FLOW_FIXTURE_ID") ||
    configuration.getConfigurationtype() !== expected.spec.type ||
    configuration.getProvider() !== expected.spec.provider ||
    configuration.getEnabled() !== expected.enabled ||
    configuration.getStatus() !== "ACTIVE"
  ) {
    throw new Error(`${expected.action} returned unexpected configuration data`);
  }
  if (
    optionValue(configuration.getOptionsList(), expected.spec.optionKey) !==
      expected.optionValue ||
    optionValue(configuration.getOptionsList(), "test.marker") !==
      `${expected.enabled ? "initial" : "updated"}-${expected.spec.type}`
  ) {
    throw new Error(`${expected.action} returned unexpected configuration options`);
  }
}

async function runConfigurationLifecycle(connection, assistantID, auth, spec) {
  const createRequest = configurationRequest(
    connection,
    "createAssistantConfiguration",
    assistantID,
    spec,
    undefined,
    spec.initialValue,
    true,
  );
  const createResponse = await invokeAssistantClient(
    connection,
    "createAssistantConfiguration",
    createRequest,
    auth,
  );
  requireSuccess(createResponse, `${spec.type} configuration creation`);
  const configurationID = createResponse.getData()?.getId();
  if (!configurationID) {
    throw new Error(`${spec.type} configuration creation returned no identifier`);
  }
  assertConfiguration(createResponse.getData(), {
    id: configurationID,
    assistantID,
    spec,
    enabled: true,
    optionValue: spec.initialValue,
    action: `${spec.type} configuration creation`,
  });

  const GetRequest =
    connection.assistantClient.getAssistantConfiguration.requestType;
  const getRequest = new GetRequest();
  getRequest.setId(configurationID);
  getRequest.setAssistantid(assistantID);
  const getResponse = await invokeAssistantClient(
    connection,
    "getAssistantConfiguration",
    getRequest,
    auth,
  );
  requireSuccess(getResponse, `${spec.type} configuration lookup`);
  assertConfiguration(getResponse.getData(), {
    id: configurationID,
    assistantID,
    spec,
    enabled: true,
    optionValue: spec.initialValue,
    action: `${spec.type} configuration lookup`,
  });

  const ListRequest =
    connection.assistantClient.getAllAssistantConfiguration.requestType;
  const listRequest = new ListRequest();
  listRequest.setAssistantid(assistantID);
  listRequest.setConfigurationtype(spec.type);
  listRequest.setProvider(spec.provider);
  listRequest.setPaginate(pagination());
  const listResponse = await invokeAssistantClient(
    connection,
    "getAllAssistantConfiguration",
    listRequest,
    auth,
  );
  requireSuccess(listResponse, `${spec.type} configuration list`);
  if (
    !listResponse
      .getDataList()
      .some((configuration) => configuration.getId() === configurationID)
  ) {
    throw new Error(`${spec.type} configuration list omitted the created item`);
  }

  const updateRequest = configurationRequest(
    connection,
    "updateAssistantConfiguration",
    assistantID,
    spec,
    configurationID,
    spec.updatedValue,
    false,
  );
  const updateResponse = await invokeAssistantClient(
    connection,
    "updateAssistantConfiguration",
    updateRequest,
    auth,
  );
  requireSuccess(updateResponse, `${spec.type} configuration update`);
  assertConfiguration(updateResponse.getData(), {
    id: configurationID,
    assistantID,
    spec,
    enabled: false,
    optionValue: spec.updatedValue,
    action: `${spec.type} configuration update`,
  });

  const updatedGetResponse = await invokeAssistantClient(
    connection,
    "getAssistantConfiguration",
    getRequest,
    auth,
  );
  requireSuccess(updatedGetResponse, `${spec.type} updated configuration lookup`);
  assertConfiguration(updatedGetResponse.getData(), {
    id: configurationID,
    assistantID,
    spec,
    enabled: false,
    optionValue: spec.updatedValue,
    action: `${spec.type} updated configuration lookup`,
  });

  const DeleteRequest =
    connection.assistantClient.deleteAssistantConfiguration.requestType;
  const deleteRequest = new DeleteRequest();
  deleteRequest.setId(configurationID);
  deleteRequest.setAssistantid(assistantID);
  const deleteResponse = await invokeAssistantClient(
    connection,
    "deleteAssistantConfiguration",
    deleteRequest,
    auth,
  );
  requireSuccess(deleteResponse, `${spec.type} configuration deletion`);

  const deletedListResponse = await invokeAssistantClient(
    connection,
    "getAllAssistantConfiguration",
    listRequest,
    auth,
  );
  requireSuccess(deletedListResponse, `${spec.type} configuration list after deletion`);
  if (
    deletedListResponse
      .getDataList()
      .some((configuration) => configuration.getId() === configurationID)
  ) {
    throw new Error(`${spec.type} configuration list retained the archived item`);
  }

  console.log(`Node SDK assistant ${spec.type} lifecycle passed`);
}

async function main() {
  const connection = new ConnectionConfig({ assistant: assistantEndpoint }, true);
  const modes = authenticationModes();

  const personalAssistantID = await createAssistant(
    connection,
    modes.personalToken,
  );
  await runToolLifecycle(connection, personalAssistantID, modes.personalToken);
  for (const spec of configurationSpecs) {
    await runConfigurationLifecycle(
      connection,
      personalAssistantID,
      modes.personalToken.auth,
      spec,
    );
  }

  const projectKeyAssistantID = await createAssistant(
    connection,
    modes.projectKey,
  );
  await runToolLifecycle(connection, projectKeyAssistantID, modes.projectKey);

  console.log("Node SDK assistant components flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

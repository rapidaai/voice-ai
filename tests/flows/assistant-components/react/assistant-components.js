require("../../react-environment");

const {
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantConfiguration,
  CreateAssistantConfigurationRequest,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  CreateAssistantTool,
  DeleteAssistantConfiguration,
  DeleteAssistantConfigurationRequest,
  DeleteAssistantTool,
  GetAllAssistantConfiguration,
  GetAllAssistantConfigurationRequest,
  GetAllAssistantTool,
  GetAssistantConfiguration,
  GetAssistantConfigurationRequest,
  GetAssistantTool,
  Metadata,
  Paginate,
  UpdateAssistantConfiguration,
  UpdateAssistantConfigurationRequest,
  UpdateAssistantTool,
} = require("@rapidaai/react");

const webEndpoint = process.env.WEB_API_REACT_ENDPOINT || "http://web-api:9001";

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

function personalToken() {
  return ConnectionConfig.WithPersonalToken({
    Authorization: requireValue("FLOW_TOKEN"),
    AuthId: requireValue("FLOW_FIXTURE_ID"),
    ProjectId: requireValue("FLOW_PROJECT_ID"),
  });
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

function callbackCall(invoke) {
  return new Promise((resolve, reject) => {
    invoke((error, response) => {
      if (error) {
        reject(error);
        return;
      }
      resolve(response);
    });
  });
}

function optionValue(options, key) {
  return options.find((option) => option.getKey() === key)?.getValue();
}

async function createAssistant(connection, auth) {
  const model =
    new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  model.setModelprovidername("openai");
  const provider = new CreateAssistantProviderRequest();
  provider.setDescription("CI assistant components model provider");
  provider.setModel(model);

  const request = new CreateAssistantRequest();
  request.setName(requireValue("FLOW_ASSISTANT"));
  request.setDescription("Created by the assistant components flow");
  request.setVisibility("private");
  request.setLanguage("english");
  request.setAssistantprovider(provider);

  const response = await CreateAssistant(connection, request, auth);
  requireSuccess(response, "assistant creation");
  const assistant = response.getData();
  if (
    !assistant?.getId() ||
    assistant.getProjectid() !== requireValue("FLOW_PROJECT_ID") ||
    assistant.getOrganizationid() !== requireValue("FLOW_FIXTURE_ID")
  ) {
    throw new Error("assistant creation returned wrong ownership");
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
    expected.endpointID
  ) {
    throw new Error(`${expected.action} returned unexpected tool options`);
  }
  const fields = tool.getFields()?.toJavaScript();
  if (fields?.type !== "object") {
    throw new Error(`${expected.action} returned unexpected tool fields`);
  }
}

async function runToolLifecycle(connection, assistantID, auth) {
  const initialName = "ci_react_tool";
  const initialDescription = "CI React assistant tool";
  const updatedName = "ci_react_tool_updated";
  const updatedDescription = "CI React assistant tool updated";
  const initialEndpointID = "initial-react-endpoint";
  const updatedEndpointID = "updated-react-endpoint";
  const fields = {
    type: "object",
    properties: { query: { type: "string" } },
  };
  const initialOptions = [
    metadata("tool.endpoint_id", initialEndpointID),
    metadata("tool.parameters", '{"tool.query":"query"}'),
  ];

  const createResponse = await callbackCall((callback) =>
    CreateAssistantTool(
      connection,
      assistantID,
      initialName,
      initialDescription,
      fields,
      "endpoint",
      initialOptions,
      callback,
      auth,
    ),
  );
  requireSuccess(createResponse, "tool creation");
  const toolID = createResponse.getData()?.getId();
  if (!toolID) {
    throw new Error("tool creation returned no identifier");
  }
  assertTool(createResponse.getData(), {
    id: toolID,
    assistantID,
    name: initialName,
    description: initialDescription,
    endpointID: initialEndpointID,
    action: "tool creation",
  });

  const getResponse = await callbackCall((callback) =>
    GetAssistantTool(connection, assistantID, toolID, callback, auth),
  );
  requireSuccess(getResponse, "tool lookup");
  assertTool(getResponse.getData(), {
    id: toolID,
    assistantID,
    name: initialName,
    description: initialDescription,
    endpointID: initialEndpointID,
    action: "tool lookup",
  });

  const listResponse = await callbackCall((callback) =>
    GetAllAssistantTool(connection, assistantID, 1, 100, [], callback, auth),
  );
  requireSuccess(listResponse, "tool list");
  if (!listResponse.getDataList().some((tool) => tool.getId() === toolID)) {
    throw new Error("tool list omitted the created tool");
  }

  const updateResponse = await callbackCall((callback) =>
    UpdateAssistantTool(
      connection,
      assistantID,
      toolID,
      updatedName,
      updatedDescription,
      fields,
      "endpoint",
      [
        metadata("tool.endpoint_id", updatedEndpointID),
        metadata("tool.parameters", '{"tool.query":"query"}'),
      ],
      callback,
      auth,
    ),
  );
  requireSuccess(updateResponse, "tool update");

  const updatedGetResponse = await callbackCall((callback) =>
    GetAssistantTool(connection, assistantID, toolID, callback, auth),
  );
  requireSuccess(updatedGetResponse, "updated tool lookup");
  assertTool(updatedGetResponse.getData(), {
    id: toolID,
    assistantID,
    name: updatedName,
    description: updatedDescription,
    endpointID: updatedEndpointID,
    action: "updated tool lookup",
  });

  const deleteResponse = await callbackCall((callback) =>
    DeleteAssistantTool(connection, assistantID, toolID, callback, auth),
  );
  requireSuccess(deleteResponse, "tool deletion");

  const deletedListResponse = await callbackCall((callback) =>
    GetAllAssistantTool(connection, assistantID, 1, 100, [], callback, auth),
  );
  requireSuccess(deletedListResponse, "tool list after deletion");
  if (deletedListResponse.getDataList().some((tool) => tool.getId() === toolID)) {
    throw new Error("tool list retained the archived tool");
  }

  console.log("React SDK assistant tool lifecycle passed");
}

function createConfigurationRequest(assistantID, spec) {
  const request = new CreateAssistantConfigurationRequest();
  request.setAssistantid(assistantID);
  request.setConfigurationtype(spec.type);
  request.setProvider(spec.provider);
  request.setEnabled(true);
  request.setOptionsList([
    metadata(spec.optionKey, spec.initialValue),
    metadata("test.marker", `initial-${spec.type}`),
  ]);
  return request;
}

function updateConfigurationRequest(assistantID, configurationID, spec) {
  const request = new UpdateAssistantConfigurationRequest();
  request.setId(configurationID);
  request.setAssistantid(assistantID);
  request.setConfigurationtype(spec.type);
  request.setProvider(spec.provider);
  request.setEnabled(false);
  request.setOptionsList([
    metadata(spec.optionKey, spec.updatedValue),
    metadata("test.marker", `updated-${spec.type}`),
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

function configurationListRequest(assistantID, spec) {
  const request = new GetAllAssistantConfigurationRequest();
  request.setAssistantid(assistantID);
  request.setConfigurationtype(spec.type);
  request.setProvider(spec.provider);
  request.setPaginate(pagination());
  return request;
}

async function runConfigurationLifecycle(connection, assistantID, auth, spec) {
  const createResponse = await CreateAssistantConfiguration(
    connection,
    createConfigurationRequest(assistantID, spec),
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

  const getRequest = new GetAssistantConfigurationRequest();
  getRequest.setId(configurationID);
  getRequest.setAssistantid(assistantID);
  const getResponse = await GetAssistantConfiguration(
    connection,
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

  const listRequest = configurationListRequest(assistantID, spec);
  const listResponse = await GetAllAssistantConfiguration(
    connection,
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

  const updateResponse = await UpdateAssistantConfiguration(
    connection,
    updateConfigurationRequest(assistantID, configurationID, spec),
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

  const updatedGetResponse = await GetAssistantConfiguration(
    connection,
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

  const deleteRequest = new DeleteAssistantConfigurationRequest();
  deleteRequest.setId(configurationID);
  deleteRequest.setAssistantid(assistantID);
  const deleteResponse = await DeleteAssistantConfiguration(
    connection,
    deleteRequest,
    auth,
  );
  requireSuccess(deleteResponse, `${spec.type} configuration deletion`);

  const deletedListResponse = await GetAllAssistantConfiguration(
    connection,
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

  console.log(`React SDK assistant ${spec.type} lifecycle passed`);
}

async function main() {
  const connection = new ConnectionConfig(
    { web: webEndpoint },
    process.env.FLOW_SDK_DEBUG === "true",
  );
  const auth = personalToken();
  const assistantID = await createAssistant(connection, auth);

  await runToolLifecycle(connection, assistantID, auth);
  for (const spec of configurationSpecs) {
    await runConfigurationLifecycle(connection, assistantID, auth, spec);
  }

  console.log("React SDK assistant components flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

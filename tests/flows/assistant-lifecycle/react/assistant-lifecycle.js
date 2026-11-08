require("../../react-environment");

const {
  AssistantDefinition,
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantProvider,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  DeleteAssistantRequest,
  GetAllAssistant,
  GetAllAssistantProvider,
  GetAllAssistantRequest,
  GetAssistant,
  GetAssistantRequest,
  Paginate,
  UpdateAssistantDetailRequest,
  UpdateAssistantVersion,
  UpdateAssistantVersionRequest,
  WithAuthContext,
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
  ];
}

function call(invoke) {
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

function getRequest(assistantID) {
  const definition = new AssistantDefinition();
  definition.setAssistantid(assistantID);
  const request = new GetAssistantRequest();
  request.setAssistantdefinition(definition);
  return request;
}

function listRequest() {
  const paginate = new Paginate();
  paginate.setPage(1);
  paginate.setPagesize(100);
  const request = new GetAllAssistantRequest();
  request.setPaginate(paginate);
  return request;
}

function assertAssistant(assistant, expected) {
  if (
    !assistant ||
    assistant.getId() !== expected.id ||
    assistant.getName() !== expected.name ||
    assistant.getDescription() !== expected.description ||
    assistant.getProjectid() !== requireValue("FLOW_PROJECT_ID") ||
    assistant.getOrganizationid() !== requireValue("FLOW_FIXTURE_ID")
  ) {
    throw new Error(`${expected.action} returned unexpected assistant data`);
  }
  if (
    expected.providerID &&
    (assistant.getAssistantprovider() !== "MODEL" ||
      assistant.getAssistantproviderid() !== expected.providerID)
  ) {
    throw new Error(`${expected.action} returned the wrong assistant version`);
  }
}

function assertListed(response, assistantID, shouldExist, action) {
  requireSuccess(response, action);
  const exists = response
    .getDataList()
    .some((assistant) => assistant.getId() === assistantID);
  if (exists !== shouldExist) {
    throw new Error(`${action} returned unexpected assistant membership`);
  }
}

async function runFlow(mode) {
  const auth = mode.auth;
  const connection = new ConnectionConfig(
    { web: webEndpoint },
    process.env.FLOW_SDK_DEBUG === "true",
  );
  const initialName = `${requireValue("FLOW_ASSISTANT")} ${mode.name}`;
  const initialDescription = "Created by the CI assistant lifecycle flow";
  const updatedName = `${initialName} Updated`;
  const updatedDescription = "Updated by the CI assistant lifecycle flow";

  const initialModel =
    new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  initialModel.setModelprovidername("openai");
  const initialProvider = new CreateAssistantProviderRequest();
  initialProvider.setDescription("CI initial model provider");
  initialProvider.setModel(initialModel);
  const createRequest = new CreateAssistantRequest();
  createRequest.setName(initialName);
  createRequest.setDescription(initialDescription);
  createRequest.setVisibility("private");
  createRequest.setLanguage("english");
  createRequest.setAssistantprovider(initialProvider);

  const createResponse = await CreateAssistant(connection, createRequest, auth);
  requireSuccess(createResponse, "assistant creation");
  const created = createResponse.getData();
  const assistantID = created?.getId();
  if (!assistantID) {
    throw new Error("assistant creation returned no identifier");
  }
  assertAssistant(created, {
    id: assistantID,
    name: initialName,
    description: initialDescription,
    action: "assistant creation",
  });

  const getResponse = await GetAssistant(
    connection,
    getRequest(assistantID),
    auth,
  );
  requireSuccess(getResponse, "assistant lookup");
  assertAssistant(getResponse.getData(), {
    id: assistantID,
    name: initialName,
    description: initialDescription,
    action: "assistant lookup",
  });

  assertListed(
    await GetAllAssistant(connection, listRequest(), auth),
    assistantID,
    true,
    "assistant list before update",
  );

  const detailRequest = new UpdateAssistantDetailRequest();
  detailRequest.setAssistantid(assistantID);
  detailRequest.setName(updatedName);
  detailRequest.setDescription(updatedDescription);
  const detailResponse = await call((callback) =>
    connection.assistantClient.updateAssistantDetail(
      detailRequest,
      WithAuthContext(auth),
      callback,
    ),
  );
  requireSuccess(detailResponse, "assistant detail update");
  assertAssistant(detailResponse.getData(), {
    id: assistantID,
    name: updatedName,
    description: updatedDescription,
    action: "assistant detail update",
  });

  const secondaryModel =
    new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  secondaryModel.setModelprovidername("anthropic");
  const providerRequest = new CreateAssistantProviderRequest();
  providerRequest.setAssistantid(assistantID);
  providerRequest.setDescription("CI secondary model provider");
  providerRequest.setModel(secondaryModel);
  const providerResponse = await CreateAssistantProvider(
    connection,
    providerRequest,
    auth,
  );
  requireSuccess(providerResponse, "assistant provider creation");
  const providersResponse = await call((callback) =>
    GetAllAssistantProvider(
      connection,
      assistantID,
      1,
      100,
      [],
      callback,
      auth,
    ),
  );
  requireSuccess(providersResponse, "assistant provider list");
  const secondaryProviderID = providersResponse
    .getDataList()
    .map((provider) => provider.getAssistantprovidermodel())
    .find((provider) => provider?.getModelprovidername() === "anthropic")
    ?.getId();
  if (!secondaryProviderID) {
    throw new Error("assistant provider creation returned no identifier");
  }

  const versionRequest = new UpdateAssistantVersionRequest();
  versionRequest.setAssistantid(assistantID);
  versionRequest.setAssistantprovider("MODEL");
  versionRequest.setAssistantproviderid(secondaryProviderID);
  const versionResponse = await UpdateAssistantVersion(
    connection,
    versionRequest,
    auth,
  );
  requireSuccess(versionResponse, "assistant version update");

  const updatedGetResponse = await GetAssistant(
    connection,
    getRequest(assistantID),
    auth,
  );
  requireSuccess(updatedGetResponse, "updated assistant lookup");
  assertAssistant(updatedGetResponse.getData(), {
    id: assistantID,
    name: updatedName,
    description: updatedDescription,
    providerID: secondaryProviderID,
    action: "updated assistant lookup",
  });

  const deleteRequest = new DeleteAssistantRequest();
  deleteRequest.setId(assistantID);
  const deleteResponse = await call((callback) =>
    connection.assistantClient.deleteAssistant(
      deleteRequest,
      WithAuthContext(auth),
      callback,
    ),
  );
  requireSuccess(deleteResponse, "assistant deletion");

  assertListed(
    await GetAllAssistant(connection, listRequest(), auth),
    assistantID,
    false,
    "assistant list after deletion",
  );

  console.log(`React SDK assistant lifecycle passed with ${mode.name}`);
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

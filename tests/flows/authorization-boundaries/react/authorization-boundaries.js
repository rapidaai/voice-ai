require("../../react-environment");

const {
  AssistantDefinition,
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  GetAssistant,
  GetAssistantRequest,
} = require("@rapidaai/react");

const webEndpoint = process.env.WEB_API_REACT_ENDPOINT || "http://web-api:9001";

function requireValue(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

function createRequest() {
  const model =
    new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  model.setModelprovidername("openai");
  const provider = new CreateAssistantProviderRequest();
  provider.setModel(model);
  const request = new CreateAssistantRequest();
  request.setName("Rejected React authorization request");
  request.setVisibility("private");
  request.setLanguage("english");
  request.setAssistantprovider(provider);
  return request;
}

function getRequest(assistantID) {
  const definition = new AssistantDefinition();
  definition.setAssistantid(assistantID);
  const request = new GetAssistantRequest();
  request.setAssistantdefinition(definition);
  return request;
}

async function expectDenied(action, invoke) {
  try {
    await invoke();
    throw new Error(`${action} unexpectedly succeeded`);
  } catch (error) {
    if (error?.message === `${action} unexpectedly succeeded`) {
      throw error;
    }
    if (error?.code === 7 || error?.code === 16) {
      console.log(`${action} was rejected with transport code ${error.code}`);
      return;
    }
    throw new Error(`${action} returned an unexpected error`, { cause: error });
  }
}

async function expectMissing(action, invoke) {
  try {
    await invoke();
    throw new Error(`${action} unexpectedly succeeded`);
  } catch (error) {
    if (error?.message === `${action} unexpectedly succeeded`) {
      throw error;
    }
    if (error?.code === 2 && error?.message === "record not found") {
      console.log(`${action} was rejected as not found`);
      return;
    }
    throw new Error(`${action} returned an unexpected error`, { cause: error });
  }
}

async function main() {
  const connection = new ConnectionConfig(
    { web: webEndpoint },
    process.env.FLOW_SDK_DEBUG === "true",
  );
  const ownerID = requireValue("FLOW_FIXTURE_ID");
  const ownerProjectID = requireValue("FLOW_PROJECT_ID");
  const ownerToken = requireValue("FLOW_TOKEN");

  await expectDenied("invalid personal access token", () =>
    CreateAssistant(
      connection,
      createRequest(),
      ConnectionConfig.WithPersonalToken({
        Authorization: "invalid-personal-access-token",
        AuthId: ownerID,
        ProjectId: ownerProjectID,
      }),
    ),
  );

  await expectDenied("invalid project API key", () =>
    CreateAssistant(
      connection,
      createRequest(),
      ConnectionConfig.WithSDK({
        ApiKey: "invalid",
        UserId: "",
      }),
    ),
  );

  await expectDenied("cross-project access", () =>
    CreateAssistant(
      connection,
      createRequest(),
      ConnectionConfig.WithPersonalToken({
        Authorization: ownerToken,
        AuthId: ownerID,
        ProjectId: requireValue("FLOW_OTHER_PROJECT_ID"),
      }),
    ),
  );

  await expectDenied("user without project permission", () =>
    CreateAssistant(
      connection,
      createRequest(),
      ConnectionConfig.WithPersonalToken({
        Authorization: requireValue("FLOW_RESTRICTED_TOKEN"),
        AuthId: requireValue("FLOW_RESTRICTED_USER_ID"),
        ProjectId: ownerProjectID,
      }),
    ),
  );

  await expectMissing("nonexistent assistant identifier", () =>
    GetAssistant(
      connection,
      getRequest("999999999999"),
      ConnectionConfig.WithPersonalToken({
        Authorization: ownerToken,
        AuthId: ownerID,
        ProjectId: ownerProjectID,
      }),
    ),
  );

  console.log("React SDK authorization boundary flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

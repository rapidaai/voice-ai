const {
  ConnectionConfig,
  GetAllAssistant,
  GetAllAssistantRequest,
  Paginate,
} = require("@rapidaai/nodejs");

const assistantEndpoint = "assistant-api:9007";

function createRequest() {
  const paginate = new Paginate();
  paginate.setPage(1);
  paginate.setPagesize(20);

  const request = new GetAllAssistantRequest();
  request.setPaginate(paginate);
  return request;
}

async function verifyAuthentication(name, authentication) {
  const connection = ConnectionConfig.DefaultConnectionConfig(authentication)
    .withCustomEndpoint({ assistant: assistantEndpoint }, true);
  const response = await GetAllAssistant(connection, createRequest());

  if (!response.getSuccess() || response.getCode() !== 200) {
    throw new Error(`${name} SDK request failed with code ${response.getCode()}`);
  }

  console.log(`${name} SDK authentication passed`);
}

async function main() {
  await verifyAuthentication(
    "Project API key",
    ConnectionConfig.WithSDK({
      ApiKey: process.env.CI_STACK_PROJECT_API_KEY,
      UserId: "",
    }),
  );
  await verifyAuthentication(
    "Personal access token",
    ConnectionConfig.WithPersonalToken({
      Authorization: process.env.CI_STACK_AUTH_TOKEN,
      AuthId: "1",
      ProjectId: "1",
    }),
  );
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

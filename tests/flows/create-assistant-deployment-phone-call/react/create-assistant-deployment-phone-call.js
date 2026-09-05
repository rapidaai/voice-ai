require("../../react-environment");

const {
  AssistantDefinition,
  AssistantPhoneDeployment,
  ConnectionConfig,
  CreateAssistant,
  CreateAssistantDeploymentRequest,
  CreateAssistantPhoneDeployment,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
  CreatePhoneCall,
  CreatePhoneCallRequest,
  Metadata,
} = require("@rapidaai/react");
const { startMockARIServer } = require("../mock-ari-server");

const webEndpoint = process.env.WEB_API_REACT_ENDPOINT || "http://web-api:9001";
const assistantEndpoint =
  process.env.ASSISTANT_API_REACT_ENDPOINT || "http://assistant-api:9007";

function requireValue(name) {
  const value = process.env[name];
  if (!value) throw new Error(`${name} is required`);
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

async function runFlow(mode) {
  const auth = mode.auth;
  const connection = new ConnectionConfig(
    { web: webEndpoint, assistant: assistantEndpoint },
    true,
  );

  const providerModel =
    new CreateAssistantProviderRequest.CreateAssistantProviderModel();
  providerModel.setModelprovidername("openai");
  const provider = new CreateAssistantProviderRequest();
  provider.setDescription("CI phone flow model provider");
  provider.setModel(providerModel);

  const assistantRequest = new CreateAssistantRequest();
  assistantRequest.setName(`${requireValue("FLOW_ASSISTANT")} ${mode.name}`);
  assistantRequest.setVisibility("private");
  assistantRequest.setLanguage("english");
  assistantRequest.setAssistantprovider(provider);

  const assistantResponse = await CreateAssistant(
    connection,
    assistantRequest,
    auth,
  );
  requireSuccess(assistantResponse, "assistant creation");
  const assistantID = assistantResponse.getData()?.getId();
  if (!assistantID)
    throw new Error("assistant creation returned no identifier");

  const credentialOption = new Metadata();
  credentialOption.setKey("rapida.credential_id");
  credentialOption.setValue(requireValue("FLOW_VAULT_ID"));

  const phoneDeployment = new AssistantPhoneDeployment();
  phoneDeployment.setAssistantid(assistantID);
  phoneDeployment.setPhoneprovidername("asterisk");
  phoneDeployment.setGreeting("Hello from the CI phone flow");
  phoneDeployment.setIdealtimeout("30");
  phoneDeployment.setIdealtimeoutbackoff("2");
  phoneDeployment.setMaxsessionduration("180");
  phoneDeployment.addPhoneoptions(credentialOption);

  const deploymentRequest = new CreateAssistantDeploymentRequest();
  deploymentRequest.setPhone(phoneDeployment);
  const deploymentResponse = await CreateAssistantPhoneDeployment(
    connection,
    deploymentRequest,
    auth,
  );
  requireSuccess(deploymentResponse, "phone deployment creation");
  if (deploymentResponse.getData()?.getAssistantid() !== assistantID) {
    throw new Error("phone deployment returned the wrong assistant");
  }

  const assistant = new AssistantDefinition();
  assistant.setAssistantid(assistantID);
  assistant.setVersion("latest");
  const callRequest = new CreatePhoneCallRequest();
  callRequest.setAssistant(assistant);
  callRequest.setFromnumber(requireValue("FLOW_FROM_NUMBER"));
  callRequest.setTonumber(requireValue("FLOW_TO_NUMBER"));

  const callResponse = await CreatePhoneCall(connection, callRequest, auth);
  requireSuccess(callResponse, "phone call creation");
  if (!callResponse.getData()?.getId()) {
    throw new Error("phone call creation returned no conversation identifier");
  }

  console.log(
    `React SDK assistant deployment phone call flow passed with ${mode.name}`,
  );
}

async function main() {
  const mockServer = await startMockARIServer(
    Number(requireValue("FLOW_ARI_PORT")),
  );
  try {
    for (const mode of authenticationModes()) {
      await runFlow(mode);
    }
  } finally {
    await new Promise((resolve, reject) => {
      mockServer.close((error) => (error ? reject(error) : resolve()));
    });
  }
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

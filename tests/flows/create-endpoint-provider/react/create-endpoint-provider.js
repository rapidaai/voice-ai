require("../../react-environment");

const {
  ConnectionConfig,
  CreateEndpoint,
  CreateEndpointProviderModel,
  EndpointAttribute,
  EndpointProviderModelAttribute,
  GetEndpoint,
  Metadata,
  UpdateEndpointVersion,
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

function createModelAttribute(provider, model, description) {
  const option = new Metadata();
  option.setKey("model");
  option.setValue(model);

  const attribute = new EndpointProviderModelAttribute();
  attribute.setDescription(description);
  attribute.setModelprovidername(provider);
  attribute.addEndpointmodeloptions(option);
  return attribute;
}

function createEndpoint(connection, model, endpoint, auth) {
  return new Promise((resolve, reject) => {
    CreateEndpoint(connection, model, endpoint, [], auth, (error, response) => {
      if (error) {
        reject(error);
        return;
      }
      resolve(response);
    });
  });
}

function createEndpointProviderModel(connection, endpointID, model, auth) {
  return new Promise((resolve, reject) => {
    CreateEndpointProviderModel(
      connection,
      endpointID,
      model,
      auth,
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

function getEndpoint(connection, endpointID, auth) {
  return new Promise((resolve, reject) => {
    GetEndpoint(
      connection,
      endpointID,
      undefined,
      auth,
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

function deployEndpointVersion(connection, endpointID, modelID, auth) {
  return new Promise((resolve, reject) => {
    UpdateEndpointVersion(
      connection,
      endpointID,
      modelID,
      auth,
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

function assertModel(model, endpointID, modelID, provider, optionValue) {
  if (
    !model?.getId() ||
    model.getEndpointid() !== endpointID ||
    (modelID && model.getId() !== modelID) ||
    model.getModelprovidername() !== provider
  ) {
    throw new Error(
      `${provider} endpoint provider model returned unexpected data`,
    );
  }

  if (optionValue) {
    const options = model.getEndpointmodeloptionsList();
    if (
      options.length !== 1 ||
      options[0].getKey() !== "model" ||
      options[0].getValue() !== optionValue
    ) {
      throw new Error(`${provider} endpoint model option was not returned`);
    }
  }
}

function assertEndpoint(endpoint, endpointID, endpointName, modelID) {
  if (
    !endpoint ||
    endpoint.getId() !== endpointID ||
    endpoint.getOrganizationid() !== requireValue("FLOW_FIXTURE_ID") ||
    endpoint.getProjectid() !== requireValue("FLOW_PROJECT_ID") ||
    endpoint.getName() !== endpointName ||
    endpoint.getEndpointprovidermodelid() !== modelID
  ) {
    throw new Error("endpoint lookup returned unexpected data");
  }
}

function authentication() {
  return {
    name: "personal access token",
    auth: ConnectionConfig.WithPersonalToken({
      Authorization: requireValue("FLOW_TOKEN"),
      AuthId: requireValue("FLOW_FIXTURE_ID"),
      ProjectId: requireValue("FLOW_PROJECT_ID"),
    }),
  };
}

async function runFlow(mode, endpointNumber) {
  const connection = new ConnectionConfig(
    { web: webEndpoint },
    process.env.FLOW_SDK_DEBUG === "true",
  );
  const endpointName = `${requireValue("FLOW_ENDPOINT")} ${endpointNumber}`;
  const endpointAttribute = new EndpointAttribute();
  endpointAttribute.setName(endpointName);
  endpointAttribute.setDescription("CI React SDK endpoint");
  endpointAttribute.setVisibility("private");

  const initialModel = createModelAttribute(
    "openai",
    "gpt-4o-mini",
    "CI initial endpoint provider model",
  );
  const endpointResponse = await createEndpoint(
    connection,
    initialModel,
    endpointAttribute,
    mode.auth,
  );
  requireSuccess(endpointResponse, "endpoint creation");

  const endpoint = endpointResponse.getData();
  if (
    !endpoint?.getId() ||
    endpoint.getOrganizationid() !== requireValue("FLOW_FIXTURE_ID") ||
    endpoint.getProjectid() !== requireValue("FLOW_PROJECT_ID") ||
    endpoint.getName() !== endpointName
  ) {
    throw new Error("endpoint creation returned unexpected ownership data");
  }

  const initialModelID = endpoint.getEndpointprovidermodel().getId();
  assertModel(
    endpoint.getEndpointprovidermodel(),
    endpoint.getId(),
    initialModelID,
    "openai",
  );

  const initialGetResponse = await getEndpoint(
    connection,
    endpoint.getId(),
    mode.auth,
  );
  requireSuccess(initialGetResponse, "initial endpoint lookup");
  const initialEndpoint = initialGetResponse.getData();
  assertEndpoint(initialEndpoint, endpoint.getId(), endpointName, initialModelID);
  assertModel(
    initialEndpoint.getEndpointprovidermodel(),
    endpoint.getId(),
    initialModelID,
    "openai",
    "gpt-4o-mini",
  );

  const secondaryModel = createModelAttribute(
    "anthropic",
    "claude-3-5-sonnet",
    "CI secondary endpoint provider model",
  );
  const providerResponse = await createEndpointProviderModel(
    connection,
    endpoint.getId(),
    secondaryModel,
    mode.auth,
  );
  requireSuccess(providerResponse, "endpoint provider model creation");
  const secondaryModelID = providerResponse.getData().getId();
  assertModel(
    providerResponse.getData(),
    endpoint.getId(),
    secondaryModelID,
    "anthropic",
  );

  const deployResponse = await deployEndpointVersion(
    connection,
    endpoint.getId(),
    secondaryModelID,
    mode.auth,
  );
  requireSuccess(deployResponse, "endpoint version deployment");

  const deployedGetResponse = await getEndpoint(
    connection,
    endpoint.getId(),
    mode.auth,
  );
  requireSuccess(deployedGetResponse, "deployed endpoint lookup");
  const deployedEndpoint = deployedGetResponse.getData();
  assertEndpoint(
    deployedEndpoint,
    endpoint.getId(),
    endpointName,
    secondaryModelID,
  );
  assertModel(
    deployedEndpoint.getEndpointprovidermodel(),
    endpoint.getId(),
    secondaryModelID,
    "anthropic",
    "claude-3-5-sonnet",
  );

  console.log(
    `React SDK endpoint lifecycle ${endpointNumber} passed with ${mode.name}`,
  );
}

async function main() {
  const endpointCount = Number.parseInt(requireValue("FLOW_EXPECTED_COUNT"), 10);
  if (!Number.isInteger(endpointCount) || endpointCount < 1) {
    throw new Error("FLOW_EXPECTED_COUNT must be a positive integer");
  }

  const mode = authentication();
  for (
    let endpointNumber = 1;
    endpointNumber <= endpointCount;
    endpointNumber += 1
  ) {
    await runFlow(mode, endpointNumber);
  }
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

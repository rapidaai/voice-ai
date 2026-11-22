require("../../react-environment");

const http = require("node:http");
const {
  ConnectionConfig,
  CreateEndpoint,
  CreateProjectCredential,
  Criteria,
  EndpointAttribute,
  EndpointDefinition,
  EndpointProviderModelAttribute,
  GetRole,
  GetRoleRequest,
  GetAllProjectCredential,
  Invoke,
  InvokeRequest,
  ListRoles,
  ListRolesRequest,
  Metadata,
  Paginate,
  TextChatCompletePrompt,
  TextPrompt,
  UpdateEndpointVersion,
  Variable,
} = require("@rapidaai/react");
const { Any } = require("google-protobuf/google/protobuf/any_pb");
const { StringValue } = require("google-protobuf/google/protobuf/wrappers_pb");

const webEndpoint = process.env.WEB_API_REACT_ENDPOINT || "http://web-api:9001";
const expectedReply = "runtime project API key invocation passed";
const mockAuthorization = "Bearer ci-flow-custom-llm";

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

async function expectUnavailable(action, invoke) {
  try {
    await invoke();
    throw new Error(`${action} unexpectedly succeeded`);
  } catch (error) {
    if (error?.message === `${action} unexpectedly succeeded`) {
      throw error;
    }
    if (error?.code === 14) {
      console.log(`${action} is unavailable under the enforced secret policy`);
      return;
    }
    throw new Error(`${action} returned an unexpected error`, { cause: error });
  }
}

function metadata(key, value) {
  const option = new Metadata();
  option.setKey(key);
  option.setValue(value);
  return option;
}

function stringAny(value) {
  const wrapped = new StringValue();
  wrapped.setValue(value);
  const packed = new Any();
  packed.pack(
    wrapped.serializeBinary(),
    "type.googleapis.com/google.protobuf.StringValue",
  );
  return packed;
}

function createEndpoint(connection, model, endpoint, auth) {
  return new Promise((resolve, reject) => {
    CreateEndpoint(connection, model, endpoint, [], auth, (error, response) => {
      if (error) reject(error);
      else resolve(response);
    });
  });
}

function deployEndpoint(connection, endpointID, modelID, auth) {
  return new Promise((resolve, reject) => {
    UpdateEndpointVersion(
      connection,
      endpointID,
      modelID,
      auth,
      (error, response) => {
        if (error) reject(error);
        else resolve(response);
      },
    );
  });
}

function createProjectCredential(connection, roleID, auth) {
  return new Promise((resolve, reject) => {
    CreateProjectCredential(
      connection,
      requireValue("FLOW_PROJECT_ID"),
      requireValue("FLOW_API_KEY_NAME"),
      roleID,
      (error, response) => {
        if (error) reject(error);
        else resolve(response);
      },
      auth,
    );
  });
}

function listProjectCredentials(connection, auth) {
  return new Promise((resolve, reject) => {
    GetAllProjectCredential(
      connection,
      requireValue("FLOW_PROJECT_ID"),
      (error, response) => {
        if (error) reject(error);
        else resolve(response);
      },
      auth,
    );
  });
}

function roleCriteria(key, value) {
  const criteria = new Criteria();
  criteria.setKey(key);
  criteria.setLogic(key === "key" ? "contains" : "=");
  criteria.setValue(value);
  return criteria;
}

async function findAutomationRole(connection, auth) {
  const request = new ListRolesRequest();
  const paginate = new Paginate();
  paginate.setPage(1);
  paginate.setPagesize(100);
  request.setPaginate(paginate);
  request.setCriteriasList([
    roleCriteria("scope", "PROJECT"),
    roleCriteria("subject", "PROJECT_TOKEN"),
    roleCriteria("key", "automation_client"),
  ]);

  const response = await ListRoles(connection, request, auth);
  requireSuccess(response, "runtime role lookup");
  const roles = response.getRolesList();
  if (roles.length !== 1 || roles[0].getKey() !== "automation_client") {
    throw new Error(`automation role lookup returned ${roles.length} roles`);
  }
  const detailRequest = new GetRoleRequest();
  detailRequest.setRoledefinitionid(roles[0].getId());
  const detailResponse = await GetRole(connection, detailRequest, auth);
  requireSuccess(detailResponse, "runtime role detail lookup");
  const role = detailResponse.getRole();
  const permissions = role.getPermissionsList().map((item) => item.getKey());
  if (
    !permissions.includes("project.runtime.invoke") ||
    !permissions.includes("project.provider_credentials.use")
  ) {
    throw new Error(
      `automation role omitted runtime permissions: ${permissions.join(",")}`,
    );
  }
  return role;
}

function startMockProvider(port) {
  let invocations = 0;
  const server = http.createServer((request, response) => {
    if (request.method !== "POST" || request.url !== "/v1/chat/completions") {
      response.writeHead(404, { "Content-Type": "application/json" });
      response.end(JSON.stringify({ error: { message: "not found" } }));
      return;
    }

    let body = "";
    request.setEncoding("utf8");
    request.on("data", (chunk) => {
      body += chunk;
    });
    request.on("end", () => {
      const payload = JSON.parse(body);
      const userMessage = payload.messages?.find((item) => item.role === "user");
      if (
        request.headers.authorization !== mockAuthorization ||
        payload.model !== "ci-runtime-model" ||
        userMessage?.content !== "Invoke the deployed endpoint"
      ) {
        response.writeHead(400, { "Content-Type": "application/json" });
        response.end(
          JSON.stringify({ error: { message: "unexpected provider request" } }),
        );
        return;
      }

      invocations += 1;
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(
        JSON.stringify({
          id: "chatcmpl-ci-flow",
          object: "chat.completion",
          created: 1,
          model: "ci-runtime-model",
          choices: [
            {
              index: 0,
              message: { role: "assistant", content: expectedReply },
              finish_reason: "stop",
            },
          ],
          usage: { prompt_tokens: 4, completion_tokens: 6, total_tokens: 10 },
        }),
      );
    });
  });

  return new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(port, "0.0.0.0", () => {
      resolve({ server, invocationCount: () => invocations });
    });
  });
}

function closeServer(server) {
  return new Promise((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()));
  });
}

async function main() {
  const port = Number.parseInt(requireValue("FLOW_MOCK_PORT"), 10);
  if (!Number.isInteger(port) || port < 1) {
    throw new Error("FLOW_MOCK_PORT must be a positive integer");
  }
  const mock = await startMockProvider(port);
  try {
    const connection = new ConnectionConfig(
      { web: webEndpoint },
      process.env.FLOW_SDK_DEBUG === "true",
    );
    const userAuth = ConnectionConfig.WithPersonalToken({
      Authorization: requireValue("FLOW_TOKEN"),
      AuthId: requireValue("FLOW_FIXTURE_ID"),
      ProjectId: requireValue("FLOW_PROJECT_ID"),
    });

    const automationRole = await findAutomationRole(connection, userAuth);
    await expectUnavailable("project API key creation", () =>
      createProjectCredential(connection, automationRole.getId(), userAuth),
    );
    await expectUnavailable("project API key secret inventory", () =>
      listProjectCredentials(connection, userAuth),
    );

    const prompt = new TextChatCompletePrompt();
    const promptVariable = new Variable();
    promptVariable.setName("message");
    promptVariable.setType("string");
    prompt.addPromptvariables(promptVariable);
    const userPrompt = new TextPrompt();
    userPrompt.setRole("user");
    userPrompt.setContent("{{ message }}");
    prompt.addPrompt(userPrompt);

    const model = new EndpointProviderModelAttribute();
    model.setDescription("CI custom LLM deployment for runtime-key invocation");
    model.setModelprovidername("custom-llm");
    model.setChatcompleteprompt(prompt);
    model.setEndpointmodeloptionsList([
      metadata("rapida.credential_id", requireValue("FLOW_VAULT_ID")),
      metadata("model.name", "ci-runtime-model"),
    ]);

    const endpointAttribute = new EndpointAttribute();
    endpointAttribute.setName(requireValue("FLOW_ENDPOINT"));
    endpointAttribute.setDescription("CI endpoint invoked by a runtime project key");
    endpointAttribute.setVisibility("private");
    const endpointResponse = await createEndpoint(
      connection,
      model,
      endpointAttribute,
      userAuth,
    );
    requireSuccess(endpointResponse, "endpoint creation");
    const endpoint = endpointResponse.getData();
    const endpointID = endpoint?.getId();
    const modelID = endpoint?.getEndpointprovidermodel()?.getId();
    if (!endpointID || !modelID) {
      throw new Error("endpoint creation returned no endpoint or model id");
    }

    const deploymentResponse = await deployEndpoint(
      connection,
      endpointID,
      modelID,
      userAuth,
    );
    requireSuccess(deploymentResponse, "endpoint deployment");

    const invokeRequest = new InvokeRequest();
    const definition = new EndpointDefinition();
    definition.setEndpointid(endpointID);
    definition.setVersion(modelID);
    invokeRequest.setEndpoint(definition);
    invokeRequest
      .getArgsMap()
      .set("message", stringAny("Invoke the deployed endpoint"));
    const projectAuth = ConnectionConfig.WithSDK({
      ApiKey: requireValue("FLOW_API_KEY"),
    });
    const invokeResponse = await Invoke(connection, invokeRequest, projectAuth);
    requireSuccess(invokeResponse, "deployment invocation");
    if (
      invokeResponse.getDataList().length !== 1 ||
      invokeResponse.getDataList()[0] !== expectedReply ||
      !invokeResponse.getRequestid() ||
      mock.invocationCount() !== 1
    ) {
      throw new Error("deployment invocation returned unexpected data");
    }

    console.log(
      "React SDK enforced project-key creation policy and invoked a deployment with an existing automation key",
    );
  } finally {
    await closeServer(mock.server);
  }
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);

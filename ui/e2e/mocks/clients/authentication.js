const {
  fulfillGrpcWeb,
  protobufBoolean,
  protobufBytes,
  protobufString,
  protobufUnsignedInteger,
} = require('../grpc-web');

const AUTHENTICATION_PATHS = Object.freeze({
  signIn: '/web_api.AuthenticationService/Authenticate',
  signUp: '/web_api.AuthenticationService/RegisterUser',
  authorize: '/web_api.AuthenticationService/Authorize',
  forgotPassword: '/web_api.AuthenticationService/ForgotPassword',
  createPassword: '/web_api.AuthenticationService/CreatePassword',
});

const responseContracts = Object.freeze({
  signIn: { includesAuthentication: true, errorField: 4 },
  signUp: { includesAuthentication: true, errorField: 4 },
  authorize: { includesAuthentication: true, errorField: 4 },
  forgotPassword: { includesAuthentication: false, errorField: 3 },
  createPassword: { includesAuthentication: false, errorField: 3 },
});

function authenticationMessage({ organization = true, project = true } = {}) {
  const user = Buffer.concat([
    protobufUnsignedInteger(1, 101),
    protobufString(2, 'E2E User'),
    protobufString(3, 'user@example.test'),
  ]);
  const token = Buffer.concat([
    protobufUnsignedInteger(1, 1),
    protobufString(2, 'token-e2e'),
  ]);
  const organizationRole = Buffer.concat([
    protobufUnsignedInteger(1, 201),
    protobufUnsignedInteger(2, 202),
    protobufString(3, 'admin'),
    protobufString(4, 'E2E Organization'),
  ]);
  const projectRole = Buffer.concat([
    protobufUnsignedInteger(1, 301),
    protobufUnsignedInteger(2, 302),
    protobufString(3, 'admin'),
    protobufString(4, 'E2E Project'),
  ]);

  const authentication = [protobufBytes(1, user), protobufBytes(2, token)];
  if (organization) authentication.push(protobufBytes(3, organizationRole));
  if (project) authentication.push(protobufBytes(4, projectRole));
  return Buffer.concat(authentication);
}

function operationResponse(operation, scenario) {
  const contract = responseContracts[operation];
  if (!contract)
    throw new Error(`Unknown authentication operation: ${operation}`);

  if (scenario.success) {
    const fields = [protobufBoolean(2, true)];
    if (contract.includesAuthentication) {
      fields.push(
        protobufBytes(3, authenticationMessage(scenario.authentication)),
      );
    }
    return Buffer.concat(fields);
  }

  const error = protobufString(
    3,
    scenario.message || 'Unable to process your request.',
  );
  return protobufBytes(contract.errorField, error);
}

async function mockAuthenticationOperation(
  page,
  operation,
  scenario = { success: true },
) {
  const path = AUTHENTICATION_PATHS[operation];
  if (!path) throw new Error(`Unknown authentication operation: ${operation}`);

  await page.route(`**${path}`, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    const currentScenario =
      typeof scenario === 'function' ? scenario() : scenario;
    return fulfillGrpcWeb(route, operationResponse(operation, currentScenario));
  });
}

async function installAuthenticationClientMock(page) {
  for (const operation of Object.keys(AUTHENTICATION_PATHS)) {
    await mockAuthenticationOperation(page, operation);
  }
}

module.exports = {
  AUTHENTICATION_PATHS,
  installAuthenticationClientMock,
  mockAuthenticationOperation,
};

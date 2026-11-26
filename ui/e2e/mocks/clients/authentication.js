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
  const hasProject = organization && project;
  const organizationPermissions = [
    'organization.members.read',
    'organization.members.invite',
    'organization.members.remove',
    'organization.access.manage',
    'organization.roles.read',
    'organization.project_access_tokens.revoke',
  ].map((key, index) =>
    Buffer.concat([
      protobufUnsignedInteger(1, 1000 + index),
      protobufString(2, key),
      protobufString(3, 'organization'),
    ]),
  );
  const projectPermissions = [
    'project.read',
    'project.access_tokens.revoke',
  ].map((key, index) =>
    Buffer.concat([
      protobufUnsignedInteger(1, 2000 + index),
      protobufString(2, key),
      protobufString(3, 'project'),
    ]),
  );
  const organizationRole = Buffer.concat([
    protobufUnsignedInteger(1, 102),
    protobufString(2, 'organization_admin'),
    protobufString(3, 'Organization Admin'),
    protobufString(5, 'organization'),
    ...organizationPermissions.map(permission => protobufBytes(8, permission)),
  ]);
  const projectRole = Buffer.concat([
    protobufUnsignedInteger(1, 110),
    protobufString(2, 'project_admin'),
    protobufString(3, 'Project Admin'),
    protobufString(5, 'project'),
    ...projectPermissions.map(permission => protobufBytes(8, permission)),
  ]);
  const selectedTenant = Buffer.concat([
    ...(organization ? [protobufUnsignedInteger(1, 202)] : []),
    ...(hasProject ? [protobufUnsignedInteger(2, 302)] : []),
  ]);
  const principal = Buffer.concat([
    protobufBytes(
      1,
      Buffer.concat([
        protobufString(1, 'user'),
        protobufUnsignedInteger(2, 101),
      ]),
    ),
    protobufBytes(3, selectedTenant),
    protobufBytes(
      4,
      Buffer.concat([
        protobufString(1, 'user_session'),
        protobufBytes(
          3,
          Buffer.concat([protobufString(1, 'token'), protobufString(2, '1')]),
        ),
      ]),
    ),
  ]);
  const user = Buffer.concat([
    protobufUnsignedInteger(1, 101),
    protobufString(2, 'E2E User'),
    protobufString(3, 'user@example.test'),
    protobufString(4, 'active'),
  ]);
  const projectValue = Buffer.concat([
    protobufUnsignedInteger(1, 302),
    protobufString(2, 'E2E Project'),
    protobufBytes(3, projectRole),
  ]);
  const organizationValue = Buffer.concat([
    protobufUnsignedInteger(1, 202),
    protobufString(2, 'E2E Organization'),
    protobufBytes(3, organizationRole),
    ...(hasProject ? [protobufBytes(5, projectValue)] : []),
  ]);

  const authentication = [protobufBytes(1, principal), protobufBytes(2, user)];
  if (organization) authentication.push(protobufBytes(3, organizationValue));
  return Buffer.concat(authentication);
}

function tokenMessage() {
  return Buffer.concat([
    protobufUnsignedInteger(1, 1),
    protobufString(2, 'token-e2e'),
    protobufString(3, 'Bearer'),
  ]);
}

function operationResponse(operation, scenario) {
  const contract = responseContracts[operation];
  if (!contract)
    throw new Error(`Unknown authentication operation: ${operation}`);

  if (scenario.success) {
    const fields = [protobufBoolean(2, true)];
    if (contract.includesAuthentication) {
      fields.push(
        protobufBytes(6, authenticationMessage(scenario.authentication)),
      );
      fields.push(protobufBytes(7, tokenMessage()));
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

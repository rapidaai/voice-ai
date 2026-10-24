const {
  fulfillGrpcWeb,
  protobufBoolean,
  protobufBytes,
  protobufString,
  protobufUnsignedInteger,
} = require('../grpc-web');

const WORKSPACE_LIST_PATHS = [
  '**/web_api.AuthenticationService/GetAllUser',
  '**/web_api.ProjectService/GetAllProject',
];
const ORGANIZATION_PATH = '**/web_api.OrganizationService/GetOrganization';
const WORKSPACE_CREATION_PATHS = Object.freeze({
  organization: '/web_api.OrganizationService/CreateOrganization',
  project: '/web_api.ProjectService/CreateProject',
});

const EMPTY_WORKSPACE_LIST_RESPONSE = Buffer.concat([
  protobufBoolean(2, true),
  protobufBytes(5, Buffer.alloc(0)),
]);

const ORGANIZATION = Buffer.concat([
  protobufUnsignedInteger(1, 1),
  protobufString(2, 'E2E Organization'),
  protobufString(3, 'Workspace used by deterministic browser journeys.'),
  protobufString(4, 'Software'),
  protobufString(5, 'e2e@example.test'),
  protobufString(6, '1-10'),
]);

const ORGANIZATION_RESPONSE = Buffer.concat([
  protobufBoolean(2, true),
  protobufBytes(3, ORGANIZATION),
]);

const WORKSPACE_CREATION_RESPONSE = protobufBoolean(2, true);

async function mockWorkspaceCreation(page, resource, onCreate) {
  const path = WORKSPACE_CREATION_PATHS[resource];
  if (!path) throw new Error(`Unknown workspace resource: ${resource}`);

  await page.route(`**${path}`, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    onCreate?.();
    return fulfillGrpcWeb(route, WORKSPACE_CREATION_RESPONSE);
  });
}

async function installWorkspaceClientMock(page) {
  for (const path of WORKSPACE_LIST_PATHS) {
    await page.route(path, route => {
      if (route.request().method() !== 'POST') return route.fallback();
      return fulfillGrpcWeb(route, EMPTY_WORKSPACE_LIST_RESPONSE);
    });
  }

  await page.route(ORGANIZATION_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, ORGANIZATION_RESPONSE);
  });
}

module.exports = {
  ORGANIZATION_PATH,
  WORKSPACE_CREATION_PATHS,
  WORKSPACE_LIST_PATHS,
  installWorkspaceClientMock,
  mockWorkspaceCreation,
};

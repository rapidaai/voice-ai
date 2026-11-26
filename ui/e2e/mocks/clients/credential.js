const {
  fulfillGrpcWeb,
  protobufBoolean,
  protobufBytes,
} = require('../grpc-web');
const { Timestamp } = require('google-protobuf/google/protobuf/timestamp_pb');

const previousWindow = global.window;
const previousSelf = global.self;
global.window = global;
global.self = global;
const {
  CreateProjectCredentialResponse,
  GetAllProjectCredentialResponse,
  GetAllOrganizationCredentialResponse,
  Paginated,
  ProjectCredential,
  RevokeProjectCredentialResponse,
  VaultCredential,
} = require('@rapidaai/react');
if (previousWindow === undefined) delete global.window;
else global.window = previousWindow;
if (previousSelf === undefined) delete global.self;
else global.self = previousSelf;

const CREDENTIAL_LIST_PATH =
  '**/vault_api.VaultService/GetAllOrganizationCredential';
const PROJECT_CREDENTIAL_LIST_PATH =
  '**/web_api.ProjectService/GetAllProjectCredential';
const PROJECT_CREDENTIAL_CREATE_PATH =
  '**/web_api.ProjectService/CreateProjectCredential';
const PROJECT_CREDENTIAL_REVOKE_PATH =
  '**/web_api.ProjectService/RevokeProjectCredential';
const EMPTY_CREDENTIAL_LIST_RESPONSE = Buffer.concat([
  protobufBoolean(2, true),
  protobufBytes(5, Buffer.alloc(0)),
]);

function buildOrganizationCredentialListResponse() {
  const credential = new VaultCredential();
  credential.setId('2301664620831572200');
  credential.setName('OpenAI test credential');
  credential.setProvider('openai');
  credential.setStatus('active');

  const response = new GetAllOrganizationCredentialResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.addData(credential);
  return Buffer.from(response.serializeBinary());
}

function buildProjectCredentialCreateResponse() {
  const credential = new ProjectCredential();
  credential.setId('2301664620831572300');
  credential.setProjectid('302');
  credential.setOrganizationid('202');
  credential.setName('publishable key');
  credential.setKey('rpx_e2e_project_credential');
  credential.setRoledefinitionid('516');
  credential.setRolename('Observer Client');
  credential.setStatus('ACTIVE');

  const response = new CreateProjectCredentialResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setData(credential);
  return Buffer.from(response.serializeBinary());
}

function projectCredential(id, name, key, roleDefinitionId, roleName) {
  const createdDate = new Timestamp();
  createdDate.fromDate(new Date('2026-11-08T10:00:00.000Z'));
  const credential = new ProjectCredential();
  credential.setId(id);
  credential.setProjectid('302');
  credential.setOrganizationid('202');
  credential.setName(name);
  credential.setKey(key);
  credential.setRoledefinitionid(roleDefinitionId);
  credential.setRolename(roleName);
  credential.setStatus('ACTIVE');
  credential.setCreateddate(createdDate);
  return credential;
}

function buildProjectCredentialListResponse(credentials) {
  const paginated = new Paginated();
  paginated.setCurrentpage(1);
  paginated.setTotalitem(credentials.length);

  const response = new GetAllProjectCredentialResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList(credentials);
  response.setPaginated(paginated);
  return Buffer.from(response.serializeBinary());
}

function buildProjectCredentialRevokeResponse() {
  const response = new RevokeProjectCredentialResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setId('2301664620831572301');
  return Buffer.from(response.serializeBinary());
}

async function mockOrganizationCredentials(page) {
  await page.route(CREDENTIAL_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildOrganizationCredentialListResponse());
  });
}

async function installCredentialClientMock(page) {
  const projectCredentials = [
    projectCredential(
      '2301664620831572301',
      'Web widget',
      'rpx_e2e_web_widget',
      '514',
      'Runtime Invoker',
    ),
    projectCredential(
      '2301664620831572302',
      'Phone assistant',
      'rpx_e2e_phone_assistant',
      '516',
      'Observer Client',
    ),
  ];

  await page.route(CREDENTIAL_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, EMPTY_CREDENTIAL_LIST_RESPONSE);
  });
  await page.route(PROJECT_CREDENTIAL_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(
      route,
      buildProjectCredentialListResponse(projectCredentials),
    );
  });
  await page.route(PROJECT_CREDENTIAL_CREATE_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildProjectCredentialCreateResponse());
  });
  await page.route(PROJECT_CREDENTIAL_REVOKE_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    projectCredentials.splice(
      projectCredentials.findIndex(
        credential => credential.getId() === '2301664620831572301',
      ),
      1,
    );
    return fulfillGrpcWeb(route, buildProjectCredentialRevokeResponse());
  });
}

module.exports = {
  CREDENTIAL_LIST_PATH,
  PROJECT_CREDENTIAL_LIST_PATH,
  PROJECT_CREDENTIAL_CREATE_PATH,
  PROJECT_CREDENTIAL_REVOKE_PATH,
  buildProjectCredentialCreateResponse,
  buildProjectCredentialListResponse,
  buildProjectCredentialRevokeResponse,
  buildOrganizationCredentialListResponse,
  installCredentialClientMock,
  mockOrganizationCredentials,
};

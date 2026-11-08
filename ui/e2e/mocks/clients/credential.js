const {
  fulfillGrpcWeb,
  protobufBoolean,
  protobufBytes,
} = require('../grpc-web');

const previousWindow = global.window;
const previousSelf = global.self;
global.window = global;
global.self = global;
const {
  GetAllOrganizationCredentialResponse,
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

async function mockOrganizationCredentials(page) {
  await page.route(CREDENTIAL_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildOrganizationCredentialListResponse());
  });
}

async function installCredentialClientMock(page) {
  for (const path of [CREDENTIAL_LIST_PATH, PROJECT_CREDENTIAL_LIST_PATH]) {
    await page.route(path, route => {
      if (route.request().method() !== 'POST') return route.fallback();
      return fulfillGrpcWeb(route, EMPTY_CREDENTIAL_LIST_RESPONSE);
    });
  }
}

module.exports = {
  CREDENTIAL_LIST_PATH,
  PROJECT_CREDENTIAL_LIST_PATH,
  buildOrganizationCredentialListResponse,
  installCredentialClientMock,
  mockOrganizationCredentials,
};

const {
  fulfillGrpcWeb,
  protobufBoolean,
  protobufBytes,
  protobufString,
  protobufUnsignedInteger,
} = require('../grpc-web');

const ENDPOINT_LIST_PATH = '**/endpoint_api.EndpointService/GetAllEndpoint';
const ENDPOINT_DETAIL_PATH = '**/endpoint_api.EndpointService/GetEndpoint';
const ENDPOINT_LOG_LIST_PATH =
  '**/endpoint_api.EndpointService/GetAllEndpointLog';
const ENDPOINT_VERSION_LIST_PATH =
  '**/endpoint_api.EndpointService/GetAllEndpointProviderModel';
const ENDPOINT_ID = '2301664620831571968';
const ENDPOINT_PROVIDER_MODEL_ID = '2301664620831571999';
const EMPTY_ENDPOINT_LIST_RESPONSE = Buffer.concat([
  protobufBoolean(2, true),
  protobufBytes(5, Buffer.alloc(0)),
]);

const ENDPOINT_PROVIDER_MODEL = Buffer.concat([
  protobufUnsignedInteger(1, ENDPOINT_PROVIDER_MODEL_ID),
  protobufString(4, 'OpenAI'),
  protobufString(12, 'active'),
  protobufUnsignedInteger(19, ENDPOINT_ID),
  protobufString(20, 'GPT endpoint'),
]);

const ENDPOINT = Buffer.concat([
  protobufUnsignedInteger(1, ENDPOINT_ID),
  protobufString(2, 'active'),
  protobufString(3, 'project'),
  protobufUnsignedInteger(9, ENDPOINT_PROVIDER_MODEL_ID),
  protobufBytes(10, ENDPOINT_PROVIDER_MODEL),
  protobufString(16, 'English'),
  protobufString(18, 'Customer support analysis'),
  protobufString(19, 'Analyze completed support conversations.'),
]);

const ENDPOINT_DETAIL_RESPONSE = Buffer.concat([
  protobufBoolean(2, true),
  protobufBytes(3, ENDPOINT),
]);

function metric(name, value) {
  return Buffer.concat([protobufString(1, name), protobufString(2, value)]);
}

function timestamp(seconds) {
  return protobufUnsignedInteger(1, seconds);
}

function endpointLog({ id, source, status, latency, tokens, createdAt }) {
  return Buffer.concat([
    protobufUnsignedInteger(1, id),
    protobufUnsignedInteger(2, ENDPOINT_ID),
    protobufString(3, source),
    protobufString(15, status),
    protobufUnsignedInteger(19, ENDPOINT_PROVIDER_MODEL_ID),
    protobufUnsignedInteger(25, latency),
    protobufBytes(26, timestamp(createdAt)),
    protobufBytes(30, metric('agent_total_token', tokens)),
  ]);
}

const ENDPOINT_LOGS = [
  endpointLog({
    id: '2301664620831572101',
    source: 'playground',
    status: 'SUCCESS',
    latency: '184000000',
    tokens: '286',
    createdAt: '1780144200',
  }),
  endpointLog({
    id: '2301664620831572100',
    source: 'assistant',
    status: 'SUCCESS',
    latency: '231000000',
    tokens: '412',
    createdAt: '1780140600',
  }),
  endpointLog({
    id: '2301664620831572099',
    source: 'api',
    status: 'FAILED',
    latency: '492000000',
    tokens: '94',
    createdAt: '1780137000',
  }),
  endpointLog({
    id: '2301664620831572098',
    source: 'assistant',
    status: 'SUCCESS',
    latency: '167000000',
    tokens: '327',
    createdAt: '1780133400',
  }),
  endpointLog({
    id: '2301664620831572097',
    source: 'api',
    status: 'SUCCESS',
    latency: '205000000',
    tokens: '251',
    createdAt: '1780129800',
  }),
];

const ENDPOINT_LOG_LIST_RESPONSE = Buffer.concat([
  protobufBoolean(2, true),
  ...ENDPOINT_LOGS.map(log => protobufBytes(3, log)),
  protobufBytes(
    5,
    Buffer.concat([
      protobufUnsignedInteger(1, 1),
      protobufUnsignedInteger(2, 128),
    ]),
  ),
]);
const EMPTY_ENDPOINT_VERSION_LIST_RESPONSE = protobufBoolean(2, true);

async function installEndpointClientMock(page) {
  await page.route(ENDPOINT_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, EMPTY_ENDPOINT_LIST_RESPONSE);
  });
  await page.route(ENDPOINT_DETAIL_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, ENDPOINT_DETAIL_RESPONSE);
  });
  await page.route(ENDPOINT_LOG_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, ENDPOINT_LOG_LIST_RESPONSE);
  });
  await page.route(ENDPOINT_VERSION_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, EMPTY_ENDPOINT_VERSION_LIST_RESPONSE);
  });
}

module.exports = {
  ENDPOINT_DETAIL_PATH,
  ENDPOINT_LIST_PATH,
  ENDPOINT_LOG_LIST_PATH,
  ENDPOINT_VERSION_LIST_PATH,
  installEndpointClientMock,
};

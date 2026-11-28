const {
  fulfillGrpcWeb,
  protobufBoolean,
  protobufBytes,
  protobufString,
  protobufUnsignedInteger,
} = require('../grpc-web');
const { Struct } = require('google-protobuf/google/protobuf/struct_pb.js');
const {
  Timestamp,
} = require('google-protobuf/google/protobuf/timestamp_pb.js');

const LLM_ACTIVITY_LIST_PATH =
  '**/integration_api.AuditLoggingService/GetAllAuditLog';
const LLM_ACTIVITY_DETAIL_PATH =
  '**/integration_api.AuditLoggingService/GetAuditLog';

const ACTIVITY_LIST_PATHS = [
  '**/integration_api.AuditLoggingService/GetAllAuditLog',
  '**/assistant_api.AssistantService/GetAllAssistantHTTPLog',
  '**/assistant_api.AssistantService/GetAllAssistantToolLog',
  '**/assistant_api.AssistantService/GetAllMessage',
  '**/observability_api.ObservabilityService/GetAllTelemetry',
];

const EMPTY_ACTIVITY_LIST_RESPONSE = Buffer.concat([
  protobufBoolean(2, true),
  protobufBytes(5, Buffer.alloc(0)),
]);

async function installActivityClientMock(page) {
  for (const path of ACTIVITY_LIST_PATHS) {
    await page.route(path, route => {
      if (route.request().method() !== 'POST') return route.fallback();
      return fulfillGrpcWeb(route, EMPTY_ACTIVITY_LIST_RESPONSE);
    });
  }
}

function buildLLMActivity() {
  const createdAt = new Timestamp();
  createdAt.fromDate(new Date('2026-11-25T16:30:00Z'));
  const request = Struct.fromJavaScript({
    model: 'gpt-4.1',
    messages: [{ role: 'user', content: 'Summarize the trace' }],
  });
  const response = Struct.fromJavaScript({
    role: 'assistant',
    content: 'Trace completed normally.',
  });

  return Buffer.concat([
    protobufUnsignedInteger(1, 9001),
    protobufUnsignedInteger(4, 200),
    protobufUnsignedInteger(5, 8_900_000),
    protobufString(6, 'SUCCESS'),
    protobufBytes(11, Buffer.from(createdAt.serializeBinary())),
    protobufBytes(13, Buffer.from(request.serializeBinary())),
    protobufBytes(14, Buffer.from(response.serializeBinary())),
  ]);
}

async function mockLLMActivity(page) {
  const activity = buildLLMActivity();
  const listResponse = Buffer.concat([
    protobufBoolean(2, true),
    protobufBytes(3, activity),
  ]);
  const detailResponse = Buffer.concat([
    protobufBoolean(2, true),
    protobufBytes(3, activity),
  ]);

  await page.route(LLM_ACTIVITY_LIST_PATH, route =>
    fulfillGrpcWeb(route, listResponse),
  );
  await page.route(LLM_ACTIVITY_DETAIL_PATH, route =>
    fulfillGrpcWeb(route, detailResponse),
  );
}

module.exports = {
  ACTIVITY_LIST_PATHS,
  installActivityClientMock,
  mockLLMActivity,
};

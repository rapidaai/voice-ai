const { Timestamp } = require('google-protobuf/google/protobuf/timestamp_pb');
const { fulfillGrpcWeb } = require('../grpc-web');

const previousWindow = global.window;
const previousSelf = global.self;
global.window = global;
global.self = global;
const {
  Assistant,
  AssistantConversation,
  AssistantConversationMessage,
  AssistantDashboard,
  AssistantDashboardBucket,
  AssistantDashboardDistribution,
  AssistantDashboardLatency,
  AssistantDashboardSummary,
  AssistantDashboardUsage,
  AssistantDebuggerDeployment,
  AssistantPhoneDeployment,
  AssistantProviderModel,
  DeploymentAudioProvider,
  GetAllAssistantConversationResponse,
  GetAllAssistantProviderResponse,
  GetAllAssistantResponse,
  GetAllConversationMessageResponse,
  GetAssistantConversationResponse,
  GetAssistantDashboardResponse,
  GetAssistantResponse,
  Metadata,
  Metric,
  Paginated,
  TextChatCompletePrompt,
  Variable,
} = require('@rapidaai/react');
if (previousWindow === undefined) delete global.window;
else global.window = previousWindow;
if (previousSelf === undefined) delete global.self;
else global.self = previousSelf;

const ASSISTANT_ID = '2301664620831571968';
const ASSISTANT_VERSION_ID = '2301664620831571999';
const ASSISTANT_CONVERSATION_ID = '2301664620831572101';
const ASSISTANT_SERVICE_PATH = '**/assistant_api.AssistantService/*';
const ASSISTANT_LIST_PATH = '**/assistant_api.AssistantService/GetAllAssistant';
const ASSISTANT_DETAIL_PATH = '**/assistant_api.AssistantService/GetAssistant';
const ASSISTANT_DASHBOARD_PATH =
  '**/assistant_api.AssistantService/GetAssistantDashboard';
const ASSISTANT_PROVIDER_LIST_PATH =
  '**/assistant_api.AssistantService/GetAllAssistantProvider';
const ASSISTANT_CONVERSATION_LIST_PATH =
  '**/assistant_api.AssistantService/GetAllAssistantConversation';
const ASSISTANT_CONVERSATION_DETAIL_PATH =
  '**/assistant_api.AssistantService/GetAssistantConversation';
const ASSISTANT_MESSAGE_LIST_PATH =
  '**/assistant_api.AssistantService/GetAllConversationMessage';

function timestamp(value) {
  const result = new Timestamp();
  result.fromDate(new Date(value));
  return result;
}

function paginated(totalItem = 1) {
  const result = new Paginated();
  result.setCurrentpage(1);
  result.setTotalitem(totalItem);
  return result;
}

function metric(name, value, description = '') {
  const result = new Metric();
  result.setName(name);
  result.setValue(value);
  result.setDescription(description);
  return result;
}

function metadata(key, value) {
  const result = new Metadata();
  result.setKey(key);
  result.setValue(value);
  return result;
}

function buildAssistant() {
  const provider = new AssistantProviderModel();
  provider.setId(ASSISTANT_VERSION_ID);
  provider.setAssistantid(ASSISTANT_ID);
  provider.setModelprovidername('OpenAI');
  provider.setDescription('Production support assistant');
  provider.setStatus('active');
  provider.setCreateddate(timestamp('2026-10-01T09:00:00Z'));
  provider.setUpdateddate(timestamp('2026-10-17T08:28:00Z'));

  const assistant = new Assistant();
  assistant.setId(ASSISTANT_ID);
  assistant.setName('Support bot');
  assistant.setDescription(
    'Answers product questions and resolves customer support requests.',
  );
  assistant.setStatus('active');
  assistant.setVisibility('private');
  assistant.setSource('dashboard');
  assistant.setProjectid('302');
  assistant.setOrganizationid('202');
  assistant.setAssistantprovider('MODEL');
  assistant.setAssistantproviderid(ASSISTANT_VERSION_ID);
  assistant.setAssistantprovidermodel(provider);
  assistant.setCreateddate(timestamp('2026-10-01T09:00:00Z'));
  assistant.setUpdateddate(timestamp('2026-10-17T08:28:00Z'));
  return assistant;
}

function buildAssistantListResponse(assistants = [buildAssistant()]) {
  const response = new GetAllAssistantResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList(assistants);
  response.setPaginated(paginated(assistants.length));
  return Buffer.from(response.serializeBinary());
}

function buildAssistantDetailResponse() {
  const response = new GetAssistantResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setData(buildAssistant());
  return Buffer.from(response.serializeBinary());
}

function buildDebuggerAssistantDetailResponse({
  withSpeechToText = true,
} = {}) {
  const assistant = buildAssistant();
  assistant.setName('Delivery concierge');

  const provider = assistant.getAssistantprovidermodel();
  const prompt = new TextChatCompletePrompt();
  prompt.setPromptvariablesList(
    [
      ['customer_name', 'string', 'Prashant'],
      [
        'delivery_instructions',
        'paragraph',
        'Leave the package with reception if no one answers.',
      ],
      ['order_context', 'json', '{ "priority": "express" }'],
    ].map(([name, type, defaultValue]) => {
      const item = new Variable();
      item.setName(name);
      item.setType(type);
      item.setDefaultvalue(defaultValue);
      return item;
    }),
  );
  provider.setTemplate(prompt);
  provider.setAssistantmodeloptionsList([
    metadata('model.name', 'gpt-4.1-mini'),
    metadata('model.temperature', '0.7'),
    metadata('model.top_p', '1'),
    metadata('rapida.credential_id', 'credential-secret-123'),
  ]);

  const audioProvider = (name, type, options) => {
    const audio = new DeploymentAudioProvider();
    audio.setAudioprovider(name);
    audio.setAudiotype(type);
    audio.setAudiooptionsList(
      options.map(([key, value]) => metadata(key, value)),
    );
    return audio;
  };
  const deployment = new AssistantDebuggerDeployment();
  if (withSpeechToText) {
    deployment.setInputaudio(
      audioProvider('Deepgram', 'speech-to-text', [
        ['listen.model', 'nova-3'],
        ['listen.language', 'en'],
      ]),
    );
  }
  deployment.setOutputaudio(
    audioProvider('Cartesia', 'text-to-speech', [
      ['speak.model', 'sonic-2'],
      ['speak.voice.id', 'Professional English'],
    ]),
  );
  assistant.setDebuggerdeployment(deployment);

  const phoneDeployment = new AssistantPhoneDeployment();
  phoneDeployment.setPhoneprovidername('Twilio');
  phoneDeployment.setInputaudio(
    audioProvider('Deepgram', 'speech-to-text', [
      ['listen.model', 'nova-3'],
      ['listen.language', 'en'],
    ]),
  );
  phoneDeployment.setOutputaudio(
    audioProvider('Cartesia', 'text-to-speech', [
      ['speak.model', 'sonic-2'],
      ['speak.voice.id', 'Professional English'],
    ]),
  );
  assistant.setPhonedeployment(phoneDeployment);

  const response = new GetAssistantResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setData(assistant);
  return Buffer.from(response.serializeBinary());
}

async function mockDebuggerAssistant(page, options) {
  await page.route(ASSISTANT_DETAIL_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildDebuggerAssistantDetailResponse(options));
  });
}

function buildAssistantDashboardResponse() {
  const summary = new AssistantDashboardSummary();
  summary.setTotalsessions(1284);
  summary.setActivesessions(18);
  summary.setCompletedsessions(1225);
  summary.setFailedsessions(41);
  summary.setTotalmessages(8426);
  summary.setUsermessages(4213);
  summary.setFailurerate(3.2);
  summary.setAveragesessiondurationseconds(154);

  const latency = new AssistantDashboardLatency();
  latency.setAveragems(742);
  latency.setSttms(118);
  latency.setEosms(205);
  latency.setTtsms(146);
  latency.setLlmms(273);

  const usage = new AssistantDashboardUsage();
  usage.setTotaltokens(1842000);
  usage.setSttdurationseconds(14820);
  usage.setTtsdurationseconds(12640);
  usage.setTotaldurationseconds(27460);

  const buckets = [
    ['2026-10-11T00:00:00Z', 82, 112, 198, 139, 261],
    ['2026-10-12T00:00:00Z', 108, 116, 203, 142, 269],
    ['2026-10-13T00:00:00Z', 136, 121, 211, 149, 284],
    ['2026-10-14T00:00:00Z', 124, 119, 207, 145, 278],
    ['2026-10-15T00:00:00Z', 161, 115, 201, 141, 266],
    ['2026-10-16T00:00:00Z', 153, 117, 204, 143, 271],
    ['2026-10-17T00:00:00Z', 96, 114, 200, 140, 264],
  ].map(([start, count, stt, eos, tts, llm]) => {
    const bucket = new AssistantDashboardBucket();
    bucket.setStartdate(timestamp(start));
    bucket.setEnddate(
      timestamp(new Date(new Date(start).getTime() + 86400000).toISOString()),
    );
    bucket.setMessagecount(count);
    bucket.setSttlatencyms(stt);
    bucket.setEoslatencyms(eos);
    bucket.setTtslatencyms(tts);
    bucket.setLlmlatencyms(llm);
    return bucket;
  });

  const distributions = values =>
    values.map(([name, count, percentage]) => {
      const item = new AssistantDashboardDistribution();
      item.setName(name);
      item.setCount(count);
      item.setPercentage(percentage);
      return item;
    });

  const dashboard = new AssistantDashboard();
  dashboard.setSummary(summary);
  dashboard.setLatency(latency);
  dashboard.setUsage(usage);
  dashboard.setBucketsList(buckets);
  dashboard.setSourcesList(
    distributions([
      ['Web widget', 672, 52.3],
      ['Phone', 408, 31.8],
      ['API', 204, 15.9],
    ]),
  );
  dashboard.setLanguagesList(
    distributions([
      ['English', 1036, 80.7],
      ['Hindi', 166, 12.9],
      ['Spanish', 82, 6.4],
    ]),
  );

  const response = new GetAssistantDashboardResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setData(dashboard);
  return Buffer.from(response.serializeBinary());
}

function buildConversation() {
  const conversation = new AssistantConversation();
  conversation.setId(ASSISTANT_CONVERSATION_ID);
  conversation.setAssistantid(ASSISTANT_ID);
  conversation.setAssistantprovidermodelid(ASSISTANT_VERSION_ID);
  conversation.setName('Password reset request');
  conversation.setIdentifier('customer@example.test');
  conversation.setSource('web-plugin');
  conversation.setStatus('completed');
  conversation.setDirection('inbound');
  conversation.setProjectid('302');
  conversation.setOrganizationid('202');
  conversation.setCreateddate(timestamp('2026-10-17T08:24:00Z'));
  conversation.setUpdateddate(timestamp('2026-10-17T08:26:34Z'));
  conversation.setMetricsList([
    metric('status', 'COMPLETE'),
    metric('conversation.duration_ms', '154000'),
    metric('agent_total_token', '1284'),
  ]);
  conversation.setMetadataList([
    metadata('language', 'en-IN'),
    metadata('analysis.summary', 'The user completed a password reset.'),
  ]);
  return conversation;
}

function buildConversationListResponse(conversations = [buildConversation()]) {
  const response = new GetAllAssistantConversationResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList(conversations);
  response.setPaginated(paginated(conversations.length));
  return Buffer.from(response.serializeBinary());
}

function buildConversationDetailResponse() {
  const response = new GetAssistantConversationResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setData(buildConversation());
  return Buffer.from(response.serializeBinary());
}

function buildMessageListResponse() {
  const userMessage = new AssistantConversationMessage();
  userMessage.setId('2301664620831572201');
  userMessage.setMessageid('2301664620831572201');
  userMessage.setAssistantid(ASSISTANT_ID);
  userMessage.setAssistantconversationid(ASSISTANT_CONVERSATION_ID);
  userMessage.setRole('user');
  userMessage.setBody('I need help resetting my password.');
  userMessage.setStatus('complete');
  userMessage.setCreateddate(timestamp('2026-10-17T08:24:10Z'));

  const assistantMessage = new AssistantConversationMessage();
  assistantMessage.setId('2301664620831572202');
  assistantMessage.setMessageid('2301664620831572202');
  assistantMessage.setAssistantid(ASSISTANT_ID);
  assistantMessage.setAssistantconversationid(ASSISTANT_CONVERSATION_ID);
  assistantMessage.setRole('assistant');
  assistantMessage.setBody(
    'I sent a secure password reset link to your registered email address.',
  );
  assistantMessage.setStatus('complete');
  assistantMessage.setCreateddate(timestamp('2026-10-17T08:24:18Z'));

  const response = new GetAllConversationMessageResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList([userMessage, assistantMessage]);
  response.setPaginated(paginated(2));
  return Buffer.from(response.serializeBinary());
}

function buildProviderListResponse() {
  const provider = buildAssistant().getAssistantprovidermodel();
  const item = new GetAllAssistantProviderResponse.AssistantProvider();
  item.setAssistantprovidermodel(provider);

  const response = new GetAllAssistantProviderResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList([item]);
  response.setPaginated(paginated(1));
  return Buffer.from(response.serializeBinary());
}

const EMPTY_SUCCESS_RESPONSE = Buffer.from([0x10, 0x01]);
const EMPTY_ASSISTANT_LIST_RESPONSE = buildAssistantListResponse([]);
const POPULATED_ASSISTANT_LIST_RESPONSE = buildAssistantListResponse();

async function installAssistantClientMock(page) {
  await page.route(ASSISTANT_SERVICE_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, EMPTY_SUCCESS_RESPONSE);
  });
  await page.route(ASSISTANT_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, POPULATED_ASSISTANT_LIST_RESPONSE);
  });
  await page.route(ASSISTANT_DETAIL_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildAssistantDetailResponse());
  });
  await page.route(ASSISTANT_DASHBOARD_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildAssistantDashboardResponse());
  });
  await page.route(ASSISTANT_PROVIDER_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildProviderListResponse());
  });
  await page.route(ASSISTANT_CONVERSATION_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildConversationListResponse());
  });
  await page.route(ASSISTANT_CONVERSATION_DETAIL_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildConversationDetailResponse());
  });
  await page.route(ASSISTANT_MESSAGE_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, buildMessageListResponse());
  });
}

module.exports = {
  ASSISTANT_CONVERSATION_DETAIL_PATH,
  ASSISTANT_CONVERSATION_ID,
  ASSISTANT_CONVERSATION_LIST_PATH,
  ASSISTANT_DASHBOARD_PATH,
  ASSISTANT_DETAIL_PATH,
  ASSISTANT_ID,
  ASSISTANT_LIST_PATH,
  ASSISTANT_MESSAGE_LIST_PATH,
  ASSISTANT_PROVIDER_LIST_PATH,
  EMPTY_ASSISTANT_LIST_RESPONSE,
  POPULATED_ASSISTANT_LIST_RESPONSE,
  buildAssistantDashboardResponse,
  buildAssistantDetailResponse,
  buildDebuggerAssistantDetailResponse,
  buildConversationDetailResponse,
  buildConversationListResponse,
  buildMessageListResponse,
  fulfillGrpcWeb,
  installAssistantClientMock,
  mockDebuggerAssistant,
};

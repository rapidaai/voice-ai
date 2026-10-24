const { Timestamp } = require('google-protobuf/google/protobuf/timestamp_pb');
const { fulfillGrpcWeb } = require('../grpc-web');

const previousWindow = global.window;
const previousSelf = global.self;
global.window = global;
global.self = global;
const {
  DashboardActivityBucket,
  DashboardAnalyticsSection,
  DashboardAssistant,
  DashboardAssistantHealth,
  DashboardComponentUsage,
  DashboardCountComparison,
  DashboardFailedComparison,
  DashboardFailureCode,
  DashboardHealth,
  DashboardLatencyComparison,
  DashboardProjectState,
  DashboardRateComparison,
  DashboardRecentFailure,
  DashboardSectionState,
  DashboardSummary,
  DashboardUsageComponent,
  DashboardUsageSection,
  GetProjectDashboardResponse,
  GetProjectDashboardRequest,
  ProjectDashboard,
  ProjectDashboardAnalytics,
} = require('@rapidaai/react');
if (previousWindow === undefined) delete global.window;
else global.window = previousWindow;
if (previousSelf === undefined) delete global.self;
else global.self = previousSelf;

const DASHBOARD_PATH = '**/dashboard_api.DashboardService/GetProjectDashboard';
const dashboardRequests = new WeakMap();

function timestamp(value) {
  const result = new Timestamp();
  result.fromDate(new Date(value));
  return result;
}

function buildSummary() {
  const sessions = new DashboardCountComparison();
  sessions.setValue(1284);
  sessions.setPreviousvalue(1142);
  sessions.setChangepercent(12.4);

  const successRate = new DashboardRateComparison();
  successRate.setValuepercent(96.8);
  successRate.setPreviousvaluepercent(95.6);
  successRate.setChangepoints(1.2);

  const latency = new DashboardLatencyComparison();
  latency.setValuems(742);
  latency.setPreviousvaluems(790);
  latency.setChangems(-48);

  const failedSessions = new DashboardFailedComparison();
  failedSessions.setValue(41);
  failedSessions.setPreviousvalue(49);
  failedSessions.setSharepercent(3.2);

  const summary = new DashboardSummary();
  summary.setSessions(sessions);
  summary.setSuccessrate(successRate);
  summary.setResponselatency(latency);
  summary.setFailedsessions(failedSessions);
  return summary;
}

function buildActivity() {
  return [
    ['2026-10-10T00:00:00Z', 52, 3],
    ['2026-10-11T00:00:00Z', 41, 2],
    ['2026-10-12T00:00:00Z', 72, 3],
    ['2026-10-13T00:00:00Z', 146, 7],
    ['2026-10-14T00:00:00Z', 132, 4],
    ['2026-10-15T00:00:00Z', 178, 8],
    ['2026-10-16T00:00:00Z', 156, 6],
    ['2026-10-17T00:00:00Z', 81, 3],
  ].map(([date, sessions, failures]) => {
    const bucket = new DashboardActivityBucket();
    bucket.setFromdate(timestamp(date));
    bucket.setSessions(sessions);
    bucket.setFailedsessions(failures);
    return bucket;
  });
}

function buildAssistants() {
  return [
    [
      '2301664620831571968',
      'Support bot',
      528,
      97.2,
      612,
      '2026-10-17T08:28:00Z',
      DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_HEALTHY,
    ],
    [
      '2301664620831571969',
      'Sales assistant',
      431,
      92.1,
      1100,
      '2026-10-17T08:24:00Z',
      DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_WATCH,
    ],
    [
      '2301664620831571970',
      'Appointment booking',
      325,
      84.8,
      1604,
      '2026-10-17T08:19:00Z',
      DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_NEEDS_ATTENTION,
    ],
  ].map(([id, name, sessions, successRate, latency, lastActivity, health]) => {
    const assistant = new DashboardAssistant();
    assistant.setAssistantid(id);
    assistant.setName(name);
    assistant.setSessions(sessions);
    assistant.setSuccessratepercent(successRate);
    assistant.setResponselatencyms(latency);
    assistant.setLastactivityat(timestamp(lastActivity));
    assistant.setHealth(health);
    return assistant;
  });
}

function buildFailures() {
  return [
    [
      '2301664620831572101',
      '2301664620831571968',
      'Support bot',
      'Phone',
      94000,
      '2026-10-17T08:25:00Z',
      DashboardFailureCode.DASHBOARD_FAILURE_CODE_PROVIDER,
    ],
    [
      '2301664620831572102',
      '2301664620831571969',
      'Sales assistant',
      'Web',
      38000,
      '2026-10-17T08:12:00Z',
      DashboardFailureCode.DASHBOARD_FAILURE_CODE_NETWORK,
    ],
    [
      '2301664620831572103',
      '2301664620831571970',
      'Appointment booking',
      'Phone',
      67000,
      '2026-10-17T07:54:00Z',
      DashboardFailureCode.DASHBOARD_FAILURE_CODE_NO_ANSWER,
    ],
  ].map(
    ([
      id,
      assistantId,
      assistantName,
      channel,
      duration,
      occurredAt,
      failureCode,
    ]) => {
      const failure = new DashboardRecentFailure();
      failure.setConversationid(id);
      failure.setAssistantid(assistantId);
      failure.setAssistantname(assistantName);
      failure.setChannel(channel);
      failure.setDurationms(duration);
      failure.setOccurredat(timestamp(occurredAt));
      failure.setFailurecode(failureCode);
      return failure;
    },
  );
}

function buildUsage() {
  return [
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_STT, 172800000],
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_TTS, 136800000],
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_LLM, 108000000],
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_VAD, 75600000],
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_EOS, 50400000],
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_DENOISING, 28800000],
  ].map(([component, duration]) => {
    const usage = new DashboardComponentUsage();
    usage.setComponent(component);
    usage.setDurationms(duration);
    return usage;
  });
}

function buildDashboardResponse() {
  const projectState = new DashboardProjectState();
  projectState.setAssistantcount(3);
  projectState.setHasassistants(true);
  projectState.setHasactivity(true);

  const health = new DashboardHealth();
  health.setFailedsessions(41);
  health.setSlowsessions(27);
  health.setSlowsessionthresholdms(1000);
  health.setHealthyassistants(1);
  health.setWatchassistants(1);
  health.setNeedsattentionassistants(1);

  const analytics = new ProjectDashboardAnalytics();
  analytics.setProjectstate(projectState);
  analytics.setSummary(buildSummary());
  analytics.setActivityList(buildActivity());
  analytics.setHealth(health);
  analytics.setAssistantsList(buildAssistants());
  analytics.setRecentfailuresList(buildFailures());

  const analyticsSection = new DashboardAnalyticsSection();
  analyticsSection.setState(
    DashboardSectionState.DASHBOARD_SECTION_STATE_READY,
  );
  analyticsSection.setData(analytics);

  const usageSection = new DashboardUsageSection();
  usageSection.setState(DashboardSectionState.DASHBOARD_SECTION_STATE_READY);
  usageSection.setComponentsList(buildUsage());

  const dashboard = new ProjectDashboard();
  dashboard.setGeneratedat(timestamp('2026-10-17T08:30:00Z'));
  dashboard.setAnalytics(analyticsSection);
  dashboard.setUsage(usageSection);

  const response = new GetProjectDashboardResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setData(dashboard);
  return Buffer.from(response.serializeBinary());
}

function buildEmptyDashboardResponse() {
  const analyticsSection = new DashboardAnalyticsSection();
  analyticsSection.setState(
    DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY,
  );

  const usageSection = new DashboardUsageSection();
  usageSection.setState(DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY);

  const dashboard = new ProjectDashboard();
  dashboard.setGeneratedat(timestamp('2026-10-17T08:30:00Z'));
  dashboard.setAnalytics(analyticsSection);
  dashboard.setUsage(usageSection);

  const response = new GetProjectDashboardResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setData(dashboard);
  return Buffer.from(response.serializeBinary());
}

async function installEmptyDashboardClientMock(page) {
  const response = buildEmptyDashboardResponse();
  const handler = route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, response);
  };
  await page.route(DASHBOARD_PATH, handler);
  return () => page.unroute(DASHBOARD_PATH, handler);
}

async function installDashboardClientMock(page) {
  const response = buildDashboardResponse();
  dashboardRequests.set(page, []);
  await page.route(DASHBOARD_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    const request = route.request();
    const body = request.postDataBuffer();
    if (!body || body.length < 5 || body[0] !== 0) {
      throw new Error('dashboard request did not contain a gRPC data frame');
    }
    const messageLength = body.readUInt32BE(1);
    const dashboardRequest = GetProjectDashboardRequest.deserializeBinary(
      body.subarray(5, 5 + messageLength),
    );
    const headers = request.headers();
    if (
      headers.authorization !== 'token-e2e' ||
      headers['x-auth-id'] !== '101' ||
      headers['x-project-id'] !== '302'
    ) {
      throw new Error('dashboard request did not include project metadata');
    }
    if (
      !dashboardRequest.hasFromdate() ||
      !dashboardRequest.hasTodate() ||
      dashboardRequest.getAssistantlimit() !== 3 ||
      dashboardRequest.getRecentfailurelimit() !== 5
    ) {
      throw new Error(
        'dashboard request did not include the selected range and limits',
      );
    }
    const fromDate = dashboardRequest.getFromdate().toDate();
    const toDate = dashboardRequest.getTodate().toDate();
    const durationMs = toDate.getTime() - fromDate.getTime();
    const supportedDurations = [24, 7 * 24, 30 * 24].map(
      hours => hours * 60 * 60 * 1000,
    );
    if (!supportedDurations.includes(durationMs)) {
      throw new Error(
        `dashboard request used unsupported duration ${durationMs}`,
      );
    }
    dashboardRequests.get(page).push({ fromDate, toDate, durationMs });
    return fulfillGrpcWeb(route, response);
  });
}

function getDashboardRequests(page) {
  return dashboardRequests.get(page) || [];
}

module.exports = {
  DASHBOARD_PATH,
  getDashboardRequests,
  installDashboardClientMock,
  installEmptyDashboardClientMock,
};

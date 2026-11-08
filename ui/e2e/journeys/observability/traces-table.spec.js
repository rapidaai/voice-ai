const { Timestamp } = require('google-protobuf/google/protobuf/timestamp_pb');
const {
  authenticatePage,
  checkA11y,
  expect,
  setColorMode,
  test,
} = require('../../fixtures');
const { fulfillGrpcWeb } = require('../../mocks/grpc-web');
const {
  GetAllTelemetryResponse,
  ObservabilityMetricRecord,
  ObservabilityRecord,
  ObservabilityRecordKind,
  Paginated,
} = require('@rapidaai/react');

const TELEMETRY_LIST_PATH =
  '**/observability_api.ObservabilityService/GetAllTelemetry';
const tracesPath = '/logs/traces';
const viewport = { width: 1024, height: 900 };
const journey = { id: 'logs.traces' };

function buildTelemetryListResponse() {
  const occurredAt = new Timestamp();
  occurredAt.fromDate(new Date('2026-11-08T10:00:00.000Z'));

  const metric = new ObservabilityMetricRecord();
  metric.setId('metric-1');
  metric.setKind(ObservabilityRecordKind.OBSERVABILITY_RECORD_KIND_METRIC);
  metric.setName('stt.latency_ms');
  metric.setValue('120');
  metric.setDescription('milliseconds');
  metric.setProjectid('302');
  metric.setOrganizationid('202');
  metric.setScope('message');
  metric.setOccurredat(occurredAt);
  metric.getAttributesMap().set('component', 'stt');
  metric.getContextMap().set('traceId', 'trace-1');
  metric.getScopeattributesMap().set('assistantId', 'assistant-1');
  metric
    .getScopeattributesMap()
    .set('assistantConversationId', 'conversation-1');
  metric.getScopeattributesMap().set('messageId', 'message-1');
  metric.getScopeattributesMap().set('messageRole', 'user');

  const record = new ObservabilityRecord();
  record.setMetric(metric);

  const paginated = new Paginated();
  paginated.setCurrentpage(1);
  paginated.setTotalitem(1);

  const response = new GetAllTelemetryResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList([record]);
  response.setPaginated(paginated);
  return Buffer.from(response.serializeBinary());
}

for (const colorMode of ['light', 'dark']) {
  test(`populated traces keep pagination at the bottom in ${colorMode} mode`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await page.route(TELEMETRY_LIST_PATH, route =>
      fulfillGrpcWeb(route, buildTelemetryListResponse()),
    );

    await page.goto(tracesPath);

    await expect(page.getByText('metric-1', { exact: true })).toBeVisible();
    const pagination = page.locator('.cds--pagination');
    await expect(pagination).toHaveClass(/sticky/);
    await expect(pagination).toHaveClass(/bottom-0/);
    const paginationBox = await pagination.boundingBox();
    expect(paginationBox).not.toBeNull();
    expect(
      Math.abs(viewport.height - (paginationBox.y + paginationBox.height)),
    ).toBeLessThanOrEqual(1);
    await checkA11y(page, journey);
  });
}

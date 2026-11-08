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
  AssistantConfiguration,
  GetAllAssistantConfigurationResponse,
  Metadata,
  Paginated,
} = require('@rapidaai/react');

const CONFIGURATION_LIST_PATH =
  '**/assistant_api.AssistantService/GetAllAssistantConfiguration';
const assistantId = '2301664620831571968';
const viewport = { width: 1280, height: 900 };

const pages = [
  {
    id: 'deployment.assistantAnalysis',
    slug: 'analysis',
    type: 'analysis',
    path: `/deployment/assistant/${assistantId}/configure-analysis`,
    rowText: 'Sentiment review',
    search: 'Search analysis...',
  },
  {
    id: 'deployment.assistantStorage',
    slug: 'storage',
    type: 'storage',
    path: `/deployment/assistant/${assistantId}/configure-storage`,
    rowText: 'recordings',
    search: 'Search storage...',
  },
  {
    id: 'deployment.assistantTelemetry',
    slug: 'telemetry',
    type: 'telemetry',
    path: `/deployment/assistant/${assistantId}/configure-telemetry`,
    rowText: 'https://telemetry.example.test',
    search: 'Search telemetry...',
  },
  {
    id: 'deployment.assistantWebhooks',
    slug: 'webhooks',
    type: 'webhook',
    path: `/deployment/assistant/${assistantId}/configure-webhook`,
    rowText: 'https://hooks.example.test/events',
    search: 'Search webhooks...',
  },
  {
    id: 'deployment.assistantAuthentication',
    slug: 'authentication',
    type: 'authentication',
    path: `/deployment/assistant/${assistantId}/configure-authentication`,
    rowText: 'https://auth.example.test/session',
  },
];

function timestamp(value) {
  const result = new Timestamp();
  result.fromDate(new Date(value));
  return result;
}

function metadata(key, value) {
  const option = new Metadata();
  option.setKey(key);
  option.setValue(value);
  return option;
}

function buildConfigurationListResponse(type) {
  const configuration = new AssistantConfiguration();
  configuration.setId('2301664620831574001');
  configuration.setAssistantid(assistantId);
  configuration.setConfigurationtype(type);
  configuration.setProvider(
    type === 'storage'
      ? 'aws-cloud'
      : type === 'authentication'
        ? 'http'
        : 'custom',
  );
  configuration.setEnabled(true);
  configuration.setStatus('ACTIVE');
  configuration.setCreateddate(timestamp('2026-10-17T08:28:00Z'));
  configuration.setOptionsList([
    metadata('name', 'Sentiment review'),
    metadata('endpoint_id', 'endpoint-1'),
    metadata('endpoint_version', 'v1'),
    metadata('execution_priority', '1'),
    metadata('s3_bucket_name', 'recordings'),
    metadata(
      'endpoint',
      type === 'authentication'
        ? 'https://auth.example.test/session'
        : 'https://telemetry.example.test',
    ),
    metadata('http_method', 'POST'),
    metadata(
      'http_url',
      type === 'authentication'
        ? 'https://auth.example.test/session'
        : 'https://hooks.example.test/events',
    ),
    metadata('assistant_events', '["conversation.begin"]'),
    metadata('max_retry_count', '3'),
    metadata('timeout_seconds', '10'),
  ]);

  const paginated = new Paginated();
  paginated.setCurrentpage(1);
  paginated.setTotalitem(1);

  const response = new GetAllAssistantConfigurationResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList([configuration]);
  response.setPaginated(paginated);
  return Buffer.from(response.serializeBinary());
}

for (const colorMode of ['light', 'dark']) {
  for (const configurationPage of pages) {
    test(`${configurationPage.slug} table in ${colorMode} mode`, async ({
      page,
    }) => {
      await page.setViewportSize(viewport);
      await setColorMode(page, colorMode);
      await authenticatePage(page);
      await page.route(CONFIGURATION_LIST_PATH, route =>
        fulfillGrpcWeb(
          route,
          buildConfigurationListResponse(configurationPage.type),
        ),
      );

      await page.goto(configurationPage.path);

      await expect(
        page.getByText(configurationPage.rowText, { exact: true }),
      ).toBeVisible();
      await checkA11y(page, { id: configurationPage.id });

      if (configurationPage.search) {
        await expect(
          page.getByPlaceholder(configurationPage.search),
        ).toBeVisible();
        await expect(page.locator('.cds--pagination')).toHaveClass(/shrink-0/);
      }

      await expect(page).toHaveScreenshot(
        [
          'deployment',
          'configuration-tables',
          configurationPage.slug,
          'success',
          `xl-${colorMode}.png`,
        ],
        { fullPage: true, maxDiffPixelRatio: 0.01 },
      );

      if (!configurationPage.search) return;

      await page.locator('.cds--table-column-radio label').click();
      const batchActions = page.locator('.cds--batch-actions');
      await expect(batchActions).toContainText('1 item selected');
      await expect(
        batchActions.getByRole('button', { name: 'Edit' }),
      ).toBeVisible();
      await expect(
        batchActions.getByRole('button', { name: 'Disable' }),
      ).toBeVisible();
      await expect(
        batchActions.getByRole('button', { name: 'Delete' }),
      ).toBeVisible();

      await expect(page).toHaveScreenshot(
        [
          'deployment',
          'configuration-tables',
          configurationPage.slug,
          'batch-actions',
          `xl-${colorMode}.png`,
        ],
        { fullPage: true, maxDiffPixelRatio: 0.01 },
      );
    });
  }
}

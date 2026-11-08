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
  AssistantTool,
  GetAllAssistantToolResponse,
  Paginated,
} = require('@rapidaai/react');

const TOOL_LIST_PATH = '**/assistant_api.AssistantService/GetAllAssistantTool';
const toolsPath = '/deployment/assistant/2301664620831571968/configure-tool';
const viewport = { width: 1024, height: 900 };
const journey = { id: 'deployment.assistantTools' };

function timestamp(value) {
  const result = new Timestamp();
  result.fromDate(new Date(value));
  return result;
}

function buildToolListResponse() {
  const tool = new AssistantTool();
  tool.setId('2301664620831574001');
  tool.setAssistantid('2301664620831571968');
  tool.setName('Order lookup');
  tool.setDescription('Looks up an order by reference.');
  tool.setExecutionmethod('mcp');
  tool.setStatus('ACTIVE');
  tool.setCreateddate(timestamp('2026-10-17T08:28:00Z'));

  const paginated = new Paginated();
  paginated.setCurrentpage(1);
  paginated.setTotalitem(1);

  const response = new GetAllAssistantToolResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList([tool]);
  response.setPaginated(paginated);
  return Buffer.from(response.serializeBinary());
}

for (const colorMode of ['light', 'dark']) {
  test(`assistant tools loading in ${colorMode} mode`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await page.route(TOOL_LIST_PATH, () => {});

    await page.goto(toolsPath);

    await expect(
      page.getByRole('heading', { name: 'Tool & MCP' }),
    ).toBeVisible();
    await expect(
      page.locator('.cds--data-table-container.cds--skeleton'),
    ).toBeVisible();

    await expect(page).toHaveScreenshot(
      ['assistant', 'tools', 'loading', `lg-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });

  test(`assistant tools table and batch actions in ${colorMode} mode`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await page.route(TOOL_LIST_PATH, route =>
      fulfillGrpcWeb(route, buildToolListResponse()),
    );

    await page.goto(toolsPath);

    await expect(page.getByText('Order lookup', { exact: true })).toBeVisible();
    await expect(page.getByPlaceholder('Search tools...')).toBeVisible();
    await checkA11y(page, journey);

    await expect(page).toHaveScreenshot(
      ['assistant', 'tools', 'success', `lg-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );

    await page.locator('.cds--table-column-radio label').click();
    const batchActions = page.locator('.cds--batch-actions');
    await expect(batchActions).toContainText('1 item selected');
    await expect(
      batchActions.getByRole('button', { name: 'Edit' }),
    ).toBeVisible();
    await expect(
      batchActions.getByRole('button', { name: 'Delete' }),
    ).toBeVisible();

    await expect(page).toHaveScreenshot(
      ['assistant', 'tools', 'batch-actions', `lg-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });
}

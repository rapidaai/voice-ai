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
  GetAllProjectResponse,
  Paginated,
  Project,
  User,
} = require('@rapidaai/react');

const PROJECT_LIST_PATH = '**/web_api.ProjectService/GetAllProject';
const projectsPath = '/organization/projects';
const viewport = { width: 1024, height: 900 };
const journey = { id: 'organization.projects' };

function timestamp(value) {
  const result = new Timestamp();
  result.fromDate(new Date(value));
  return result;
}

function buildProjectListResponse(itemCount = 1) {
  const member = new User();
  member.setId('101');
  member.setName('E2E User');
  member.setEmail('e2e@example.test');

  const projects = Array.from({ length: itemCount }, (_, index) => {
    const project = new Project();
    project.setId(String(302 + index));
    project.setName(index === 0 ? 'Customer support' : `Project ${index + 1}`);
    project.setDescription('Customer support automation workspace.');
    project.setMembersList([member]);
    project.setStatus('ACTIVE');
    project.setCreateddate(timestamp('2026-10-17T08:28:00Z'));
    return project;
  });

  const paginated = new Paginated();
  paginated.setCurrentpage(1);
  paginated.setTotalitem(itemCount);

  const response = new GetAllProjectResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList(projects);
  response.setPaginated(paginated);
  return Buffer.from(response.serializeBinary());
}

test('organization projects keep table controls sticky through long results', async ({
  page,
}) => {
  await page.setViewportSize(viewport);
  await authenticatePage(page);
  await page.route(PROJECT_LIST_PATH, route =>
    fulfillGrpcWeb(route, buildProjectListResponse(25)),
  );

  await page.goto(projectsPath);

  await expect(page.getByText('Project 25', { exact: true })).toBeVisible();
  const pageScroll = page.locator('[data-page-scroll]');
  await pageScroll.evaluate(element => {
    element.scrollTop = element.scrollHeight;
  });
  const scrollPosition = await pageScroll.evaluate(element => ({
    clientHeight: element.clientHeight,
    scrollHeight: element.scrollHeight,
    scrollTop: element.scrollTop,
  }));
  expect(scrollPosition.scrollHeight).toBeGreaterThan(
    scrollPosition.clientHeight,
  );
  expect(scrollPosition.scrollTop).toBeGreaterThan(0);

  const toolbar = page.locator('[data-page-header-sticky-toolbar]');
  await expect(toolbar).toBeVisible();
  await expect
    .poll(async () => {
      const pageScrollBox = await pageScroll.boundingBox();
      const toolbarBox = await toolbar.boundingBox();
      if (!pageScrollBox || !toolbarBox) return Number.MAX_SAFE_INTEGER;
      return Math.abs(toolbarBox.y - (pageScrollBox.y + 40));
    })
    .toBeLessThanOrEqual(1);
});

for (const colorMode of ['light', 'dark']) {
  test(`organization projects loading in ${colorMode} mode`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await page.route(PROJECT_LIST_PATH, () => {});

    await page.goto(projectsPath);

    await expect(
      page.getByRole('heading', { level: 1, name: 'Projects' }),
    ).toBeVisible();
    await expect(
      page.locator('.cds--data-table-container.cds--skeleton'),
    ).toBeVisible();

    await expect(page).toHaveScreenshot(
      ['organization', 'projects', 'loading', `lg-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });

  test(`organization projects table and batch actions in ${colorMode} mode`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await page.route(PROJECT_LIST_PATH, route =>
      fulfillGrpcWeb(route, buildProjectListResponse()),
    );

    await page.goto(projectsPath);

    await expect(
      page.getByText('Customer support', { exact: true }),
    ).toBeVisible();
    await expect(page.getByPlaceholder('Search projects...')).toBeVisible();
    const pagination = page.locator('.cds--pagination');
    await expect(pagination).toHaveClass(/sticky/);
    await expect(pagination).toHaveClass(/bottom-0/);
    const paginationBox = await pagination.boundingBox();
    expect(paginationBox).not.toBeNull();
    expect(
      Math.abs(viewport.height - (paginationBox.y + paginationBox.height)),
    ).toBeLessThanOrEqual(1);
    await checkA11y(page, journey);

    await expect(page).toHaveScreenshot(
      ['organization', 'projects', 'success', `lg-${colorMode}.png`],
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
      ['organization', 'projects', 'batch-actions', `lg-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });
}

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

function buildProjectListResponse() {
  const member = new User();
  member.setId('101');
  member.setName('E2E User');
  member.setEmail('e2e@example.test');

  const project = new Project();
  project.setId('302');
  project.setName('Customer support');
  project.setDescription('Customer support automation workspace.');
  project.setMembersList([member]);
  project.setStatus('ACTIVE');
  project.setCreateddate(timestamp('2026-10-17T08:28:00Z'));

  const paginated = new Paginated();
  paginated.setCurrentpage(1);
  paginated.setTotalitem(1);

  const response = new GetAllProjectResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList([project]);
  response.setPaginated(paginated);
  return Buffer.from(response.serializeBinary());
}

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
      page.getByRole('main').locator('.text-base', { hasText: 'Projects' }),
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
    await expect(page.locator('.cds--pagination')).toHaveClass(/shrink-0/);
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

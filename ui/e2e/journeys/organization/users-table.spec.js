const { Timestamp } = require('google-protobuf/google/protobuf/timestamp_pb');
const {
  authenticatePage,
  checkA11y,
  expect,
  setColorMode,
  test,
} = require('../../fixtures');
const { fulfillGrpcWeb } = require('../../mocks/grpc-web');
const { GetAllUserResponse, Paginated, User } = require('@rapidaai/react');

const USER_LIST_PATH = '**/web_api.AuthenticationService/GetAllUser';
const usersPath = '/organization/users';
const viewport = { width: 1024, height: 900 };
const journey = { id: 'organization.users' };

function timestamp(value) {
  const result = new Timestamp();
  result.fromDate(new Date(value));
  return result;
}

function buildUserListResponse() {
  const user = new User();
  user.setId('101');
  user.setName('Ada Lovelace');
  user.setEmail('ada@example.test');
  user.setRole('member');
  user.setStatus('ACTIVE');
  user.setCreateddate(timestamp('2026-10-17T08:28:00Z'));

  const paginated = new Paginated();
  paginated.setCurrentpage(1);
  paginated.setTotalitem(1);

  const response = new GetAllUserResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setDataList([user]);
  response.setPaginated(paginated);
  return Buffer.from(response.serializeBinary());
}

for (const colorMode of ['light', 'dark']) {
  test(`organization users loading in ${colorMode} mode`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await page.route(USER_LIST_PATH, () => {});

    await page.goto(usersPath);

    await expect(
      page.getByRole('heading', { level: 1, name: 'Users' }),
    ).toBeVisible();
    await expect(
      page.locator('.cds--data-table-container.cds--skeleton'),
    ).toBeVisible();

    await expect(page).toHaveScreenshot(
      ['organization', 'users', 'loading', `lg-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });

  test(`organization users table and batch actions in ${colorMode} mode`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await page.route(USER_LIST_PATH, route =>
      fulfillGrpcWeb(route, buildUserListResponse()),
    );

    await page.goto(usersPath);

    await expect(page.getByText('Ada Lovelace', { exact: true })).toBeVisible();
    await expect(page.getByPlaceholder('Search users...')).toBeVisible();
    await expect(page.locator('.cds--pagination')).toHaveClass(/shrink-0/);
    await checkA11y(page, journey);

    await expect(page).toHaveScreenshot(
      ['organization', 'users', 'success', `lg-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );

    await page.locator('.cds--table-column-radio label').click();
    const batchActions = page.locator('.cds--batch-actions');
    await expect(batchActions).toContainText('1 item selected');
    await expect(
      batchActions.getByRole('button', { name: 'Invite to project' }),
    ).toBeVisible();
    await expect(
      batchActions.getByRole('button', { name: 'Make admin' }),
    ).toBeVisible();
    await expect(
      batchActions.getByRole('button', { name: 'Delete' }),
    ).toBeVisible();

    await expect(page).toHaveScreenshot(
      ['organization', 'users', 'batch-actions', `lg-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });
}

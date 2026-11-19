const {
  expect,
  gotoJourney,
  setColorMode,
  test,
  WORKSPACE_LIST_PATHS,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');
const {
  fulfillGrpcWeb,
  protobufBytes,
  protobufString,
} = require('../../mocks/grpc-web');

const journey = id =>
  routeJourneys.find(routeJourney => routeJourney.id === `organization.${id}`);

const userListPath = WORKSPACE_LIST_PATHS.find(path =>
  path.endsWith('/web_api.AuthenticationService/GetAllUser'),
);
const projectListPath = WORKSPACE_LIST_PATHS.find(path =>
  path.endsWith('/web_api.ProjectService/GetAllProject'),
);

for (const [colorMode, expectedFill] of [
  ['light', 'rgb(22, 22, 22)'],
  ['dark', 'rgb(244, 244, 244)'],
]) {
  test(`invite modal close icon uses the Carbon ${colorMode} theme color`, async ({
    page,
  }) => {
    await setColorMode(page, colorMode);
    await gotoJourney(page, journey('users'));
    await page
      .getByRole('button', { name: 'Invite user', exact: true })
      .first()
      .click();

    await expect(
      page.locator('.cds--modal.is-visible .cds--modal-close__icon'),
    ).toHaveCSS('fill', expectedFill);
  });
}

test('organization listings show actionable empty states', async ({ page }) => {
  await gotoJourney(page, journey('users'));
  await expect(
    page.getByRole('heading', { name: 'No organization users' }),
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'Invite user', exact: true }),
  ).toHaveCount(2);

  await gotoJourney(page, journey('projects'));
  await expect(
    page.getByRole('heading', { name: 'No projects' }),
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'Create new project', exact: true }),
  ).toHaveCount(2);
});

test('project listing shows an application response error', async ({
  page,
}) => {
  const error = protobufString(3, 'Unable to load projects.');

  await gotoJourney(page, journey('projects'));

  await page.route(projectListPath, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, protobufBytes(4, error));
  });

  await page.getByRole('button', { name: 'Refresh' }).click();

  await expect(page.getByText('Unable to load projects.')).toBeVisible();
});

test('user listing shows a transport request error', async ({ page }) => {
  await gotoJourney(page, journey('users'));

  await page.route(userListPath, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return route.abort('failed');
  });

  await page.getByRole('button', { name: 'Refresh' }).click();

  await expect(
    page.getByText(
      'Unable to get all the users for your organization, please try again in sometime.',
    ),
  ).toBeVisible();
});

test('organization user listing opens the invite subjourney and validates input', async ({
  page,
}) => {
  await gotoJourney(page, journey('users'));

  await page
    .getByRole('button', { name: 'Invite user', exact: true })
    .first()
    .click();
  await expect(
    page.getByRole('heading', { name: 'Invite User to Organization' }),
  ).toBeVisible();

  await page.getByRole('button', { name: 'Add project access' }).click();
  const projectAccessTable = page.getByRole('table');
  await expect(projectAccessTable).toHaveCSS('table-layout', 'fixed');
  const projectAccessScroller = projectAccessTable.locator('xpath=..');
  expect(
    await projectAccessScroller.evaluate(
      element => element.scrollWidth <= element.clientWidth,
    ),
  ).toBe(true);
  const columnWidthsBeforeSelection = await projectAccessTable
    .locator('tbody tr')
    .first()
    .locator('td')
    .evaluateAll(cells =>
      cells.map(cell => cell.getBoundingClientRect().width),
    );
  const projectRoleSelector = projectAccessTable.locator(
    '.cds--multi-select .cds--list-box__field',
  );
  await expect(projectRoleSelector).toBeEnabled();
  await projectRoleSelector.click();

  const firstProjectRole = page.getByRole('option', {
    name: 'Support manager',
    exact: true,
  });
  await expect(firstProjectRole).toBeVisible();
  await expect(page.locator('.cds--list-box__menu:visible')).toHaveCSS(
    'position',
    'fixed',
  );
  await expect(page.getByText('Custom', { exact: true })).not.toBeVisible();
  await expect(page.getByText('Built-in', { exact: true })).not.toBeVisible();
  await expect(
    page.getByRole('option', { name: 'Organization Owner', exact: true }),
  ).not.toBeVisible();
  await expect(
    page.getByRole('option', { name: 'Runtime Invoker', exact: true }),
  ).not.toBeVisible();
  await firstProjectRole.click();
  await page
    .getByRole('option', { name: 'Project viewer', exact: true })
    .click();
  await expect(page.getByText('1 role selected')).not.toBeVisible();
  await page.keyboard.press('Escape');
  const columnWidthsAfterSelection = await projectAccessTable
    .locator('tbody tr')
    .first()
    .locator('td')
    .evaluateAll(cells =>
      cells.map(cell => cell.getBoundingClientRect().width),
    );
  expect(columnWidthsAfterSelection).toEqual(columnWidthsBeforeSelection);
  await projectRoleSelector.focus();
  const projectRoleField = projectRoleSelector.locator('xpath=..');
  await expect(projectRoleField).toHaveCSS('outline-width', '2px');
  expect(
    await projectRoleField.evaluate(element => {
      const fieldWidth = element.getBoundingClientRect().width;
      const listBoxWidth = element.parentElement?.getBoundingClientRect().width;
      return (
        listBoxWidth !== undefined && Math.abs(fieldWidth - listBoxWidth) < 1
      );
    }),
  ).toBe(true);

  await page
    .getByRole('button', { name: 'Invite user', exact: true })
    .last()
    .click();
  await expect(
    page.getByText('Please provide a valid email to invite user.'),
  ).toBeVisible();
});

test('project listing opens the create project subjourney', async ({
  page,
}) => {
  await gotoJourney(page, journey('projects'));

  await page
    .getByRole('button', { name: 'Create new project', exact: true })
    .first()
    .click();
  await expect(
    page.getByRole('heading', { name: 'Create a project' }),
  ).toBeVisible();
});

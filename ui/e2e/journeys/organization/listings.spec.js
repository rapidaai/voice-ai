const {
  expect,
  gotoJourney,
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
    page.getByRole('heading', { name: 'Invite user to organization' }),
  ).toBeVisible();

  await page.getByRole('button', { name: 'Add project access' }).click();
  await page.getByRole('combobox', { name: 'Select role' }).click();

  const firstProjectRole = page.getByRole('option', {
    name: 'Super Admin',
    exact: true,
  });
  await expect(firstProjectRole).toBeVisible();
  expect(
    await firstProjectRole.evaluate(element => {
      const bounds = element.getBoundingClientRect();
      const hitTarget = document.elementFromPoint(
        bounds.left + bounds.width / 2,
        bounds.top + bounds.height / 2,
      );

      return hitTarget === element || element.contains(hitTarget);
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

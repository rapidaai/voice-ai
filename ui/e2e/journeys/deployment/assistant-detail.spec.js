const {
  ASSISTANT_DETAIL_PATH,
  authenticatePage,
  expect,
  gotoJourney,
  test,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');

const journey = id => routeJourneys.find(candidate => candidate.id === id);

test('populated assistant journeys expose overview, sessions, versions, and settings', async ({
  page,
}) => {
  await gotoJourney(page, journey('deployment.assistantOverview'));
  await expect(
    page.getByRole('heading', { name: 'Assistant activity' }),
  ).toBeVisible();

  await page.getByRole('link', { name: 'Sessions', exact: true }).click();
  await expect(page.getByText('customer@example.test')).toBeVisible();

  await page.getByRole('button', { name: 'Versions', exact: true }).click();
  await page.getByRole('link', { name: 'View all', exact: true }).click();
  await expect(page.getByText('vrsn_2301664620831571999')).toBeVisible();

  await page.getByRole('link', { name: 'General', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: 'General settings' }),
  ).toBeVisible();
});

test('assistant session opens its populated message detail', async ({
  page,
}) => {
  await gotoJourney(page, journey('deployment.assistantSessions'));

  await page.getByRole('link', { name: /2301664620831572101/ }).click();

  await expect(page).toHaveURL(/\/sessions\/2301664620831572101$/);
  await expect(
    page.getByText('I need help resetting my password.'),
  ).toBeVisible();
  await expect(page.getByText('2 messages')).toBeVisible();
});

test('dashboard assistant drilldowns reach assistant and failed-session detail', async ({
  page,
}) => {
  await gotoJourney(page, journey('dashboard.home'));

  await page
    .getByRole('table', { name: 'Top assistants' })
    .getByRole('link', { name: 'Support bot' })
    .click();
  await expect(
    page.getByRole('heading', { name: 'Assistant activity' }),
  ).toBeVisible();

  await gotoJourney(page, journey('dashboard.home'));
  await page
    .getByRole('table', { name: 'Recent failures' })
    .getByRole('link', { name: 'Support bot' })
    .click();
  await expect(
    page.getByText('I need help resetting my password.'),
  ).toBeVisible();
});

test('assistant transport failure renders the full unavailable state', async ({
  page,
}) => {
  await page.route(ASSISTANT_DETAIL_PATH, route => route.abort('failed'));
  await authenticatePage(page);
  await page.goto('/deployment/assistant/2301664620831571968/overview');

  await expect(
    page.getByRole('heading', { name: 'Assistant not available' }),
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'Back to assistants' }),
  ).toBeVisible();
  await expect(page.getByText('Assistant activity')).toHaveCount(0);
});

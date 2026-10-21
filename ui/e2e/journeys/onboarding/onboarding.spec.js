const {
  authenticatePage,
  expect,
  gotoJourney,
  mockAuthenticationOperation,
  mockWorkspaceCreation,
  test,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');

const onboardingJourney = id =>
  routeJourneys.find(journey => journey.id === `onboarding.${id}`);

async function submitSignUp(page) {
  await page.getByLabel('Full name', { exact: true }).fill('E2E User');
  await page.getByLabel('Email address').fill('user@example.test');
  await page.locator('#signup-password').fill('correct horse battery staple');
  await page.getByRole('button', { name: 'Create account' }).click();
}

test('new account completes onboarding before opening the dashboard', async ({
  page,
}) => {
  const workspace = { organization: false, project: false };
  await mockAuthenticationOperation(page, 'signUp', {
    success: true,
    authentication: workspace,
  });
  await mockAuthenticationOperation(page, 'authorize', () => ({
    success: true,
    authentication: workspace,
  }));
  await mockWorkspaceCreation(page, 'organization', () => {
    workspace.organization = true;
  });
  await mockWorkspaceCreation(page, 'project', () => {
    workspace.project = true;
  });

  await page.goto('/auth/signup');
  await submitSignUp(page);

  await expect(page).toHaveURL(/\/onboarding\/organization$/);
  await page.getByLabel('Operating model').selectOption('agency');
  await page.getByLabel('Industry').fill('Agency services');
  await page.getByRole('button', { name: 'Create organization' }).click();

  await expect(page).toHaveURL(/\/onboarding\/project$/);
  await page.getByRole('button', { name: 'Create project' }).click();

  await expect(page).toHaveURL(/\/dashboard$/);
  await expect(
    page.getByRole('heading', { name: 'Project overview' }),
  ).toBeVisible();
});

test('organization-only account resumes at project creation', async ({
  page,
}) => {
  await authenticatePage(page, { organization: true, project: false });

  await page.goto('/dashboard');

  await expect(page).toHaveURL(/\/onboarding\/project$/);
  await expect(
    page.getByRole('heading', { name: 'Create your first project' }),
  ).toBeVisible();
});

test('onboarding uses semantic progress without mobile overflow', async ({
  page,
}) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await gotoJourney(page, onboardingJourney('organization'));

  await expect(page.getByText('Step 1 of 2')).toBeVisible();
  await expect(
    page.getByRole('progressbar', { name: 'Onboarding progress' }),
  ).toHaveAttribute('aria-valuenow', '1');
  await expect(
    page.getByRole('button', { name: 'Create organization' }),
  ).toBeVisible();

  const hasHorizontalOverflow = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth,
  );
  expect(hasHorizontalOverflow).toBe(false);
});

test('project setup cannot return to completed organization creation', async ({
  page,
}) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await gotoJourney(page, onboardingJourney('project'));

  await expect(page.getByText('Step 2 of 2')).toBeVisible();
  await expect(
    page.getByRole('progressbar', { name: 'Onboarding progress' }),
  ).toHaveAttribute('aria-valuenow', '2');
  await expect(page.getByRole('button', { name: 'Back' })).toHaveCount(0);

  const projectName = await page.getByLabel('Project name').boundingBox();
  const createProject = await page
    .getByRole('button', { name: 'Create project' })
    .boundingBox();
  expect(createProject?.width).toBe(projectName?.width);
});

test('onboarding remains centered on a wide screen', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await gotoJourney(page, onboardingJourney('organization'));

  const workspaceSetup = page.getByRole('region', {
    name: 'Workspace setup',
  });
  const bounds = await workspaceSetup.boundingBox();

  expect(bounds?.width).toBe(640);
  expect(bounds?.x).toBe(320);
  await expect(page.getByRole('complementary')).toHaveCount(0);
});

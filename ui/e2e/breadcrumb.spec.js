const { test, expect, authenticatePage } = require('./fixtures');

const ASSISTANT_ID = '2301664620831571968';

test('workspace breadcrumb uses route metadata and dynamic entity names', async ({
  page,
}) => {
  await authenticatePage(page);
  await page.goto(`/deployment/assistant/${ASSISTANT_ID}/overview`);
  await expect(page.getByText('Assistant activity')).toBeVisible();

  const breadcrumb = page.locator('.global-breadcrumb');
  await expect(
    breadcrumb.getByRole('link', { name: 'Support bot' }),
  ).toHaveAttribute('href', `/deployment/assistant/${ASSISTANT_ID}`);
  await expect(breadcrumb).not.toContainText(ASSISTANT_ID);
  await expect(breadcrumb.getByText('Overview')).toHaveAttribute(
    'aria-current',
    'page',
  );

  await breadcrumb
    .getByRole('button', { name: 'Show hidden breadcrumb items' })
    .click();
  await expect(
    page.getByRole('menuitem', { name: 'Assistants' }),
  ).toBeVisible();
});

test('dashboard breadcrumb exposes only the current page', async ({ page }) => {
  await authenticatePage(page);
  await page.goto('/dashboard');
  await expect(
    page.getByRole('heading', { name: 'Top assistants' }),
  ).toBeVisible();

  const breadcrumb = page.locator('.global-breadcrumb');
  await expect(breadcrumb.getByText('Dashboard')).toHaveAttribute(
    'aria-current',
    'page',
  );
  await expect(breadcrumb.getByRole('link', { name: 'Dashboard' })).toHaveCount(
    0,
  );
});

test('mobile breadcrumb keeps overflow and the current route visible', async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await authenticatePage(page);
  await page.goto(
    `/deployment/assistant/${ASSISTANT_ID}/sessions/2301664620831572101`,
  );
  await expect(page.getByText('2 messages')).toBeVisible();

  const breadcrumb = page.locator('.global-breadcrumb');
  await expect(
    breadcrumb.getByRole('button', { name: 'Show hidden breadcrumb items' }),
  ).toBeVisible();
  await expect(breadcrumb.getByText('Session details')).toHaveAttribute(
    'aria-current',
    'page',
  );
});

test('mobile two-item breadcrumb keeps its parent link', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await authenticatePage(page);
  await page.goto('/organization/users');
  await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible();

  const breadcrumb = page.locator('.global-breadcrumb');
  await expect(
    breadcrumb.getByRole('link', { name: 'Organization' }),
  ).toHaveAttribute('href', '/organization');
  await expect(breadcrumb.getByText('Users')).toHaveAttribute(
    'aria-current',
    'page',
  );
});

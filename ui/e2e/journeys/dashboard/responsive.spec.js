const {
  COLOR_MODES,
  checkA11y,
  expect,
  expandDashboardForScreenshot,
  gotoJourney,
  setColorMode,
  test,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');
const { getDashboardRequests } = require('../../mocks/clients/dashboard');

const dashboardJourney = routeJourneys.find(
  journey => journey.id === 'dashboard.home',
);

test('dashboard requests each selected range exactly once', async ({
  page,
}) => {
  await gotoJourney(page, dashboardJourney);
  await expect.poll(() => getDashboardRequests(page).length).toBe(1);
  expect(getDashboardRequests(page)[0].durationMs).toBe(
    30 * 24 * 60 * 60 * 1000,
  );

  await page.getByLabel('Date range').click();
  await page.getByText('Last 7 days', { exact: true }).click();

  await expect.poll(() => getDashboardRequests(page).length).toBe(2);
  expect(getDashboardRequests(page)[1].durationMs).toBe(
    7 * 24 * 60 * 60 * 1000,
  );
});

test('mobile sidebar expands without reducing dashboard width', async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await gotoJourney(page, dashboardJourney);

  const sidebar = page.locator('aside').first();
  const dashboard = page.locator('[data-dashboard-page]');
  await expect(sidebar).toHaveCSS('width', '48px');
  await expect(dashboard).toHaveCSS('width', '342px');

  await page.getByRole('button', { name: 'Expand sidebar' }).click();
  await expect(sidebar).toHaveCSS('width', '256px');
  await expect(dashboard).toHaveCSS('width', '390px');
  await expect(
    page.getByText('Dashboard', { exact: true }).first(),
  ).toBeVisible();

  await page.getByRole('button', { name: 'Collapse sidebar' }).click();
  await expect(sidebar).toHaveCSS('width', '48px');
  await expect(dashboard).toHaveCSS('width', '342px');
});

for (const viewport of [
  { name: 'tablet', width: 900, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  for (const colorMode of COLOR_MODES) {
    test(`dashboard is usable on ${viewport.name} in ${colorMode} mode`, async ({
      page,
    }) => {
      await page.setViewportSize({
        width: viewport.width,
        height: viewport.height,
      });
      await setColorMode(page, colorMode);
      await gotoJourney(page, dashboardJourney);
      await checkA11y(page, dashboardJourney);
      await expandDashboardForScreenshot(page);

      const content = page.locator('[data-dashboard-content]');
      await expect(content).toHaveScreenshot(
        ['dashboard', `home-${viewport.name}-${colorMode}.png`],
        { maxDiffPixelRatio: 0.01 },
      );
      await expect(
        page.getByText('Recent failures', { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText('Top assistants', { exact: true }),
      ).toBeVisible();
    });
  }
}

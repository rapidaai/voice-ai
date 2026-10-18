const {
  COLOR_MODES,
  DASHBOARD_PATH,
  SCREENSHOT_VIEWPORTS,
  expect,
  expandDashboardForScreenshot,
  gotoJourney,
  setColorMode,
  test,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');

const dashboardJourney = routeJourneys.find(
  journey => journey.id === 'dashboard.home',
);

const dashboardStates = [
  {
    name: 'loading',
    expectText: 'Project overview',
    async install(page) {
      let releaseRequest;
      const requestGate = new Promise(resolve => {
        releaseRequest = resolve;
      });
      const handler = async route => {
        await requestGate;
        await route.fallback();
      };
      await page.route(DASHBOARD_PATH, handler);
      return async () => {
        releaseRequest();
        await page.unroute(DASHBOARD_PATH, handler);
      };
    },
    async assert(page) {
      await expect(
        page.getByRole('status', { name: 'Loading project dashboard' }),
      ).toBeVisible();
    },
  },
  {
    name: 'success',
    expectText: 'Project overview',
    async install() {
      return async () => {};
    },
    async assert(page) {
      await expect(
        page.getByText('Top assistants', { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText('Recent failures', { exact: true }),
      ).toBeVisible();
    },
  },
  {
    name: 'error',
    expectText: 'Project dashboard is unavailable',
    async install(page) {
      const handler = route => route.abort('failed');
      await page.route(DASHBOARD_PATH, handler);
      return () => page.unroute(DASHBOARD_PATH, handler);
    },
    async assert(page) {
      await expect(
        page.getByText('Project dashboard is unavailable', { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole('button', { name: 'Try again' }),
      ).toBeVisible();
    },
  },
];

for (const dashboardState of dashboardStates) {
  for (const viewport of SCREENSHOT_VIEWPORTS) {
    for (const colorMode of COLOR_MODES) {
      test(`dashboard ${dashboardState.name} state at ${viewport.name} in ${colorMode} mode`, async ({
        page,
      }) => {
        await page.setViewportSize({
          width: viewport.width,
          height: viewport.height,
        });
        await setColorMode(page, colorMode);
        const cleanup = await dashboardState.install(page);

        try {
          await gotoJourney(page, {
            ...dashboardJourney,
            expectText: dashboardState.expectText,
            readyText:
              dashboardState.name === 'success'
                ? dashboardJourney.readyText
                : undefined,
          });
          await dashboardState.assert(page);
          await page.evaluate(() => document.fonts.ready);
          if (dashboardState.name !== 'error') {
            await expandDashboardForScreenshot(page);
          }

          await expect(
            page.locator('[data-dashboard-content]'),
          ).toHaveScreenshot(
            [
              'dashboard',
              dashboardState.name,
              `${viewport.name}-${colorMode}.png`,
            ],
            { maxDiffPixelRatio: 0.01 },
          );
        } finally {
          await cleanup();
        }
      });
    }
  }
}

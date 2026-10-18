const {
  COLOR_MODES,
  SCREENSHOT_VIEWPORTS,
  expect,
  gotoJourney,
  setColorMode,
  test,
} = require('./fixtures');
const { routeJourneys } = require('./route-manifest');

const emptyJourneyIds = new Set([
  'deployment.endpoint',
  'deployment.endpointVersions',
  'integration.projectCredential',
  'integration.personalCredential',
  'logs.traces',
  'organization.users',
  'organization.projects',
  'deployment.assistantDeployments',
  'deployment.assistantTools',
  'deployment.assistantWebhooks',
  'deployment.assistantAuthentication',
  'deployment.assistantTelemetry',
  'deployment.assistantStorage',
  'deployment.assistantAnalysis',
]);

const screenshotJourneys = routeJourneys.filter(
  journey =>
    journey.id !== 'dashboard.home' && journey.checks.includes('screenshot'),
);

const slug = value =>
  value
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/[^a-z0-9]+/gi, '-')
    .toLowerCase();

for (const journey of screenshotJourneys) {
  const [area, ...name] = journey.id.split('.');
  const state = journey.id.endsWith('NotFound')
    ? 'error'
    : emptyJourneyIds.has(journey.id)
      ? 'empty'
      : 'success';

  for (const viewport of SCREENSHOT_VIEWPORTS) {
    for (const colorMode of COLOR_MODES) {
      test(`${journey.id} ${state} screenshot at ${viewport.name} in ${colorMode} mode`, async ({
        page,
      }) => {
        await page.setViewportSize(viewport);
        await setColorMode(page, colorMode);
        await gotoJourney(page, journey);
        await expect(page.locator('html')).toHaveAttribute(
          'data-color-mode',
          colorMode,
        );
        await page.evaluate(() => document.fonts.ready);

        await expect(page).toHaveScreenshot(
          [
            slug(area),
            slug(name.join('-')),
            state,
            `${viewport.name}-${colorMode}.png`,
          ],
          {
            fullPage: true,
            maxDiffPixelRatio: area === 'static' ? 0.02 : 0.01,
          },
        );
      });
    }
  }
}

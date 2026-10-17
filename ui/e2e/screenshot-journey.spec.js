const {
  COLOR_MODES,
  expect,
  expandDashboardForScreenshot,
  gotoJourney,
  setColorMode,
  screenshotBaselinePath,
  test,
} = require('./fixtures');
const { journeysWithCheck } = require('./route-manifest');

for (const journey of journeysWithCheck('screenshot')) {
  for (const colorMode of COLOR_MODES) {
    test(`${journey.id} matches screenshot journey in ${colorMode} mode`, async ({
      page,
    }) => {
      await setColorMode(page, colorMode);
      await gotoJourney(page, journey);
      await expect(page.locator('html')).toHaveAttribute(
        'data-color-mode',
        colorMode,
      );
      await page.evaluate(async () => {
        await document.fonts.ready;
      });

      const screenshotPath = screenshotBaselinePath(journey, colorMode);
      if (journey.id === 'dashboard.home') {
        await expandDashboardForScreenshot(page);
        await expect(page.locator('[data-dashboard-content]')).toHaveScreenshot(
          screenshotPath,
          { maxDiffPixelRatio: 0.01 },
        );
      } else {
        await expect(page).toHaveScreenshot(screenshotPath, {
          fullPage: true,
          maxDiffPixelRatio: 0.01,
        });
      }
    });
  }
}

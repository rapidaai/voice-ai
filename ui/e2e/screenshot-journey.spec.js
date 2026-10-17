const {
  COLOR_MODES,
  expect,
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

      await expect(page).toHaveScreenshot(
        screenshotBaselinePath(journey, colorMode),
        {
          fullPage: true,
          maxDiffPixelRatio: 0.01,
        },
      );
    });
  }
}

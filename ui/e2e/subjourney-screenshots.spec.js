const {
  COLOR_MODES,
  SCREENSHOT_VIEWPORTS,
  expect,
  gotoJourney,
  setColorMode,
  test,
} = require('./fixtures');
const { routeJourneys } = require('./route-manifest');

const routeJourney = id => routeJourneys.find(journey => journey.id === id);

const subjourneys = [
  {
    id: 'organization/invite-user',
    journey: routeJourney('organization.users'),
    async open(page) {
      await page
        .getByRole('button', { name: 'Invite user', exact: true })
        .first()
        .click();
      await expect(
        page.getByRole('heading', { name: 'Invite user to organization' }),
      ).toBeVisible();
    },
  },
  {
    id: 'organization/create-project',
    journey: routeJourney('organization.projects'),
    async open(page) {
      await page
        .getByRole('button', { name: 'Create new project', exact: true })
        .first()
        .click();
      await expect(
        page.getByRole('heading', { name: 'Create a project' }),
      ).toBeVisible();
    },
  },
  {
    id: 'account/notifications',
    journey: routeJourney('account.settings'),
    async open(page) {
      await page.getByRole('tab', { name: 'Notification' }).click();
      await expect(
        page.getByRole('heading', { name: 'Notification Preferences' }),
      ).toBeVisible();
    },
  },
];

for (const subjourney of subjourneys) {
  for (const viewport of SCREENSHOT_VIEWPORTS) {
    for (const colorMode of COLOR_MODES) {
      test(`${subjourney.id} screenshot at ${viewport.name} in ${colorMode} mode`, async ({
        page,
      }) => {
        await page.setViewportSize(viewport);
        await setColorMode(page, colorMode);
        await gotoJourney(page, subjourney.journey);
        await subjourney.open(page);
        await page.evaluate(() => document.fonts.ready);

        await expect(page).toHaveScreenshot(
          [
            ...subjourney.id.split('/'),
            'success',
            `${viewport.name}-${colorMode}.png`,
          ],
          { fullPage: true, maxDiffPixelRatio: 0.01 },
        );
      });
    }
  }
}

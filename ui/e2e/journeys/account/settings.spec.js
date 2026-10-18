const { expect, gotoJourney, test } = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');

const accountJourney = routeJourneys.find(
  journey => journey.id === 'account.settings',
);

test('account settings opens notification preferences', async ({ page }) => {
  await gotoJourney(page, accountJourney);

  await page.getByRole('tab', { name: 'Notification' }).click();

  await expect(
    page.getByRole('heading', { name: 'Notification Preferences' }),
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'Save changes' }),
  ).toBeVisible();
});

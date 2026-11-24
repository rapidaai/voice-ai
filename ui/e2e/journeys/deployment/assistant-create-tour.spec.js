const {
  authenticatePage,
  checkA11y,
  expect,
  gotoJourney,
  setColorMode,
  test,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');

const createAssistantJourney = routeJourneys.find(
  journey => journey.id === 'deployment.createAssistant',
);

const elementWidth = locator =>
  locator.evaluate(element => element.getBoundingClientRect().width);

test('create assistant tour explains the first configuration step', async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await gotoJourney(page, createAssistantJourney, {
    showInterfaceTour: 'createAssistant',
  });

  const templateCoachmark = page.getByRole('region', {
    name: 'Choose a usecase template coachmark',
  });
  await expect(templateCoachmark).toBeVisible();
  await expect(
    page.locator('[data-coachmark-target="create-assistant-template"]'),
  ).toBeVisible();
  await templateCoachmark.getByRole('button', { name: 'Next' }).click();

  const modelCoachmark = page.getByRole('region', {
    name: 'Select a model and credential coachmark',
  });
  await expect(modelCoachmark).toBeVisible();
  await expect(
    page.locator('[data-coachmark-target="create-assistant-model"]'),
  ).toBeVisible();
  await modelCoachmark.getByRole('button', { name: 'Next' }).click();

  const continueCoachmark = page.getByRole('region', {
    name: 'Continue your setup coachmark',
  });
  await expect(continueCoachmark).toBeVisible();
  await expect(page.getByRole('button', { name: 'Continue' })).toHaveAttribute(
    'aria-expanded',
    'true',
  );

  await checkA11y(page, createAssistantJourney);
  await continueCoachmark.getByRole('button', { name: 'Got it' }).click();

  await expect(page.getByRole('region', { name: /coachmark$/ })).toHaveCount(0);
  await expect
    .poll(() =>
      page.evaluate(() => {
        const stored = window.localStorage.getItem('rpd::__ui_tour');
        return stored ? JSON.parse(stored) : null;
      }),
    )
    .toEqual({
      dashboard: 'completed',
      assistant: 'completed',
      createAssistant: 'completed',
    });
});

test('create assistant tour preserves target widths between steps', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await gotoJourney(page, createAssistantJourney, {
    showInterfaceTour: 'createAssistant',
  });

  const templateTarget = page.locator(
    '[data-coachmark-target="create-assistant-template"]',
  );
  const templateWidthWithCoachmark = await elementWidth(templateTarget);
  await page
    .getByRole('region', { name: 'Choose a usecase template coachmark' })
    .getByRole('button', { name: 'Next' })
    .click();
  const templateWidthWithoutCoachmark = await elementWidth(templateTarget);
  expect(templateWidthWithCoachmark).toBeCloseTo(
    templateWidthWithoutCoachmark,
    0,
  );

  const modelTarget = page.locator(
    '[data-coachmark-target="create-assistant-model"]',
  );
  const headerContent = page.locator('[data-create-page-header-content]');
  const headerRail = page.locator('[data-create-page-header-rail]');
  const stepRail = page.locator('aside').filter({
    has: page.getByText('Configure Assistant', { exact: true }),
  });
  const [headerBox, modelBox] = await Promise.all([
    headerContent.boundingBox(),
    modelTarget.boundingBox(),
  ]);
  expect(headerBox.x).toBeCloseTo(modelBox.x, 0);
  await expect(headerRail).toHaveCSS(
    'background-color',
    await stepRail.evaluate(
      element => getComputedStyle(element).backgroundColor,
    ),
  );

  const modelRowBox = await page
    .locator('.text-provider-combo-row')
    .boundingBox();
  const credentialFieldBox = await page
    .locator('[data-coachmark-target="create-assistant-model"]')
    .locator('.domain-connected-dropdown-label')
    .locator('..')
    .boundingBox();
  expect(credentialFieldBox.y - (modelRowBox.y + modelRowBox.height)).toBe(24);

  const modelWidthWithCoachmark = await elementWidth(modelTarget);
  await page
    .getByRole('region', { name: 'Select a model and credential coachmark' })
    .getByRole('button', { name: 'Next' })
    .click();
  const modelWidthWithoutCoachmark = await elementWidth(modelTarget);
  expect(modelWidthWithCoachmark).toBeCloseTo(modelWidthWithoutCoachmark, 0);

  const continueButton = page.getByRole('button', { name: 'Continue' });
  const continueWidthWithCoachmark = await elementWidth(continueButton);
  await page
    .getByRole('region', { name: 'Continue your setup coachmark' })
    .getByRole('button', { name: 'Got it' })
    .click();
  const continueWidthWithoutCoachmark = await elementWidth(continueButton);
  expect(continueWidthWithCoachmark).toBeCloseTo(
    continueWidthWithoutCoachmark,
    0,
  );
});

for (const colorMode of ['light', 'dark']) {
  test(`create assistant tour in ${colorMode} mode`, async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await setColorMode(page, colorMode);
    await authenticatePage(page, { showInterfaceTour: 'createAssistant' });
    await page.goto(createAssistantJourney.path);

    const steps = [
      {
        action: 'Next',
        name: 'Choose a usecase template coachmark',
        screenshot: `create-assistant-template-${colorMode}.png`,
      },
      {
        action: 'Next',
        name: 'Select a model and credential coachmark',
        screenshot: `create-assistant-model-${colorMode}.png`,
      },
      {
        action: 'Got it',
        name: 'Continue your setup coachmark',
        screenshot: `create-assistant-continue-${colorMode}.png`,
      },
    ];

    for (const step of steps) {
      const coachmark = page.getByRole('region', { name: step.name });
      await expect(coachmark).toBeVisible();
      await expect(coachmark.locator('.app-coachmark__content')).toHaveCSS(
        'width',
        '290px',
      );
      await page.mouse.move(0, 0);
      await expect(page.getByRole('tooltip')).toBeHidden();
      await page.evaluate(() => document.fonts.ready);
      await expect(page).toHaveScreenshot(
        ['assistant', 'interface-tour', step.screenshot],
        { fullPage: true, maxDiffPixelRatio: 0.01 },
      );
      await coachmark.getByRole('button', { name: step.action }).click();
    }
  });
}

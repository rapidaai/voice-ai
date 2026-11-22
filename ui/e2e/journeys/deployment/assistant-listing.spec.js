const {
  ASSISTANT_LIST_PATH,
  checkA11y,
  expect,
  gotoJourney,
  test,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');
const {
  EMPTY_ASSISTANT_LIST_RESPONSE,
  fulfillGrpcWeb,
} = require('../../mocks/clients/assistant');

const assistantJourney = routeJourneys.find(
  journey => journey.id === 'deployment.assistant',
);
const assistantListError =
  'Something went wrong while retrieving your assistants. Please refresh the page or try again later.';

test('assistant listing guides a first visit through assistant creation options', async ({
  page,
}) => {
  await gotoJourney(page, assistantJourney, {
    showInterfaceTour: 'assistant',
  });

  const createCoachmark = page.getByRole('region', {
    name: 'Create an assistant coachmark',
  });
  await expect(createCoachmark).toBeVisible();
  await expect(createCoachmark.locator('.app-coachmark__content')).toHaveCSS(
    'width',
    '290px',
  );
  await expect(
    page.getByRole('button', { name: 'Create new assistant' }).first(),
  ).toHaveAttribute('aria-expanded', 'true');
  await createCoachmark
    .getByRole('button', { name: 'Show assistant types' })
    .click();

  const createDialog = page.getByRole('dialog').filter({
    has: page.getByRole('heading', { name: 'Create an assistant' }),
  });
  const typesCoachmark = page.getByRole('region', {
    name: 'Choose an assistant type coachmark',
  });
  await expect(createDialog).toBeVisible();
  await expect(typesCoachmark).toBeVisible();
  await expect(typesCoachmark.locator('.app-coachmark__content')).toHaveCSS(
    'width',
    '290px',
  );
  await expect(
    page.locator('[data-coachmark-target="assistant-types"]'),
  ).toBeVisible();
  await expect(createDialog.locator('.cds--modal-content')).toHaveCSS(
    'overflow-y',
    'visible',
  );
  await expect(
    createDialog.getByText('Prompting', { exact: true }),
  ).toBeVisible();
  for (const assistantType of ['From Prompt', 'AgentKit', 'Agentflow']) {
    await expect(
      createDialog.getByRole('heading', { name: assistantType }),
    ).toBeVisible();
  }

  await checkA11y(page, assistantJourney);
  await typesCoachmark.getByRole('button', { name: 'Got it' }).click();

  await expect(page.getByRole('region', { name: /coachmark$/ })).toHaveCount(0);
  await expect(createDialog).toBeVisible();
  await expect(
    createDialog.getByRole('button', { name: /From Prompt/ }),
  ).toBeFocused();
  await expect
    .poll(() =>
      page.evaluate(() => {
        const stored = window.localStorage.getItem('rpd::__ui_tour');
        return stored ? JSON.parse(stored) : null;
      }),
    )
    .toEqual({ dashboard: 'completed', assistant: 'completed' });
});

test('assistant tour keeps the type chooser scrollable on mobile', async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await gotoJourney(page, assistantJourney, {
    showInterfaceTour: 'assistant',
  });

  await page.getByRole('button', { name: 'Show assistant types' }).click();

  const createDialog = page.getByRole('dialog').filter({
    has: page.getByRole('heading', { name: 'Create an assistant' }),
  });
  await expect(createDialog.locator('.cds--modal-content')).toHaveCSS(
    'overflow-y',
    'auto',
  );
  const gotIt = page
    .getByRole('region', { name: 'Choose an assistant type coachmark' })
    .getByRole('button', { name: 'Got it' });
  await expect(gotIt).toBeVisible();
  await gotIt.click();
  await expect(page.getByRole('region', { name: /coachmark$/ })).toHaveCount(0);
  await expect(createDialog).toBeVisible();
});

test('assistant listing renders a successful empty result', async ({
  page,
}) => {
  let requestCount = 0;

  await page.route(ASSISTANT_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, EMPTY_ASSISTANT_LIST_RESPONSE);
  });

  page.on('request', request => {
    if (
      request.method() === 'POST' &&
      request.url().endsWith('/assistant_api.AssistantService/GetAllAssistant')
    ) {
      requestCount += 1;
    }
  });

  await gotoJourney(page, assistantJourney);
  await page.waitForLoadState('networkidle');

  await expect(page.getByText('No assistants', { exact: true })).toBeVisible();
  await expect(page.getByText(assistantListError)).toHaveCount(0);
  expect(requestCount).toBe(1);
});

test('assistant listing opens a populated assistant overview', async ({
  page,
}) => {
  await gotoJourney(page, assistantJourney);

  await expect(page.getByText('Support bot', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: /Support bot/ }).click();

  await expect(page).toHaveURL(/\/deployment\/assistant\/\d+\/overview$/);
  await expect(
    page.getByRole('heading', { name: 'Assistant activity' }),
  ).toBeVisible();
});

test('assistant listing reports one failed initial request', async ({
  page,
}) => {
  let requestCount = 0;

  await page.route(ASSISTANT_LIST_PATH, route => {
    if (route.request().method() !== 'POST') {
      return route.fallback();
    }

    requestCount += 1;
    return route.abort('failed');
  });

  await gotoJourney(page, assistantJourney);
  await page.waitForLoadState('networkidle');

  expect(requestCount).toBe(1);
  await expect(page.getByText(assistantListError)).toHaveCount(1);
});

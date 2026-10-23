const {
  COLOR_MODES,
  SCREENSHOT_VIEWPORTS,
  authenticatePage,
  expect,
  setColorMode,
  test,
} = require('../../fixtures');
const {
  ASSISTANT_CONVERSATION_LIST_PATH,
  ASSISTANT_DETAIL_PATH,
  ASSISTANT_LIST_PATH,
  EMPTY_ASSISTANT_LIST_RESPONSE,
  buildConversationListResponse,
  fulfillGrpcWeb,
  mockPopulatedAssistantDeployments,
} = require('../../mocks/clients/assistant');

const assistantPath = '/deployment/assistant';
const overviewPath = '/deployment/assistant/2301664620831571968/overview';
const sessionsPath = '/deployment/assistant/2301664620831571968/sessions';
const deploymentsPath = '/deployment/assistant/2301664620831571968/deployment';

const scenarios = [
  {
    feature: 'listing',
    state: 'loading',
    path: assistantPath,
    setup: page => page.route(ASSISTANT_LIST_PATH, () => {}),
    assert: page => expect(page.getByLabel('Loading assistants')).toBeVisible(),
  },
  {
    feature: 'listing',
    state: 'success',
    path: assistantPath,
    setup: async () => {},
    assert: page =>
      expect(page.getByText('Support bot', { exact: true })).toBeVisible(),
  },
  {
    feature: 'listing',
    state: 'empty',
    path: assistantPath,
    setup: page =>
      page.route(ASSISTANT_LIST_PATH, route =>
        fulfillGrpcWeb(route, EMPTY_ASSISTANT_LIST_RESPONSE),
      ),
    assert: page =>
      expect(page.getByText('No assistants', { exact: true })).toBeVisible(),
  },
  {
    feature: 'listing',
    state: 'error',
    path: assistantPath,
    setup: page =>
      page.route(ASSISTANT_LIST_PATH, route => route.abort('failed')),
    assert: page =>
      expect(
        page.getByText(
          'Something went wrong while retrieving your assistants. Please refresh the page or try again later.',
        ),
      ).toBeVisible(),
  },
  {
    feature: 'overview',
    state: 'loading',
    path: overviewPath,
    setup: page => page.route(ASSISTANT_DETAIL_PATH, () => {}),
    assert: page =>
      expect(page.locator('.cds--skeleton__text').first()).toBeVisible(),
  },
  {
    feature: 'overview',
    state: 'success',
    path: overviewPath,
    setup: async () => {},
    assert: page =>
      expect(
        page.getByRole('heading', { name: 'Assistant activity' }),
      ).toBeVisible(),
  },
  {
    feature: 'overview',
    state: 'error',
    path: overviewPath,
    setup: page =>
      page.route(ASSISTANT_DETAIL_PATH, route => route.abort('failed')),
    assert: async page => {
      await expect(
        page.getByRole('heading', { name: 'Assistant not available' }),
      ).toBeVisible();
      await expect(page.getByText('403', { exact: true })).toBeVisible();
      await expect(
        page.getByRole('button', { name: 'Back to assistants' }),
      ).toBeVisible();
    },
  },
  {
    feature: 'deployments',
    state: 'empty',
    path: deploymentsPath,
    setup: async () => {},
    assert: async page => {
      await expect(
        page.getByRole('heading', { name: 'Deployment', exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText(
          'Choose how you want to test or deliver your assistant.',
          { exact: true },
        ),
      ).toBeVisible();
    },
  },
  {
    feature: 'deployments',
    state: 'populated',
    path: deploymentsPath,
    setup: page => mockPopulatedAssistantDeployments(page),
    assert: async page => {
      await expect(page.getByText('vrsn_2301664620831573001')).toBeVisible();
      await expect(page.getByText('vrsn_2301664620831573002')).toBeVisible();
      await expect(page.getByText('vrsn_2301664620831573003')).toBeVisible();
      await expect(page.getByText('vrsn_2301664620831573004')).toBeVisible();
      await expect(page.getByText('No deployments found')).toHaveCount(0);
    },
  },
  {
    feature: 'sessions',
    state: 'loading',
    path: sessionsPath,
    setup: page => page.route(ASSISTANT_CONVERSATION_LIST_PATH, () => {}),
    assert: page => expect(page.locator('.cds--loading').first()).toBeVisible(),
  },
  {
    feature: 'sessions',
    state: 'success',
    path: sessionsPath,
    setup: async () => {},
    assert: page =>
      expect(page.getByText('customer@example.test')).toBeVisible(),
  },
  {
    feature: 'sessions',
    state: 'empty',
    path: sessionsPath,
    setup: page =>
      page.route(ASSISTANT_CONVERSATION_LIST_PATH, route =>
        fulfillGrpcWeb(route, buildConversationListResponse([])),
      ),
    assert: page =>
      expect(page.getByText('No conversations found')).toBeVisible(),
  },
  {
    feature: 'sessions',
    state: 'error',
    path: sessionsPath,
    setup: page =>
      page.route(ASSISTANT_CONVERSATION_LIST_PATH, route =>
        route.abort('failed'),
      ),
    assert: async page => {
      await expect(
        page.getByRole('heading', { name: 'Conversations unavailable' }),
      ).toBeVisible();
      await expect(
        page.getByRole('button', { name: 'Try again' }),
      ).toBeVisible();
      await expect(page.getByText('No conversations found')).toHaveCount(0);
    },
  },
];

for (const scenario of scenarios) {
  for (const viewport of SCREENSHOT_VIEWPORTS) {
    for (const colorMode of COLOR_MODES) {
      test(`assistant ${scenario.feature} ${scenario.state} at ${viewport.name} in ${colorMode} mode`, async ({
        page,
      }) => {
        await page.setViewportSize(viewport);
        await setColorMode(page, colorMode);
        await authenticatePage(page);
        await scenario.setup(page);
        await page.goto(scenario.path);
        await scenario.assert(page);
        await page.evaluate(() => document.fonts.ready);

        await expect(page).toHaveScreenshot(
          [
            'assistant',
            scenario.feature,
            scenario.state,
            `${viewport.name}-${colorMode}.png`,
          ],
          { fullPage: true, maxDiffPixelRatio: 0.01 },
        );
      });
    }
  }
}

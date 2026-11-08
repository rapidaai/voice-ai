const {
  COLOR_MODES,
  DASHBOARD_PATH,
  SCREENSHOT_VIEWPORTS,
  checkA11y,
  expect,
  expandDashboardForScreenshot,
  gotoJourney,
  setColorMode,
  test,
} = require('../../fixtures');
const {
  installEmptyDashboardClientMock,
} = require('../../mocks/clients/dashboard');
const { routeJourneys } = require('../../route-manifest');

const dashboardJourney = routeJourneys.find(
  journey => journey.id === 'dashboard.home',
);

const dashboardStates = [
  {
    name: 'loading',
    expectText: 'Loading project dashboard',
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
    expectText: 'Top Assistants',
    async install() {
      return async () => {};
    },
    async assert(page) {
      await expect(
        page.getByText('Top Assistants', { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText('Recent Failures', { exact: true }),
      ).toBeVisible();
    },
  },
  {
    name: 'empty',
    expectText: 'Build your first dashboard insights',
    async install(page) {
      return installEmptyDashboardClientMock(page);
    },
    async assert(page) {
      await expect(
        page.getByRole('heading', {
          name: 'Build your first dashboard insights',
        }),
      ).toBeVisible();
      await expect(
        page.getByRole('link', { name: 'Open assistants' }),
      ).toHaveAttribute('href', '/deployment/assistant');
      await expect(
        page.getByLabel('Dashboard setup checklist'),
      ).toBeVisible();
      await expect(page.getByLabel('Dashboard setup checklist')).toHaveCSS(
        'position',
        'fixed',
      );
      await expect(
        page.getByText('25% complete', { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole('button', {
          name: 'Collapse or expand dashboard setup',
        }),
      ).toHaveAttribute('aria-expanded', 'true');
      await expect(
        page.getByRole('link', { name: 'Create assistant' }),
      ).toHaveAttribute('href', '/deployment/assistant/create-assistant');
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

test('dashboard empty state links to assistant creation', async ({ page }) => {
  const cleanup = await installEmptyDashboardClientMock(page);

  try {
    await gotoJourney(page, {
      ...dashboardJourney,
      expectText: 'Build your first dashboard insights',
      readyText: undefined,
    });
    await expect(
      page.getByRole('link', { name: 'Open assistants' }),
    ).toHaveAttribute('href', '/deployment/assistant');
    await expect(
      page.getByRole('link', { name: 'Create assistant' }),
    ).toHaveAttribute('href', '/deployment/assistant/create-assistant');
    const checklistToggle = page.getByRole('button', {
      name: 'Collapse or expand dashboard setup',
    });
    await checklistToggle.click();
    await expect(checklistToggle).toHaveAttribute('aria-expanded', 'false');
    await checklistToggle.click();
    await expect(checklistToggle).toHaveAttribute('aria-expanded', 'true');
  } finally {
    await cleanup();
  }
});

test('dashboard checklist actions open their intended assistant destinations', async ({
  page,
}) => {
  const actions = [
    ['Create an assistant', '/deployment/assistant/create-assistant'],
    ['Configure and test your assistant', '/deployment/assistant'],
    ['View all assistants', '/deployment/assistant'],
  ];

  for (const [name, destination] of actions) {
    const cleanup = await installEmptyDashboardClientMock(page);
    try {
      await gotoJourney(page, {
        ...dashboardJourney,
        expectText: 'Build your first dashboard insights',
        readyText: undefined,
      });
      await page.getByRole('button', { name }).click();
      await page.waitForURL(url => url.pathname === destination);
      expect(new URL(page.url()).pathname).toBe(destination);
    } finally {
      await cleanup();
    }
  }
});

test('dashboard checklist stays in page flow on a short viewport', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1024, height: 600 });
  const cleanup = await installEmptyDashboardClientMock(page);

  try {
    await gotoJourney(page, {
      ...dashboardJourney,
      expectText: 'Build your first dashboard insights',
      readyText: undefined,
    });
    const emptyCard = page.locator('[data-dashboard-empty] > div').first();
    const checklist = page.getByLabel('Dashboard setup checklist');
    await expect(checklist).toHaveCSS('position', 'static');
    const [emptyCardBounds, checklistBounds] = await Promise.all([
      emptyCard.boundingBox(),
      checklist.boundingBox(),
    ]);
    expect(emptyCardBounds).not.toBeNull();
    expect(checklistBounds).not.toBeNull();
    expect(checklistBounds.y).toBeGreaterThanOrEqual(
      emptyCardBounds.y + emptyCardBounds.height,
    );
  } finally {
    await cleanup();
  }
});

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
          if (dashboardState.name === 'empty' && viewport.name === 'lg') {
            await checkA11y(page, dashboardJourney);
          }
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

for (const colorMode of COLOR_MODES) {
  test(`dashboard empty state below sm in ${colorMode} mode`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await setColorMode(page, colorMode);
    const cleanup = await installEmptyDashboardClientMock(page);

    try {
      await gotoJourney(page, {
        ...dashboardJourney,
        expectText: 'Build your first dashboard insights',
        readyText: undefined,
      });
      await expect(
        page.getByRole('heading', {
          name: 'Build your first dashboard insights',
        }),
      ).toBeVisible();
      await expect(
        page.getByRole('link', { name: 'Open assistants' }),
      ).toBeVisible();
      await expect(page.getByLabel('Dashboard setup checklist')).toBeVisible();
      await expect(page.getByLabel('Dashboard setup checklist')).toHaveCSS(
        'position',
        'static',
      );
      await page.evaluate(() => document.fonts.ready);
      await expandDashboardForScreenshot(page);

      await expect(page.locator('[data-dashboard-content]')).toHaveScreenshot(
        ['dashboard', 'empty', `mobile-${colorMode}.png`],
        { maxDiffPixelRatio: 0.015 },
      );
    } finally {
      await cleanup();
    }
  });
}

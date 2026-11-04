const {
  ASSISTANT_DETAIL_PATH,
  authenticatePage,
  expect,
  gotoJourney,
  setColorMode,
  test,
} = require('../../fixtures');
const {
  ASSISTANT_CONVERSATION_DETAIL_PATH,
  mockPopulatedAssistantDeployments,
} = require('../../mocks/clients/assistant');
const { routeJourneys } = require('../../route-manifest');

const journey = id => routeJourneys.find(candidate => candidate.id === id);

const deploymentDetailScreenshots = [
  {
    name: 'SDK / API',
    dialogName: '2301664620831573002',
    folder: 'api-details',
  },
  {
    name: 'Web Widget',
    dialogName: '2301664620831573004',
    folder: 'widget-details',
  },
  {
    name: 'Phone Call',
    dialogName: '2301664620831573003',
    folder: 'phone-details',
  },
];

for (const colorMode of ['light', 'dark']) {
  test(`debugger deployment uses one Carbon detail surface in ${colorMode} mode`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await mockPopulatedAssistantDeployments(page);
    await page.goto('/deployment/assistant/2301664620831571968/deployment');

    const debuggerRow = page.getByRole('row', { name: /Debugger/ });
    await debuggerRow
      .getByRole('button', { name: 'Actions for Debugger' })
      .click();
    await page.getByText('View deployment details', { exact: true }).click();

    const drawer = page.getByRole('dialog', { name: 'Debugger' });
    await expect(drawer).toBeVisible();
    await expect(
      page.getByText('Edit deployment', { exact: true }),
    ).toBeHidden();
    await page.evaluate(() => document.activeElement?.blur());
    await page.mouse.move(0, 0);
    await expect(page.getByRole('tooltip')).toBeHidden();
    await expect(drawer.getByLabel('Debugger configuration')).toBeVisible();
    await expect(drawer.getByText('Hello, how can I help?')).toBeVisible();
    await expect(drawer.getByText('Deepgram · nova-3')).toBeVisible();
    await expect(drawer.getByText('Cartesia · sonic-2')).toBeVisible();
    await expect(drawer.getByRole('tablist')).toHaveCount(0);
    await expect(
      drawer.getByRole('button', { name: 'Edit configuration' }),
    ).toBeVisible();
    await expect(
      drawer.getByRole('button', { name: 'Open debugger' }),
    ).toBeVisible();

    await page.evaluate(() => document.fonts.ready);
    await expect(drawer).toHaveScreenshot(
      ['assistant', 'debugger-details', 'success', `lg-${colorMode}.png`],
      { animations: 'disabled', maxDiffPixelRatio: 0.01 },
    );
  });

  test(`other deployment detail drawers retain ${colorMode} screenshots`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await mockPopulatedAssistantDeployments(page);
    await page.goto('/deployment/assistant/2301664620831571968/deployment');

    for (const deployment of deploymentDetailScreenshots) {
      const row = page.getByRole('row', { name: deployment.name });
      await row
        .getByRole('button', { name: `Actions for ${deployment.name}` })
        .click();
      await page.getByText('View deployment details', { exact: true }).click();

      const drawer = page.getByRole('dialog', {
        name: deployment.dialogName,
      });
      await expect(drawer).toBeVisible();
      await expect(
        page.getByText('Edit deployment', { exact: true }),
      ).toBeHidden();
      await page.evaluate(() => document.activeElement?.blur());
      await page.mouse.move(0, 0);
      await expect(page.getByRole('tooltip')).toBeHidden();
      await page.evaluate(() => document.fonts.ready);
      await expect(drawer).toHaveScreenshot(
        ['assistant', deployment.folder, 'success', `lg-${colorMode}.png`],
        { animations: 'disabled', maxDiffPixelRatio: 0.01 },
      );

      await drawer.getByLabel('Close').click();
      await expect(drawer).toBeHidden();
    }
  });
}

test('deployment rows use the Carbon expanded chevron state', async ({
  page,
}) => {
  await authenticatePage(page);
  await mockPopulatedAssistantDeployments(page);
  await page.goto('/deployment/assistant/2301664620831571968/deployment');

  const expandButton = page.getByRole('button', {
    name: /SDK \/ API deployment details$/,
  });
  const chevron = expandButton.locator('.cds--table-expand__svg');
  const collapsedTransform = await chevron.evaluate(
    element => getComputedStyle(element).transform,
  );

  await expect(expandButton).toHaveAttribute('aria-expanded', 'false');
  await expandButton.click();
  await expect(expandButton).toHaveAttribute('aria-expanded', 'true');
  await expect(page.getByText('Session behavior')).toBeVisible();
  await expect
    .poll(() =>
      chevron.evaluate(element => getComputedStyle(element).transform),
    )
    .not.toBe(collapsedTransform);
});

test('deployment loading does not flash the empty-state tiles', async ({
  page,
}) => {
  await authenticatePage(page);
  await mockPopulatedAssistantDeployments(page, { delayMs: 1000 });
  await page.goto('/deployment/assistant/2301664620831571968/deployment');

  const loadingStatus = page.getByRole('status');
  await expect(loadingStatus).toHaveText('Loading deployments');
  await expect(
    page.getByText('Choose how you want to test or deliver your assistant.'),
  ).toBeHidden();
  await expect(page.getByRole('row', { name: /SDK \/ API/ })).toBeVisible();
  await expect(loadingStatus).toBeEmpty();
});

test('web widget deployment details use the shared integration design', async ({
  page,
}) => {
  await authenticatePage(page);
  await mockPopulatedAssistantDeployments(page);
  await page.goto('/deployment/assistant/2301664620831571968/deployment');
  await expect(page.getByText('vrsn_2301664620831573004')).toBeVisible();

  const widgetRow = page.getByRole('row', { name: /Web Widget/ });
  await widgetRow
    .getByRole('button', { name: 'Actions for Web Widget' })
    .click();
  await page.getByText('View deployment details', { exact: true }).click();

  const drawer = page.getByRole('dialog', {
    name: '2301664620831573004',
  });
  await expect(drawer).toBeVisible();
  const tabList = drawer.getByRole('tablist', {
    name: 'Web widget deployment tabs',
  });
  const setupTab = drawer.getByRole('tab', { name: 'Setup' });
  const configurationTab = drawer.getByRole('tab', {
    name: 'Configuration',
  });
  const audioTab = drawer.getByRole('tab', { name: 'Audio' });
  await expect(tabList).toBeVisible();
  await expect(drawer.getByText('Step 1')).toBeVisible();
  await expect(drawer.getByText('Step 2')).toBeVisible();
  await expect(drawer.getByText('Step 3')).toBeVisible();
  await expect(drawer.getByText('Step 4')).toHaveCount(0);
  const actions = drawer.getByLabel('Web widget deployment actions');
  await expect(actions.getByRole('button', { name: 'Close' })).toBeVisible();
  await expect(
    actions.getByRole('button', { name: 'View documentation' }),
  ).toBeVisible();
  await expect(
    drawer.getByRole('heading', { name: 'Choose a project credential' }),
  ).toBeVisible();

  const setupBounds = await setupTab.boundingBox();
  const configurationBounds = await configurationTab.boundingBox();
  const audioBounds = await audioTab.boundingBox();
  expect(setupBounds).not.toBeNull();
  expect(configurationBounds).not.toBeNull();
  expect(audioBounds).not.toBeNull();
  expect(Math.abs(setupBounds.width - audioBounds.width)).toBeLessThanOrEqual(
    1,
  );
  expect(
    Math.abs(configurationBounds.width - audioBounds.width),
  ).toBeLessThanOrEqual(1);

  await configurationTab.click();
  await expect(drawer.getByText('Session settings')).toBeVisible();
  await expect(drawer.getByText('Widget settings')).toBeVisible();
  await expect(drawer.getByText('300 seconds')).toBeVisible();
  await expect(drawer.getByText('10 seconds')).toBeVisible();
  await expect(drawer.getByText('Interruptible')).toBeVisible();
  await expect(drawer.getByText('2 seconds')).toBeVisible();

  await audioTab.click();
  await expect(drawer.getByText('Voice configuration')).toBeVisible();
  await expect(drawer.getByText('Input provider')).toBeVisible();
  await expect(drawer.getByText('Output provider')).toBeVisible();
});

test('populated assistant journeys expose overview, sessions, versions, and settings', async ({
  page,
}) => {
  await gotoJourney(page, journey('deployment.assistantOverview'));
  await expect(
    page.getByRole('heading', { name: 'Assistant activity' }),
  ).toBeVisible();

  await page.getByRole('link', { name: 'Sessions', exact: true }).click();
  await expect(page.getByText('customer@example.test')).toBeVisible();

  await page.getByRole('button', { name: 'Versions', exact: true }).click();
  await page.getByRole('link', { name: 'View all', exact: true }).click();
  await expect(page.getByText('vrsn_2301664620831571999')).toBeVisible();

  await page.getByRole('link', { name: 'General', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: 'General Settings' }),
  ).toBeVisible();
});

test('assistant internal pages share the shell header styling and requested headings', async ({
  page,
}) => {
  const pages = [
    ['deployment.assistantSettings', 'General Settings'],
    ['deployment.assistantAuthentication', 'Authentication'],
    ['deployment.assistantDeployments', 'Deployments'],
    ['deployment.assistantTools', 'Tool & MCP'],
    ['deployment.assistantWebhooks', 'Webhooks'],
    ['deployment.assistantTelemetry', 'Telemetry'],
    ['deployment.assistantStorage', 'Storage'],
    ['deployment.assistantAnalysis', 'Analysis'],
    ['deployment.assistantPlaygroundChat', 'Chat'],
  ];

  for (const [journeyId, heading] of pages) {
    await gotoJourney(page, journey(journeyId));

    const globalHeader = page.locator('header').filter({
      has: page.getByRole('navigation', { name: 'Breadcrumb' }),
    });
    const pageHeading = page.getByRole('heading', {
      name: heading,
      exact: true,
      level: 1,
    });
    const pageHeader = pageHeading.locator(
      'xpath=ancestor::*[@aria-label][contains(@aria-label, "header") or @role="toolbar"][1]',
    );

    await expect(pageHeading).toBeVisible();
    await expect(pageHeader).toBeVisible();

    const [globalStyles, pageStyles] = await Promise.all(
      [globalHeader, pageHeader].map(locator =>
        locator.evaluate(element => {
          const styles = getComputedStyle(element);
          return {
            backgroundColor: styles.backgroundColor,
            borderBottomColor: styles.borderBottomColor,
            borderBottomWidth: styles.borderBottomWidth,
            boxShadow: styles.boxShadow,
            height: styles.height,
          };
        }),
      ),
    );

    expect(pageStyles).toEqual(globalStyles);
    await expect(
      pageHeading.locator(
        'xpath=ancestor::*[contains(@class, "bg-surface")][1]',
      ),
    ).toBeVisible();
  }
});

test('assistant session opens its populated message detail', async ({
  page,
}) => {
  await gotoJourney(page, journey('deployment.assistantSessions'));

  await page.getByRole('link', { name: /2301664620831572101/ }).click();

  await expect(page).toHaveURL(/\/sessions\/2301664620831572101$/);
  await expect(
    page.getByText('I need help resetting my password.'),
  ).toBeVisible();
  await expect(page.getByText('2 messages')).toBeVisible();

  const tabListBox = await page
    .getByRole('tablist', { name: 'Session detail tabs' })
    .boundingBox();
  const messageHeaderBox = await page
    .getByText('2 messages')
    .locator('..')
    .boundingBox();

  expect(tabListBox).not.toBeNull();
  expect(messageHeaderBox).not.toBeNull();
  expect(
    messageHeaderBox.y - (tabListBox.y + tabListBox.height),
  ).toBeLessThanOrEqual(2);

  const expectColumnsAligned = async table => {
    const headers = table.getByRole('columnheader');
    const firstRow = table.getByRole('row').nth(1);
    const cells = firstRow.getByRole('cell');
    await expect(headers).toHaveCount(3);
    await expect(cells).toHaveCount(3);

    for (let index = 0; index < 3; index += 1) {
      const headerBox = await headers.nth(index).boundingBox();
      const cellBox = await cells.nth(index).boundingBox();
      expect(headerBox).not.toBeNull();
      expect(cellBox).not.toBeNull();
      expect(Math.abs(headerBox.x - cellBox.x)).toBeLessThanOrEqual(1);
      expect(Math.abs(headerBox.width - cellBox.width)).toBeLessThanOrEqual(1);
    }
  };

  await page.getByRole('tab', { name: 'Context' }).click();
  const emptyContextHeading = page.getByRole('heading', {
    name: 'No context available',
  });
  const emptyContextDescription = page.getByText(
    'No knowledge context was retrieved for this session.',
  );
  await expect(emptyContextHeading).toBeVisible();
  await expect(emptyContextDescription).toBeVisible();
  expect(
    await emptyContextHeading.evaluate(element => {
      const style = window.getComputedStyle(element);
      return { fontSize: style.fontSize, lineHeight: style.lineHeight };
    }),
  ).toEqual({ fontSize: '20px', lineHeight: '28px' });
  expect(
    await emptyContextDescription.evaluate(element => {
      const style = window.getComputedStyle(element);
      return { fontSize: style.fontSize, lineHeight: style.lineHeight };
    }),
  ).toEqual({ fontSize: '14px', lineHeight: '20px' });

  await page.getByRole('tab', { name: 'Arguments' }).click();
  await expectColumnsAligned(
    page.getByRole('table', { name: 'Session arguments' }),
  );

  await page.getByRole('tab', { name: 'Metrics' }).click();
  await expectColumnsAligned(
    page.getByRole('table', { name: 'Session metrics' }),
  );

  const tabList = page.getByRole('tablist', {
    name: 'Session detail tabs',
  });
  const tabs = tabList.getByRole('tab');
  const tabStrip = tabList.locator('..');
  const tabStripBox = await tabStrip.boundingBox();
  const firstTabBox = await tabs.first().boundingBox();
  const lastTabBox = await tabs.last().boundingBox();
  expect(tabStripBox).not.toBeNull();
  expect(firstTabBox).not.toBeNull();
  expect(lastTabBox).not.toBeNull();
  expect(Math.abs(firstTabBox.x - tabStripBox.x)).toBeLessThanOrEqual(1);
  expect(
    tabStripBox.x + tabStripBox.width - (lastTabBox.x + lastTabBox.width),
  ).toBeGreaterThan(tabStripBox.width * 0.25);

  const tabStripBorders = await tabStrip.evaluate(element => {
    const style = window.getComputedStyle(element);
    return {
      bottom: style.borderBottomWidth,
      top: style.borderTopWidth,
    };
  });
  expect(tabStripBorders).toEqual({ bottom: '1px', top: '1px' });

  const metricsTable = page.getByRole('table', { name: 'Session metrics' });
  const firstMetricRowBox = await metricsTable
    .getByRole('row')
    .nth(1)
    .boundingBox();
  expect(firstMetricRowBox).not.toBeNull();
  expect(firstMetricRowBox.height).toBeGreaterThanOrEqual(48);
  expect(firstMetricRowBox.height).toBeLessThanOrEqual(48);
  const metricsPanel = metricsTable.locator(
    'xpath=ancestor::*[@role="tabpanel"]',
  );
  const wrappedLayout = await metricsTable.evaluate(table => {
    const panel = table.closest('[role="tabpanel"]');
    const valueCell = table.querySelector('tbody td:nth-child(2)');
    if (!panel || !valueCell) return null;

    valueCell.textContent = 'unbroken-metric-value-'.repeat(40);
    const panelBox = panel.getBoundingClientRect();
    const tableBox = table.getBoundingClientRect();
    const cellStyle = window.getComputedStyle(valueCell);

    return {
      cellOverflowWrap: cellStyle.overflowWrap,
      cellWhiteSpace: cellStyle.whiteSpace,
      panelClientWidth: panel.clientWidth,
      panelScrollWidth: panel.scrollWidth,
      panelWidth: panelBox.width,
      rowHeight: valueCell.closest('tr')?.getBoundingClientRect().height,
      tableWidth: tableBox.width,
    };
  });
  expect(wrappedLayout).not.toBeNull();
  expect(Math.abs(wrappedLayout.tableWidth - wrappedLayout.panelWidth)).toBe(0);
  expect(wrappedLayout.cellWhiteSpace).toBe('normal');
  expect(wrappedLayout.cellOverflowWrap).toBe('break-word');
  expect(wrappedLayout.rowHeight).toBeGreaterThan(48);
  expect(wrappedLayout.panelScrollWidth).toBeLessThanOrEqual(
    wrappedLayout.panelClientWidth,
  );

  const scrollState = await metricsPanel.evaluate(panel => {
    const body = panel.querySelector('tbody');
    const firstRow = body?.querySelector('tr');
    if (!body || !firstRow) return null;
    for (let index = 0; index < 100; index += 1) {
      body.append(firstRow.cloneNode(true));
    }
    panel.scrollTop = panel.scrollHeight;
    return {
      clientHeight: panel.clientHeight,
      scrollHeight: panel.scrollHeight,
      scrollTop: panel.scrollTop,
    };
  });
  expect(scrollState).not.toBeNull();
  expect(scrollState.scrollHeight).toBeGreaterThan(scrollState.clientHeight);
  expect(scrollState.scrollTop).toBeGreaterThan(0);
});

for (const colorMode of ['light', 'dark']) {
  test(`session detail recovers from an initial load error in ${colorMode} mode`, async ({
    page,
  }) => {
    let requestCount = 0;
    await page.route(ASSISTANT_CONVERSATION_DETAIL_PATH, route => {
      requestCount += 1;
      return requestCount === 1 ? route.abort('failed') : route.fallback();
    });
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await page.goto(
      '/deployment/assistant/2301664620831571968/sessions/2301664620831572101',
    );

    await expect(
      page.getByRole('heading', { name: 'Session unavailable' }),
    ).toBeVisible();
    await expect(page.getByText('503')).toBeVisible();
    await expect(
      page.getByText("We couldn't load this session. Try again in a moment."),
    ).toBeVisible();

    await page.getByRole('button', { name: 'Try again' }).click();

    await expect(
      page.getByText('I need help resetting my password.'),
    ).toBeVisible();
    expect(requestCount).toBe(2);
  });
}

test('dashboard assistant drilldowns reach assistant and failed-session detail', async ({
  page,
}) => {
  await gotoJourney(page, journey('dashboard.home'));

  await page
    .getByRole('table', { name: 'Top assistants' })
    .getByRole('link', { name: 'Support bot' })
    .click();
  await expect(
    page.getByRole('heading', { name: 'Assistant activity' }),
  ).toBeVisible();

  await gotoJourney(page, journey('dashboard.home'));
  await page
    .getByRole('table', { name: 'Recent failures' })
    .getByRole('link', { name: 'Support bot' })
    .click();
  await expect(
    page.getByText('I need help resetting my password.'),
  ).toBeVisible();
});

test('assistant transport failure renders the full unavailable state', async ({
  page,
}) => {
  await page.route(ASSISTANT_DETAIL_PATH, route => route.abort('failed'));
  await authenticatePage(page);
  await page.goto('/deployment/assistant/2301664620831571968/overview');

  await expect(
    page.getByRole('heading', { name: 'Assistant not available' }),
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'Back to assistants' }),
  ).toBeVisible();
  await expect(page.getByText('Assistant activity')).toHaveCount(0);
});

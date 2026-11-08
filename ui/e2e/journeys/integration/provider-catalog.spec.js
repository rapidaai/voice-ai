const { expect, gotoJourney, test } = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');

const googleJourney = routeJourneys.find(
  journey => journey.id === 'integration.googleSpeechService',
);
const azureJourney = routeJourneys.find(
  journey => journey.id === 'integration.azureSpeechService',
);

test('provider page scrolling collapses the descriptive header', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 600 });
  await gotoJourney(page, googleJourney);

  const pageScroll = page.locator('[data-page-scroll]');
  await expect(pageScroll).toBeVisible();
  expect(
    await pageScroll.evaluate(
      element => element.scrollHeight > element.clientHeight,
    ),
  ).toBe(true);

  await pageScroll.evaluate(element => element.scrollTo({ top: 240 }));
  const stickyHeader = page.locator('[data-page-header-sticky]');
  const toolbar = page.locator('.cds--table-toolbar');
  await expect(stickyHeader).toHaveAttribute('data-stuck', 'true');
  await expect
    .poll(async () => {
      const [headerBox, toolbarBox] = await Promise.all([
        stickyHeader.locator('[data-page-header-sticky-content]').boundingBox(),
        toolbar.boundingBox(),
      ]);
      if (!headerBox || !toolbarBox) return null;
      return Math.round(toolbarBox.y - (headerBox.y + headerBox.height));
    })
    .toBe(0);
});

test('provider catalog uses the platform table toolbar and pagination', async ({
  page,
}) => {
  const previewRequests = [];
  page.on('request', request => {
    if (/\.(mp3|wav)(\?|$)/i.test(request.url())) {
      previewRequests.push(request.url());
    }
  });
  await page.route('https://docs.cloud.google.com/**', route =>
    route.fulfill({ status: 204 }),
  );

  await gotoJourney(page, googleJourney);

  await expect(
    page.locator('[data-testid="voice-catalog-table"] tbody tr'),
  ).toHaveCount(10);

  await expect(
    page.getByRole('heading', { name: 'Voice catalogue' }),
  ).toHaveCount(0);

  const toolbar = page.getByRole('group', { name: 'data table toolbar' });
  await expect(
    toolbar.getByRole('button', { name: 'Add new credential' }),
  ).toBeVisible();

  for (const header of [
    'Voice',
    'Languages',
    'Persona',
    'Use cases',
    'Voice ID',
    'Preview',
  ]) {
    await expect(
      page.getByRole('columnheader', { name: header, exact: true }),
    ).toBeVisible();
  }

  const searchContainer = page.locator(
    '.cds--toolbar-search-container-expandable',
  );
  const search = toolbar.getByRole('searchbox', { name: 'Search voices' });
  await expect(searchContainer).toHaveClass(/-expandable/);
  await expect(searchContainer).not.toHaveClass(/-active/);
  await search.focus();
  await expect(searchContainer).toHaveClass(/-active/);
  await search.blur();
  await expect(searchContainer).not.toHaveClass(/-active/);

  const pageSizeSelect = page.getByLabel('Items per page:');
  await expect(pageSizeSelect).toHaveValue('10');
  await expect(pageSizeSelect.locator('option')).toHaveText([
    '10',
    '20',
    '50',
    '100',
  ]);

  const firstExpandButton = page
    .getByRole('button', { name: /^Expand .* details$/ })
    .first();
  await firstExpandButton.click();
  await expect(firstExpandButton).toHaveAttribute('aria-expanded', 'true');
  await expect(
    page.locator('[data-testid="voice-catalog-table"] tbody tr'),
  ).toHaveCount(11);

  await expect(page.locator('audio')).toHaveCount(0);
  expect(previewRequests).toEqual([]);

  await page
    .getByRole('button', { name: /^Preview / })
    .first()
    .click();
  await expect.poll(() => previewRequests.length).toBe(1);
  await expect(page.locator('audio')).toHaveCount(0);
});

test('provider catalog keeps every column visible with long voice metadata', async ({
  page,
}) => {
  await gotoJourney(page, azureJourney);

  const table = page.getByTestId('voice-catalog-table');
  const tableViewport = table.locator('..');
  const previewHeader = page.getByRole('columnheader', {
    name: 'Preview',
    exact: true,
  });

  await expect(previewHeader).toBeVisible();
  expect(
    await table.evaluate(
      element => element.scrollWidth <= element.parentElement.clientWidth,
    ),
  ).toBe(true);
  expect(
    await previewHeader.evaluate(
      element => element.scrollWidth <= element.clientWidth,
    ),
  ).toBe(true);
  await expect(tableViewport).toHaveJSProperty('scrollLeft', 0);
});

const { expect, gotoJourney, test } = require('../../fixtures');
const {
  fulfillGrpcWeb,
  protobufBoolean,
  protobufBytes,
  protobufUnsignedInteger,
} = require('../../mocks/grpc-web');
const { ENDPOINT_LOG_LIST_PATH } = require('../../mocks/clients/endpoint');
const { routeJourneys } = require('../../route-manifest');

const EMPTY_ENDPOINT_LOG_LIST_RESPONSE = Buffer.concat([
  protobufBoolean(2, true),
  protobufBytes(5, protobufUnsignedInteger(1, 1)),
]);

const overviewJourney = routeJourneys.find(
  journey => journey.id === 'deployment.endpointOverview',
);
const settingsJourney = routeJourneys.find(
  journey => journey.id === 'deployment.endpointSettings',
);

test('endpoint overview summarizes logs and opens the simplified playground', async ({
  page,
}) => {
  await gotoJourney(page, overviewJourney);

  const endpointNav = page.getByRole('navigation', {
    name: 'Endpoint actions',
  });
  await expect(endpointNav).toBeVisible();
  await expect(endpointNav.getByRole('link', { name: 'Overview' })).toHaveClass(
    /cds--side-nav__link--current/,
  );

  await expect(
    page.getByRole('heading', { name: 'Endpoint activity' }),
  ).toBeVisible();
  await expect(page.getByText('128').first()).toBeVisible();
  await expect(page.getByText('Request details')).toBeVisible();
  await expect(page.getByText('Last 30 days')).toBeVisible();
  await expect(page.getByText('Off')).toBeVisible();
  await expect(
    page.getByTestId('endpoint-request-performance-chart'),
  ).toBeVisible();
  await expect(
    page
      .getByTestId('endpoint-page-header')
      .getByText('Customer support analysis'),
  ).toBeVisible();
  const integrationAction = page.getByRole('button', {
    name: 'View integration',
  });
  const createVersionAction = page.getByRole('button', {
    name: 'Create new version',
  });
  await expect(integrationAction).toBeVisible();
  const integrationActionBounds = await integrationAction.boundingBox();
  const createVersionActionBounds = await createVersionAction.boundingBox();
  expect(integrationActionBounds).not.toBeNull();
  expect(createVersionActionBounds).not.toBeNull();
  expect(
    Math.abs(
      integrationActionBounds.x +
        integrationActionBounds.width -
        createVersionActionBounds.x,
    ),
  ).toBeLessThanOrEqual(1);
  expect(
    Math.abs(integrationActionBounds.y - createVersionActionBounds.y),
  ).toBeLessThanOrEqual(1);
  expect(
    Math.abs(integrationActionBounds.height - createVersionActionBounds.height),
  ).toBeLessThanOrEqual(1);

  await integrationAction.click();
  const integrationDialog = page.getByRole('dialog', {
    name: 'Customer support analysis',
  });
  await expect(integrationDialog).toBeVisible();
  const languageTabs = integrationDialog.getByRole('tablist', {
    name: 'Endpoint integration languages',
  });
  await expect(languageTabs).toBeVisible();
  const pythonTabBounds = await integrationDialog
    .getByRole('tab', { name: 'Python' })
    .boundingBox();
  const typeScriptTabBounds = await integrationDialog
    .getByRole('tab', { name: 'TypeScript' })
    .boundingBox();
  const golangTabBounds = await integrationDialog
    .getByRole('tab', { name: 'Golang' })
    .boundingBox();
  expect(pythonTabBounds).not.toBeNull();
  expect(typeScriptTabBounds).not.toBeNull();
  expect(golangTabBounds).not.toBeNull();
  expect(
    Math.max(
      pythonTabBounds.width,
      typeScriptTabBounds.width,
      golangTabBounds.width,
    ) -
      Math.min(
        pythonTabBounds.width,
        typeScriptTabBounds.width,
        golangTabBounds.width,
      ),
  ).toBeLessThanOrEqual(1);
  expect(
    Math.max(pythonTabBounds.y, typeScriptTabBounds.y, golangTabBounds.y) -
      Math.min(pythonTabBounds.y, typeScriptTabBounds.y, golangTabBounds.y),
  ).toBeLessThanOrEqual(1);
  expect(
    Math.max(
      pythonTabBounds.height,
      typeScriptTabBounds.height,
      golangTabBounds.height,
    ) -
      Math.min(
        pythonTabBounds.height,
        typeScriptTabBounds.height,
        golangTabBounds.height,
      ),
  ).toBeLessThanOrEqual(1);
  await integrationDialog.getByRole('tab', { name: 'TypeScript' }).click();
  await expect(integrationDialog.getByText(/npm install/)).toBeVisible();
  await integrationDialog.getByRole('button', { name: 'Close' }).click();
  await expect(integrationDialog).toBeHidden();
  const overviewHeaderBounds = await page
    .getByTestId('endpoint-page-header')
    .boundingBox();
  await expect(page.locator('form')).toHaveCount(0);

  await endpointNav.getByRole('link', { name: 'Open playground' }).click();

  await expect(page).toHaveURL(/\/playground$/);
  await expect(page.getByRole('heading', { name: 'Playground' })).toBeVisible();
  await expect(page.locator('form')).toBeVisible();
  await expect(page.getByTestId('endpoint-playground-console')).toBeVisible();
  await expect(page.getByText('No arguments required')).toBeVisible();
  await expect(
    page.getByText('Output will appear after execution.'),
  ).toBeVisible();

  const playgroundHeaderBounds = await page
    .getByTestId('endpoint-page-header')
    .boundingBox();

  expect(overviewHeaderBounds).not.toBeNull();
  expect(playgroundHeaderBounds).not.toBeNull();
  expect(playgroundHeaderBounds.x).toBe(overviewHeaderBounds.x);
  expect(playgroundHeaderBounds.width).toBe(overviewHeaderBounds.width);
});

test('endpoint settings follows the assistant general settings pattern', async ({
  page,
}) => {
  await gotoJourney(page, settingsJourney);

  const endpointNav = page.getByRole('navigation', {
    name: 'Endpoint actions',
  });
  await expect(endpointNav.getByRole('link', { name: 'General' })).toHaveClass(
    /cds--side-nav__link--current/,
  );
  await expect(
    page.getByRole('heading', { name: 'General Settings' }),
  ).toBeVisible();
  await expect(page.getByText('Identity')).toBeVisible();
  await expect(page.getByText('General Information')).toBeVisible();
  await expect(page.locator('#endpoint-name')).toHaveValue(
    'Customer support analysis',
  );
  await expect(
    page.getByRole('button', { name: 'Save changes' }),
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'View integration' }),
  ).toHaveCount(0);
});

test('endpoint overview aligns the empty graph state with its header', async ({
  page,
}) => {
  await page.route(ENDPOINT_LOG_LIST_PATH, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWeb(route, EMPTY_ENDPOINT_LOG_LIST_RESPONSE);
  });
  await gotoJourney(page, overviewJourney);

  const graphTitle = page.getByRole('heading', { name: 'Request latency' });
  const emptyStateTitle = page.getByRole('heading', {
    name: 'No requests yet',
  });
  const emptyStateDescription = page.getByText(
    /Run this endpoint in the playground/,
  );
  const emptyStateAction = page.getByRole('button', {
    name: 'Open playground',
  });
  await expect(emptyStateTitle).toBeVisible();

  const graphTitleBounds = await graphTitle.boundingBox();
  const emptyStateTitleBounds = await emptyStateTitle.boundingBox();
  const emptyStateDescriptionBounds = await emptyStateDescription.boundingBox();
  const emptyStateActionBounds = await emptyStateAction.boundingBox();
  expect(graphTitleBounds).not.toBeNull();
  expect(emptyStateTitleBounds).not.toBeNull();
  expect(emptyStateDescriptionBounds).not.toBeNull();
  expect(emptyStateActionBounds).not.toBeNull();
  expect(
    Math.abs(emptyStateTitleBounds.x - graphTitleBounds.x),
  ).toBeLessThanOrEqual(1);
  expect(
    emptyStateActionBounds.y -
      (emptyStateDescriptionBounds.y + emptyStateDescriptionBounds.height),
  ).toBeGreaterThanOrEqual(16);
});

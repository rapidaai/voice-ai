const {
  checkA11y,
  expect,
  gotoJourney,
  setColorMode,
  test,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');

const journey = routeJourneys.find(
  routeJourney => routeJourney.id === 'integration.projectCredential',
);

async function openRoleSelector(page) {
  await gotoJourney(page, journey);
  await page.getByRole('button', { name: 'Create API key' }).click();
  const dialog = page.getByRole('dialog');
  await expect(
    dialog.getByRole('heading', { name: 'Create API key' }),
  ).toBeVisible();
  await expect(
    dialog.locator('.cds--modal-header').getByRole('button', { name: 'Close' }),
  ).toBeVisible();
  return dialog;
}

async function selectProjectRole(dialog, roleName, roleKey) {
  await dialog.getByRole('combobox', { name: 'Project Role' }).fill('Observer');
  await dialog
    .getByRole('option', { name: `${roleName} (${roleKey})`, exact: true })
    .click();
}

test('project credential creation requires a searchable role assignment', async ({
  page,
}) => {
  const dialog = await openRoleSelector(page);
  const createButton = dialog.getByRole('button', {
    name: 'Create API key',
  });
  await expect(createButton).toBeDisabled();

  await dialog
    .getByRole('button', { name: 'Application name information' })
    .click();
  await expect(
    dialog.getByText(
      'Use a name that identifies the application, service, or integration that will use this key.',
    ),
  ).toBeVisible();
  await dialog
    .getByRole('textbox', { name: 'Application Name', exact: true })
    .fill('Web widget');
  await selectProjectRole(dialog, 'Observer Client', 'observer_client');
  await expect(dialog.getByText('Observer Client permissions')).toHaveCount(0);
  await expect(dialog.getByText('project.read', { exact: true })).toHaveCount(
    0,
  );
  await expect(createButton).toBeEnabled();
  await checkA11y(page, journey);

  await createButton.click();
  await expect(dialog.getByText('API key created')).toBeVisible();
  await expect(dialog.getByLabel('API Key', { exact: true })).toHaveValue(
    'rpx_e2e_project_credential',
  );
  await dialog
    .getByRole('button', { name: 'API key security information' })
    .click();
  await expect(
    dialog.getByText(
      'Store this key in an approved secrets manager. For security, it cannot be retrieved after this dialog is closed.',
    ),
  ).toBeVisible();
  await checkA11y(page, journey);
  await dialog.getByRole('button', { name: 'Show key' }).click();
  await expect(dialog.getByLabel('API Key', { exact: true })).toHaveAttribute(
    'type',
    'text',
  );
  await dialog
    .locator('.cds--modal-header')
    .getByRole('button', { name: 'Close' })
    .click();
  await expect(dialog).toBeHidden();
  await expect(
    page.getByRole('cell', { name: 'Web widget', exact: true }),
  ).toBeVisible();
});

test('project API key inventory masks keys and revokes by confirmed name', async ({
  page,
}) => {
  await gotoJourney(page, journey);
  await expect(page.getByRole('columnheader')).toHaveText([
    'Name',
    'API key',
    'Access role',
    'Created',
    'Actions',
  ]);
  await expect(page.getByText('Runtime Invoker')).toBeVisible();
  await expect(page.getByText('rpx_e2e_web_widget')).toBeHidden();
  await expect(
    page.getByText(/Use a separate API key for each application/),
  ).toBeHidden();

  const webWidgetName = page.getByRole('cell', {
    name: 'Web widget',
    exact: true,
  });
  await expect(
    page.getByRole('button', { name: 'Web widget', exact: true }),
  ).toHaveCount(0);
  const webWidgetRow = page.getByRole('row').filter({ has: webWidgetName });
  const cells = webWidgetRow.getByRole('cell');
  await expect(
    cells.nth(1).getByRole('button', { name: 'Copy Web widget API key' }),
  ).toBeVisible();
  await expect(
    cells.nth(1).getByRole('button', { name: 'Show Web widget API key' }),
  ).toBeVisible();
  await expect(cells.nth(4).getByRole('button')).toHaveCount(1);
  await expect(
    cells.nth(4).getByRole('button', { name: 'Revoke Web widget' }),
  ).toBeVisible();

  const apiKeyCellBefore = await cells.nth(1).boundingBox();
  const accessRoleCellBefore = await cells.nth(2).boundingBox();
  await cells
    .nth(1)
    .getByRole('button', { name: 'Show Web widget API key' })
    .click();
  await expect(page.getByText('rpx_e2e_web_widget')).toBeVisible();
  const apiKeyCellAfter = await cells.nth(1).boundingBox();
  const accessRoleCellAfter = await cells.nth(2).boundingBox();
  expect(apiKeyCellAfter).toEqual(apiKeyCellBefore);
  expect(accessRoleCellAfter).toEqual(accessRoleCellBefore);
  await cells
    .nth(1)
    .getByRole('button', { name: 'Hide Web widget API key' })
    .click();
  await expect(page.getByText('rpx_e2e_web_widget')).toBeHidden();

  await page.getByRole('button', { name: 'Revoke Web widget' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText(
    'Applications using "Web widget" will lose access immediately.',
  );
  await dialog.getByLabel('Type "Web widget" to confirm').fill('Web widget');
  await dialog.getByRole('button', { name: 'Revoke API key' }).click();
  await expect(
    page.getByRole('cell', { name: 'Web widget', exact: true }),
  ).toBeHidden();
  await expect(
    page.getByRole('cell', { name: 'Phone assistant', exact: true }),
  ).toBeVisible();
  await checkA11y(page, journey);
});

for (const colorMode of ['light', 'dark']) {
  test(`project credential role selector screenshot in ${colorMode} mode`, async ({
    page,
  }) => {
    await setColorMode(page, colorMode);
    const dialog = await openRoleSelector(page);
    await dialog
      .getByRole('textbox', { name: 'Application Name', exact: true })
      .fill('Web widget');
    await selectProjectRole(dialog, 'Observer Client', 'observer_client');
    await expect(page.locator('html')).toHaveAttribute(
      'data-color-mode',
      colorMode,
    );
    await page.evaluate(async () => {
      await document.fonts.ready;
      await Promise.all(
        document
          .getAnimations()
          .filter(animation =>
            Number.isFinite(animation.effect?.getComputedTiming().endTime),
          )
          .map(animation => animation.finished.catch(() => {})),
      );
    });
    await expect(page).toHaveScreenshot(
      ['integration', `project-credential-role-selector-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });

  test(`generated project credential screenshot in ${colorMode} mode`, async ({
    page,
  }) => {
    await setColorMode(page, colorMode);
    const dialog = await openRoleSelector(page);
    await dialog
      .getByRole('textbox', { name: 'Application Name', exact: true })
      .fill('Web widget');
    await selectProjectRole(dialog, 'Observer Client', 'observer_client');
    await dialog.getByRole('button', { name: 'Create API key' }).click();
    await expect(dialog.getByText('API key created')).toBeVisible();
    await expect(page.locator('html')).toHaveAttribute(
      'data-color-mode',
      colorMode,
    );
    await page.evaluate(async () => {
      await document.fonts.ready;
      await Promise.all(
        document
          .getAnimations()
          .filter(animation =>
            Number.isFinite(animation.effect?.getComputedTiming().endTime),
          )
          .map(animation => animation.finished.catch(() => {})),
      );
    });
    await expect(page).toHaveScreenshot(
      ['integration', `project-credential-generated-key-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });
}

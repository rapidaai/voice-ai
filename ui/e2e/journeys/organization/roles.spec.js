const {
  checkA11y,
  expect,
  gotoJourney,
  mockRoleListFailureAfterInitialLoad,
  mockRoleNotFound,
  setColorMode,
  test,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');

const journey = id =>
  routeJourneys.find(routeJourney => routeJourney.id === `organization.${id}`);

test('organization roles expose inline actions and complete read-only details', async ({
  page,
}) => {
  await gotoJourney(page, journey('roles'));

  await expect(
    page.getByText('Support manager', { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'View Support manager' }),
  ).toBeEnabled();
  await expect(
    page.getByRole('button', { name: 'Edit Support manager' }),
  ).toBeEnabled();
  await expect(
    page.getByRole('button', { name: 'Delete Support manager' }),
  ).toBeEnabled();
  await expect(
    page.getByRole('button', { name: 'View Project viewer' }),
  ).toBeEnabled();
  await expect(
    page.getByRole('button', { name: 'Edit Project viewer' }),
  ).toBeDisabled();
  await expect(
    page.getByRole('button', { name: 'Delete Project viewer' }),
  ).toBeDisabled();

  const catalogUrl = page.url();
  await page.getByRole('button', { name: 'View Support manager' }).click();
  const roleDialog = page.getByRole('dialog', { name: 'Support manager' });
  await expect(roleDialog).toBeVisible();
  await expect(
    roleDialog.getByRole('heading', { name: 'Support manager' }),
  ).toBeVisible();
  await expect(
    roleDialog.getByText('support_manager', { exact: true }),
  ).toBeVisible();
  await expect(
    roleDialog.getByText('Ada Lovelace', { exact: true }),
  ).toBeVisible();
  await expect(
    roleDialog.getByText('Grace Hopper', { exact: true }),
  ).toBeVisible();

  await roleDialog.getByRole('tab', { name: 'Permissions' }).click();
  await expect(
    roleDialog.getByText('Project Read', { exact: true }),
  ).toBeVisible();
  await expect(
    roleDialog.getByText('project.read', { exact: true }),
  ).toBeVisible();
  await expect(
    roleDialog.getByText('Read project configuration and metadata.', {
      exact: true,
    }),
  ).toBeVisible();
  expect(page.url()).toBe(catalogUrl);
  await checkA11y(page, journey('roles'));

  await roleDialog.getByRole('button', { name: 'Close' }).click();
  await expect(
    page.getByRole('button', { name: 'View Support manager' }),
  ).toBeFocused();
});

test('organization roles show terminal not-found without Retry', async ({
  page,
}) => {
  await mockRoleNotFound(page);
  await gotoJourney(page, journey('roles'));
  await page.getByRole('button', { name: 'View Support manager' }).click();

  await expect(
    page.getByText(/This role is no longer available\./),
  ).toBeVisible();
  await expect(page.getByRole('button', { name: 'Retry' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Close' })).toBeVisible();
});

test('organization roles keep custom Edit in a modal', async ({ page }) => {
  await gotoJourney(page, journey('roles'));
  const catalogUrl = page.url();
  await page.getByRole('button', { name: 'Edit Support manager' }).click();

  await expect(page).toHaveURL(catalogUrl);
  const editDialog = page.getByRole('dialog', { name: 'Edit role' });
  await expect(editDialog).toBeVisible();
  await expect(editDialog.getByLabel('Name', { exact: true })).toHaveValue(
    'Support manager',
  );
  await expect(editDialog.getByLabel('Description')).toHaveValue(
    'Manages support workflows across assigned projects.',
  );
  await expect(editDialog.getByRole('tab')).toHaveCount(0);
  await expect(editDialog.getByRole('checkbox')).toHaveCount(0);
  await expect(
    editDialog.getByRole('button', { name: 'Remove permissions' }),
  ).toHaveCount(0);
  await checkA11y(page, journey('roles'));
});

test('organization role actions update and delete a custom role', async ({
  page,
}) => {
  await gotoJourney(page, journey('roles'));
  const catalogUrl = page.url();

  await page.getByRole('button', { name: 'Edit Support manager' }).click();
  const editDialog = page.getByRole('dialog', { name: 'Edit role' });
  await expect(editDialog).toBeVisible();
  await editDialog.getByLabel('Name', { exact: true }).fill('Support lead');
  await editDialog.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByText('Support lead', { exact: true })).toBeVisible();
  await expect(page).toHaveURL(catalogUrl);

  await page.getByRole('button', { name: 'Delete Support lead' }).click();
  const deleteDialog = page.getByRole('dialog', {
    name: 'Delete role',
  });
  await expect(deleteDialog).toContainText(
    'Deleting a role removes it from the active roles. Its key cannot be reused, and roles assigned to users cannot be deleted.',
  );
  await checkA11y(page, journey('roles'));
  await expect(deleteDialog.getByPlaceholder('Support lead')).toBeVisible();
  await deleteDialog.getByPlaceholder('Support lead').fill('Support lead');
  await deleteDialog.getByRole('button', { name: 'Delete role' }).click();

  await expect(page).toHaveURL(catalogUrl);
  await expect(page.getByText('Support lead', { exact: true })).toHaveCount(0);
  await expect(page.getByText('Project viewer', { exact: true })).toBeVisible();
});

test('organization role creation stays on the catalog URL', async ({
  page,
}) => {
  await gotoJourney(page, journey('roles'));
  const catalogUrl = page.url();
  await page.getByRole('button', { name: 'Create role', exact: true }).click();
  expect(page.url()).toBe(catalogUrl);
  await expect(
    page.getByRole('heading', { name: 'Create Role' }),
  ).toBeVisible();
  const tearsheet = page.locator('.role-create-tearsheet');
  await expect(tearsheet).toBeVisible();
  await expect(tearsheet).toHaveClass(/c4p--tearsheet__next--wide/);
  await expect(
    tearsheet.getByText(
      'Create a custom role for your organization and choose the permissions it grants.',
    ),
  ).toBeVisible();
  await expect(
    tearsheet.locator('.c4p--tearsheet__next__header'),
  ).toBeVisible();
  await expect(
    tearsheet.getByRole('button', { name: 'Close create role' }),
  ).toBeVisible();
  await expect(
    tearsheet.locator('.c4p--tearsheet__next__influencer'),
  ).toBeVisible();
  await expect(
    tearsheet.locator('.c4p--tearsheet__next__influencer'),
  ).toHaveCSS('padding-top', '24px');
  await expect(
    tearsheet.locator(
      '.c4p--tearsheet__next__influencer .cds--progress--vertical',
    ),
  ).toBeVisible();
  await expect(
    tearsheet.getByText('Define access type and scope'),
  ).toBeVisible();
  await expect(
    tearsheet.getByLabel('Access type', { exact: true }),
  ).toHaveValue('ROLE_SUBJECT_USER');
  await tearsheet
    .getByRole('button', { name: 'Access type information' })
    .click();
  await expect(
    tearsheet.getByText(/For signed-in users.*organization or project scope/),
  ).toBeVisible();
  await expect(
    tearsheet.getByText(/For API integrations.*Project scope only/),
  ).toBeVisible();
  await checkA11y(page, journey('roles'));
  await page.keyboard.press('Escape');
  const footer = tearsheet.locator('.c4p--tearsheet__next__footer');
  await expect(footer).toBeVisible();
  const footerActionSet = tearsheet.locator('.c4p--action-set');
  await expect(footerActionSet).toHaveCSS('height', '64px');
  for (const button of await footerActionSet.getByRole('button').all()) {
    await expect(button).toHaveCSS('height', '64px');
  }
  await expect(tearsheet.locator('.cds--modal-scroll-content')).toHaveCount(0);
  await checkA11y(page, journey('roles'));

  await page.getByLabel('Role key', { exact: true }).fill('Invalid Key');
  await tearsheet.getByRole('button', { name: 'Next', exact: true }).click();
  const errorNotification = tearsheet.locator(
    '.cds--inline-notification--error',
  );
  await expect(
    footer.locator('.cds--inline-notification--error'),
  ).toBeVisible();
  await expect(
    errorNotification.getByText('Error', { exact: true }),
  ).toBeVisible();
  await expect(errorNotification).toContainText(
    'Key must start with a lowercase letter',
  );
  await checkA11y(page, journey('roles'));

  await tearsheet.getByRole('button', { name: 'Close create role' }).click();
  expect(page.url()).toBe(catalogUrl);
  await expect(page.getByRole('heading', { name: 'Create Role' })).toHaveCount(
    0,
  );
  await expect(
    page.getByRole('button', { name: 'Create role', exact: true }),
  ).toBeFocused();

  await page.getByRole('button', { name: 'Create role', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: 'Create Role' }),
  ).toBeVisible();
  await page.reload();
  expect(page.url()).toBe(catalogUrl);
  await expect(page.getByRole('heading', { name: 'Create Role' })).toHaveCount(
    0,
  );
});

test('created-role handoff opens details when catalog refresh fails', async ({
  page,
}) => {
  await mockRoleListFailureAfterInitialLoad(page);
  await gotoJourney(page, journey('roles'));
  await expect(
    page.getByText('Support manager', { exact: true }),
  ).toBeVisible();
  await page.getByRole('button', { name: 'Create role', exact: true }).click();

  await page.getByLabel('Role name').fill('Incident manager');
  await page.getByLabel('Role key', { exact: true }).fill('incident_manager');
  await page.getByRole('button', { name: 'Next', exact: true }).click();
  const createRoleDialog = page.getByRole('dialog', { name: 'Create Role' });
  const mainContent = createRoleDialog.locator(
    '.c4p--tearsheet__next__main-content',
  );
  const permissionSelector = createRoleDialog.locator('.c4p--add-select__next');
  await expect(mainContent).toHaveClass(/c4p--tearsheet__next__flush/);
  const [mainContentBox, permissionSelectorBox] = await Promise.all([
    mainContent.boundingBox(),
    permissionSelector.boundingBox(),
  ]);
  expect(mainContentBox).not.toBeNull();
  expect(permissionSelectorBox).not.toBeNull();
  expect(Math.abs(permissionSelectorBox.x - mainContentBox.x)).toBeLessThan(1);
  expect(Math.abs(permissionSelectorBox.y - mainContentBox.y)).toBeLessThan(1);
  expect(
    Math.abs(permissionSelectorBox.width - mainContentBox.width),
  ).toBeLessThan(1);
  expect(
    Math.abs(permissionSelectorBox.height - mainContentBox.height),
  ).toBeLessThan(1);
  await checkA11y(page, journey('roles'));
  await expect(
    page.getByRole('dialog', { name: 'Select Permissions' }),
  ).toHaveCount(0);
  const permissionSearch = page.getByPlaceholder('Search permissions');
  await permissionSearch.fill('Project Read');
  const permissionRow = page.getByRole('row', {
    name: /Project Read/,
  });
  await permissionRow.getByRole('checkbox').press('Space');
  await expect(permissionRow).toHaveAttribute('aria-selected', 'true');
  await expect(
    page.locator('.c4p--add-select__next__selection-summary'),
  ).toContainText('Project Read');
  await checkA11y(page, journey('roles'));
  await createRoleDialog
    .getByRole('button', { name: 'Create role', exact: true })
    .click();

  await expect(page).toHaveURL(/\/organization\/roles$/);
  await expect(
    page.getByRole('heading', { name: 'Incident manager' }),
  ).toBeVisible();
  await expect(
    page.getByText('incident_manager', { exact: true }),
  ).toBeVisible();
});

for (const colorMode of ['light', 'dark']) {
  test(`organization.roles edit modal screenshot in ${colorMode} mode`, async ({
    page,
  }) => {
    await setColorMode(page, colorMode);
    await gotoJourney(page, journey('roles'));
    await page.getByRole('button', { name: 'Edit Support manager' }).click();
    const editDialog = page.getByRole('dialog', { name: 'Edit role' });
    await expect(editDialog.getByLabel('Name', { exact: true })).toBeVisible();
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
      ['organization', `roles-edit-modal-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });

  test(`organization.roles details panel screenshot in ${colorMode} mode`, async ({
    page,
  }) => {
    await setColorMode(page, colorMode);
    await gotoJourney(page, journey('roles'));
    await page.getByRole('button', { name: 'View Support manager' }).click();
    await expect(
      page.getByRole('heading', { name: 'Support manager' }),
    ).toBeVisible();
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
      ['organization', `roles-details-panel-${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });
}

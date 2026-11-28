const { authenticatePage, expect, test } = require('../../fixtures');
const { mockLLMActivity } = require('../../mocks/clients/activity');

test('activity log code editor fills the selected tab panel', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1024, height: 900 });
  await authenticatePage(page);
  await mockLLMActivity(page);
  await page.goto('/logs');

  await page.getByRole('button', { name: 'View detail' }).click();
  const dialog = page.getByRole('dialog', { name: '9001' });
  await expect(dialog).toBeVisible();
  await dialog.getByRole('tab', { name: 'Request' }).click();

  const tabPanel = dialog.getByRole('tabpanel');
  const editor = tabPanel.locator('.monaco-editor');
  await expect(editor).toBeVisible();

  const panelWidth = await tabPanel.evaluate(element => element.clientWidth);
  const editorWidth = await editor.evaluate(
    element => element.getBoundingClientRect().width,
  );

  expect(Math.abs(panelWidth - editorWidth)).toBeLessThanOrEqual(2);
});

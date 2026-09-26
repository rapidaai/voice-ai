import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';

test.describe('sign-in browser quality', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/auth/signin');
    await expect(page.getByRole('heading', { name: 'Signin' })).toBeVisible();
  });

  test('has no serious accessibility violations', async ({ page }) => {
    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    const blocking = results.violations.filter(violation =>
      ['critical', 'serious'].includes(violation.impact || ''),
    );

    expect(blocking).toEqual([]);
  });

  test('matches the stable authentication card', async ({ page }) => {
    await expect(page.getByRole('main').last()).toHaveScreenshot(
      'sign-in-card.png',
    );
  });
});

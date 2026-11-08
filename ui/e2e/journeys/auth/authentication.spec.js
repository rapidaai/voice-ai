const {
  AUTHENTICATION_PATHS,
  checkA11y,
  expect,
  gotoJourney,
  mockAuthenticationOperation,
  setColorMode,
  test,
} = require('../../fixtures');
const { routeJourneys } = require('../../route-manifest');
const themeContract = require('../../../src/theme/theme-contract.json');

const consumedCarbonProperties = [
  ...Object.values(themeContract.carbonProperties.brandOverrides).flat(),
  ...themeContract.carbonProperties.presetOwned,
].sort();

const authJourney = id =>
  routeJourneys.find(journey => journey.id === `auth.${id}`);

const colorChannels = color =>
  color
    .match(/\d+(?:\.\d+)?/g)
    .slice(0, 3)
    .map(Number);

const relativeLuminance = color => {
  const channels = colorChannels(color).map(channel => {
    const value = channel / 255;
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
};

const contrastRatio = (foreground, background) => {
  const light = Math.max(
    relativeLuminance(foreground),
    relativeLuminance(background),
  );
  const dark = Math.min(
    relativeLuminance(foreground),
    relativeLuminance(background),
  );
  return (light + 0.05) / (dark + 0.05);
};

async function submitSignIn(page) {
  await page.getByLabel('Email address').fill('user@example.test');
  await page.locator('#signin-password').fill('correct horse battery staple');
  await page.getByRole('button', { name: 'Log in', exact: true }).click();
}

async function submitSignUp(page) {
  await page.getByLabel('Full name', { exact: true }).fill('E2E User');
  await page.getByLabel('Email address').fill('user@example.test');
  await page.locator('#signup-password').fill('correct horse battery staple');
  await page.getByRole('button', { name: 'Create account' }).click();
}

test('authentication shell centers a borderless account surface between its Carbon bands', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await gotoJourney(page, authJourney('signin'));

  await expect(page.getByRole('banner')).toBeVisible();
  const footer = page.getByRole('contentinfo');
  await expect(footer).toBeVisible();
  await expect(footer.getByRole('link', { name: 'Support' })).toBeVisible();
  await expect(
    page.getByRole('main', { name: 'Authentication' }),
  ).toBeVisible();

  const main = page.getByRole('main', { name: 'Authentication' });
  const accountAccess = page.getByRole('region', { name: 'Account access' });
  const signInPanel = page.getByTestId('signin-panel');
  const emailInput = accountAccess.getByLabel('Email address');
  const forgotPassword = accountAccess.getByRole('link', {
    name: 'Forgot password?',
  });
  const forgotPasswordRow = forgotPassword.locator('..');
  const submitButton = accountAccess.getByRole('button', {
    name: 'Log in',
    exact: true,
  });
  const googleButton = accountAccess.getByRole('button', {
    name: 'Log in with Google',
  });

  await expect(accountAccess).toBeVisible();
  await expect(page.getByTestId('auth-center-box')).toHaveCount(0);
  await expect(page.getByTestId('auth-header-grid')).toHaveCount(0);
  await expect(page.getByTestId('auth-footer-grid')).toHaveCount(0);

  const headerBounds = await page.getByRole('banner').boundingBox();
  const mainBounds = await main.boundingBox();
  const accountAccessBounds = await accountAccess.boundingBox();
  const emailInputBounds = await emailInput.boundingBox();
  const forgotPasswordRowBounds = await forgotPasswordRow.boundingBox();
  const submitButtonBounds = await submitButton.boundingBox();
  const googleButtonBounds = await googleButton.boundingBox();

  expect(headerBounds).not.toBeNull();
  expect(mainBounds).not.toBeNull();
  expect(accountAccessBounds).not.toBeNull();
  expect(emailInputBounds).not.toBeNull();
  expect(forgotPasswordRowBounds).not.toBeNull();
  expect(submitButtonBounds).not.toBeNull();
  expect(googleButtonBounds).not.toBeNull();
  expect(headerBounds.height).toBe(48);
  expect(mainBounds.width).toBe(1280);
  expect(mainBounds.y).toBe(headerBounds.y + headerBounds.height);
  expect(accountAccessBounds.width).toBe(592);
  expect(accountAccessBounds.x + accountAccessBounds.width / 2).toBe(640);
  expect(mainBounds.x + mainBounds.width / 2).toBe(640);
  expect(emailInputBounds.x).toBe(submitButtonBounds.x);
  expect(emailInputBounds.width).toBe(submitButtonBounds.width);
  expect(googleButtonBounds.x).toBe(submitButtonBounds.x);
  expect(googleButtonBounds.width).toBe(submitButtonBounds.width);
  expect(
    submitButtonBounds.y -
      (forgotPasswordRowBounds.y + forgotPasswordRowBounds.height),
  ).toBe(24);
  const surfaceStyles = await Promise.all(
    [main, signInPanel].map(locator =>
      locator.evaluate(element => {
        const style = getComputedStyle(element);
        return {
          backgroundColor: style.backgroundColor,
          borderWidth: style.borderWidth,
        };
      }),
    ),
  );
  expect(surfaceStyles[0].backgroundColor).not.toBe(
    surfaceStyles[1].backgroundColor,
  );
  expect(surfaceStyles[1].borderWidth).toBe('0px');

  const primaryTokenColors = await page.evaluate(() => {
    const probe = document.createElement('span');
    probe.style.backgroundColor = 'var(--cds-button-primary)';
    document.body.appendChild(probe);
    const primary = getComputedStyle(probe).backgroundColor;
    probe.style.backgroundColor = 'var(--cds-button-primary-hover)';
    const hover = getComputedStyle(probe).backgroundColor;
    probe.remove();
    return { primary, hover };
  });
  const primaryButtonStyles = await submitButton.evaluate(element => {
    const style = getComputedStyle(element);
    const labelStyle = getComputedStyle(element.querySelector('span'));
    return {
      backgroundColor: style.backgroundColor,
      color: style.color,
      fontFamily: labelStyle.fontFamily,
      fontSize: labelStyle.fontSize,
      fontWeight: labelStyle.fontWeight,
      letterSpacing: labelStyle.letterSpacing,
      lineHeight: labelStyle.lineHeight,
    };
  });
  expect(primaryButtonStyles).toMatchObject({
    backgroundColor: primaryTokenColors.primary,
    color: 'rgb(255, 255, 255)',
    fontSize: '14px',
    fontWeight: '400',
    letterSpacing: '0.16px',
    lineHeight: '18px',
  });
  expect(primaryButtonStyles.fontFamily).toContain('IBM Plex Sans');
  await submitButton.hover();
  await expect
    .poll(() =>
      submitButton.evaluate(element => {
        const style = getComputedStyle(element);
        return {
          backgroundColor: style.backgroundColor,
          color: style.color,
        };
      }),
    )
    .toEqual({
      backgroundColor: primaryTokenColors.hover,
      color: 'rgb(255, 255, 255)',
    });
  await expect(page.getByTestId('auth-form-rail')).toHaveCount(0);
  await expect(page.getByTestId('auth-illustration')).toHaveCount(0);

  await page.setViewportSize({ width: 1600, height: 900 });
  const wideAccountAccessBounds = await accountAccess.boundingBox();
  expect(wideAccountAccessBounds.x + wideAccountAccessBounds.width / 2).toBe(
    800,
  );
});

test('authentication shell keeps account access aligned at medium widths', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1024, height: 900 });
  await gotoJourney(page, authJourney('signin'));

  const main = page.getByRole('main', { name: 'Authentication' });
  const accountAccess = page.getByRole('region', { name: 'Account access' });

  const mainBounds = await main.boundingBox();
  const accountAccessBounds = await accountAccess.boundingBox();

  expect(mainBounds).not.toBeNull();
  expect(accountAccessBounds).not.toBeNull();
  expect(mainBounds.width).toBe(1024);
  expect(accountAccessBounds.width).toBe(592);
  expect(accountAccessBounds.x).toBe(216);
  await expect(page.getByTestId('auth-center-box')).toHaveCount(0);
  await expect(page.getByTestId('auth-form-rail')).toHaveCount(0);
  await expect(page.getByTestId('auth-illustration')).toHaveCount(0);
});

test('version 2 dark primary action uses the configured brand pair', async ({
  page,
}) => {
  await setColorMode(page, 'dark');
  await gotoJourney(page, authJourney('signin'));

  const buttonPrimary = await page.evaluate(() => {
    const probe = document.createElement('span');
    document.body.appendChild(probe);

    probe.style.backgroundColor = 'var(--cds-button-primary)';
    const color = getComputedStyle(probe).backgroundColor;
    probe.remove();

    return color;
  });
  const buttonStyles = await page
    .getByRole('button', { name: 'Log in', exact: true })
    .evaluate(element => {
      const style = getComputedStyle(element);
      return {
        backgroundColor: style.backgroundColor,
        color: style.color,
      };
    });

  expect(buttonPrimary).toBe('rgb(69, 137, 255)');
  expect(buttonStyles).toEqual({
    backgroundColor: buttonPrimary,
    color: 'rgb(22, 22, 22)',
  });
  expect(
    contrastRatio(buttonStyles.color, buttonStyles.backgroundColor),
  ).toBeGreaterThanOrEqual(4.5);
});

test('version 1 light computed colors retain the legacy contract', async ({
  page,
}) => {
  await gotoJourney(page, authJourney('signin'));

  const styles = await page.evaluate(() => {
    const root = document.documentElement;
    root.setAttribute('data-theme-schema', '1');
    root.setAttribute('data-color-mode', 'light');
    root.setAttribute('data-carbon-theme', 'white');
    root.style.setProperty('--brand-primary', '#123456');
    root.style.setProperty('--brand-primary-hover', '#234567');
    root.style.setProperty('--brand-primary-active', '#345678');
    root.style.setProperty('--brand-on-primary', '#abcdef');

    const surface = document.createElement('span');
    const layer = document.createElement('span');
    const action = document.createElement('span');
    const token = document.createElement('span');
    surface.className = 'bg-surface';
    layer.className = 'bg-layer';
    action.className = 'bg-primary';
    document.body.append(surface, layer, action, token);

    const readColor = property => {
      token.style.backgroundColor = `var(${property})`;
      return getComputedStyle(token).backgroundColor;
    };
    const button = document.createElement('button');
    button.className = 'cds--btn cds--btn--primary app-primary-button';
    button.style.position = 'fixed';
    button.style.insetInlineStart = '-10000px';
    document.body.append(button);
    const values = {
      surface: getComputedStyle(surface).backgroundColor,
      layer: getComputedStyle(layer).backgroundColor,
      tailwindPrimary: getComputedStyle(action).backgroundColor,
      buttonPrimary: getComputedStyle(button).backgroundColor,
      buttonContent: getComputedStyle(button).color,
      onPrimary: readColor('--cds-text-on-color'),
      link: readColor('--cds-link-primary'),
      focus: readColor('--cds-focus'),
      border: readColor('--cds-border-interactive'),
      tertiary: readColor('--cds-button-tertiary'),
    };

    surface.remove();
    layer.remove();
    action.remove();
    token.remove();
    button.remove();
    return values;
  });

  expect(styles).toEqual({
    surface: 'rgb(255, 255, 255)',
    layer: 'rgb(244, 244, 244)',
    tailwindPrimary: 'rgb(18, 52, 86)',
    buttonPrimary: 'rgb(18, 52, 86)',
    buttonContent: 'rgb(255, 255, 255)',
    onPrimary: 'rgb(171, 205, 239)',
    link: 'rgb(18, 52, 86)',
    focus: 'rgb(18, 52, 86)',
    border: 'rgb(18, 52, 86)',
    tertiary: 'rgb(18, 52, 86)',
  });
});

test('version 1 dark computed colors retain the legacy contract', async ({
  page,
}) => {
  await gotoJourney(page, authJourney('signin'));

  const styles = await page.evaluate(() => {
    const root = document.documentElement;
    root.setAttribute('data-theme-schema', '1');
    root.setAttribute('data-color-mode', 'dark');
    root.setAttribute('data-carbon-theme', 'g100');
    root.style.setProperty('--brand-primary', '#123456');
    root.style.setProperty('--brand-primary-hover', '#234567');
    root.style.setProperty('--brand-primary-active', '#345678');
    root.style.setProperty('--brand-on-primary', '#abcdef');

    const surface = document.createElement('span');
    const layer = document.createElement('span');
    const action = document.createElement('span');
    const token = document.createElement('span');
    surface.className = 'bg-surface';
    layer.className = 'bg-layer';
    action.className = 'bg-primary';
    document.body.append(surface, layer, action, token);

    const readColor = property => {
      token.style.backgroundColor = `var(${property})`;
      return getComputedStyle(token).backgroundColor;
    };
    const button = document.createElement('button');
    button.className = 'cds--btn cds--btn--primary app-primary-button';
    button.style.position = 'fixed';
    button.style.insetInlineStart = '-10000px';
    document.body.append(button);
    const values = {
      surface: getComputedStyle(surface).backgroundColor,
      layer: getComputedStyle(layer).backgroundColor,
      tailwindPrimary: getComputedStyle(action).backgroundColor,
      buttonPrimary: getComputedStyle(button).backgroundColor,
      buttonContent: getComputedStyle(button).color,
      onPrimary: readColor('--cds-text-on-color'),
      link: readColor('--cds-link-primary'),
      focus: readColor('--cds-focus'),
      border: readColor('--cds-border-interactive'),
      tertiary: readColor('--cds-button-tertiary'),
    };

    surface.remove();
    layer.remove();
    action.remove();
    token.remove();
    button.remove();
    return values;
  });

  expect(styles).toEqual({
    surface: 'rgb(10, 10, 10)',
    layer: 'rgb(22, 22, 22)',
    tailwindPrimary: 'rgb(18, 52, 86)',
    buttonPrimary: 'rgb(15, 98, 254)',
    buttonContent: 'rgb(255, 255, 255)',
    onPrimary: 'rgb(255, 255, 255)',
    link: 'rgb(18, 52, 86)',
    focus: 'rgb(18, 52, 86)',
    border: 'rgb(18, 52, 86)',
    tertiary: 'rgb(18, 52, 86)',
  });
});

test('Tailwind semantic utilities and Carbon properties stay aligned for every preset', async ({
  page,
}) => {
  await gotoJourney(page, authJourney('signin'));

  const results = await page.evaluate(properties => {
    const root = document.documentElement;
    const selectedPreset = root.getAttribute('data-carbon-theme');
    const surface = document.createElement('span');
    const action = document.createElement('span');
    const token = document.createElement('span');
    const prose = document.createElement('article');
    surface.className = 'bg-surface';
    action.className = 'bg-primary';
    prose.className = 'prose';
    prose.innerHTML = `
      <p>Body</p>
      <ul><li>Bullet</li></ul>
      <ol><li>Counter</li></ol>
      <pre><code>Code</code></pre>
      <figure><figcaption>Caption</figcaption></figure>
      <table><tbody><tr><td>Cell</td></tr></tbody></table>
    `;
    document.body.append(surface, action, token, prose);

    const readColor = property => {
      token.style.backgroundColor = `var(${property})`;
      return getComputedStyle(token).backgroundColor;
    };
    const values = ['white', 'g10', 'g90', 'g100'].map(preset => {
      root.setAttribute('data-carbon-theme', preset);
      return {
        preset,
        carbonSurface: readColor('--cds-background'),
        tailwindSurface: getComputedStyle(surface).backgroundColor,
        carbonAction: readColor('--cds-button-primary'),
        tailwindAction: getComputedStyle(action).backgroundColor,
        proseBody: getComputedStyle(prose).color,
        carbonBody: readColor('--cds-text-primary'),
        proseBullet: getComputedStyle(prose.querySelector('ul li'), '::marker')
          .color,
        carbonBullet: readColor('--cds-icon-secondary'),
        proseCounter: getComputedStyle(prose.querySelector('ol li'), '::marker')
          .color,
        carbonCounter: readColor('--cds-text-secondary'),
        prosePre: getComputedStyle(prose.querySelector('pre')).backgroundColor,
        carbonPre: readColor('--cds-background-inverse'),
        prosePreCode: getComputedStyle(prose.querySelector('pre code')).color,
        carbonPreCode: readColor('--cds-text-inverse'),
        proseCaption: getComputedStyle(prose.querySelector('figcaption')).color,
        carbonCaption: readColor('--cds-text-helper'),
        unresolvedProperties: properties.filter(
          property => readColor(property) === 'rgba(0, 0, 0, 0)',
        ),
      };
    });

    if (selectedPreset) root.setAttribute('data-carbon-theme', selectedPreset);
    surface.remove();
    action.remove();
    token.remove();
    prose.remove();
    return values;
  }, consumedCarbonProperties);

  for (const result of results) {
    expect(result.tailwindSurface, result.preset).toBe(result.carbonSurface);
    expect(result.tailwindAction, result.preset).toBe(result.carbonAction);
    expect(result.proseBody, result.preset).toBe(result.carbonBody);
    expect(result.proseBullet, result.preset).toBe(result.carbonBullet);
    expect(result.proseCounter, result.preset).toBe(result.carbonCounter);
    expect(result.prosePre, result.preset).toBe(result.carbonPre);
    expect(result.prosePreCode, result.preset).toBe(result.carbonPreCode);
    expect(result.proseCaption, result.preset).toBe(result.carbonCaption);
    expect(result.tailwindSurface, result.preset).not.toBe('rgba(0, 0, 0, 0)');
    expect(result.unresolvedProperties, result.preset).toEqual([]);
  }
});

for (const authForm of [
  {
    journey: 'signup',
    heading: 'Create account',
    input: 'Full name',
    action: 'Create account',
  },
  {
    journey: 'forgotPassword',
    heading: 'Reset your password',
    input: 'Email address',
    action: 'Send reset link',
  },
  {
    journey: 'changePassword',
    heading: 'Set a new password',
    input: 'New password',
    action: 'Save new password',
  },
]) {
  test(`${authForm.journey} uses the shared authentication form measure and typography`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await gotoJourney(page, authJourney(authForm.journey));

    const accountAccess = page.getByRole('region', { name: 'Account access' });
    const input = accountAccess.getByLabel(authForm.input, { exact: true });
    const action = accountAccess.getByRole('button', {
      name: authForm.action,
      exact: true,
    });
    const heading = accountAccess.getByRole('heading', {
      level: 1,
      name: authForm.heading,
    });
    const accountAccessBounds = await accountAccess.boundingBox();
    const inputBounds = await input.boundingBox();
    const actionBounds = await action.boundingBox();
    const headingStyles = await heading.evaluate(element => {
      const style = getComputedStyle(element);
      return {
        fontFamily: style.fontFamily,
        fontSize: style.fontSize,
        fontWeight: style.fontWeight,
        letterSpacing: style.letterSpacing,
        lineHeight: style.lineHeight,
      };
    });

    expect(accountAccessBounds.width).toBe(592);
    expect(inputBounds.x).toBe(actionBounds.x);
    expect(inputBounds.width).toBe(actionBounds.width);
    expect(headingStyles).toMatchObject({
      fontSize: '32px',
      fontWeight: '400',
      letterSpacing: 'normal',
      lineHeight: '40px',
    });
    expect(headingStyles.fontFamily).toContain('IBM Plex Sans');
  });
}

test('dark primary actions keep configured content and accessible contrast', async ({
  page,
}) => {
  await setColorMode(page, 'dark');
  await gotoJourney(page, authJourney('signin'));

  const button = page.getByRole('button', { name: 'Log in', exact: true });
  const tokenColors = await page.evaluate(() => {
    const probe = document.createElement('span');
    document.body.appendChild(probe);
    const resolveColor = token => {
      probe.style.backgroundColor = `var(${token})`;
      return getComputedStyle(probe).backgroundColor;
    };
    const colors = {
      primary: resolveColor('--cds-button-primary'),
      hover: resolveColor('--cds-button-primary-hover'),
      active: resolveColor('--cds-button-primary-active'),
    };
    probe.style.color = 'var(--brand-on-primary)';
    colors.onPrimary = getComputedStyle(probe).color;
    probe.remove();
    return colors;
  });
  const buttonColors = () =>
    button.evaluate(element => {
      const style = getComputedStyle(element);
      const iconStyle = getComputedStyle(element.querySelector('svg'));
      return {
        backgroundColor: style.backgroundColor,
        color: style.color,
        iconFill: iconStyle.fill,
      };
    });

  const defaultColors = await buttonColors();
  expect(defaultColors).toMatchObject({
    backgroundColor: tokenColors.primary,
    color: tokenColors.onPrimary,
    iconFill: tokenColors.onPrimary,
  });
  expect(
    contrastRatio(defaultColors.color, defaultColors.backgroundColor),
  ).toBeGreaterThanOrEqual(4.5);
  await button.hover();
  await expect
    .poll(() => buttonColors().then(colors => colors.backgroundColor))
    .toBe(tokenColors.hover);
  const hoverColors = await buttonColors();
  expect(hoverColors).toMatchObject({
    color: tokenColors.onPrimary,
    iconFill: tokenColors.onPrimary,
  });
  expect(
    contrastRatio(hoverColors.color, hoverColors.backgroundColor),
  ).toBeGreaterThanOrEqual(4.5);
  await page.mouse.down();
  await expect
    .poll(() => buttonColors().then(colors => colors.backgroundColor))
    .toBe(tokenColors.active);
  const activeColors = await buttonColors();
  expect(activeColors).toMatchObject({
    color: tokenColors.onPrimary,
    iconFill: tokenColors.onPrimary,
  });
  expect(
    contrastRatio(activeColors.color, activeColors.backgroundColor),
  ).toBeGreaterThanOrEqual(4.5);
  await page.mouse.move(0, 0);
  await page.mouse.up();
  await checkA11y(page, authJourney('signin'));
});

test('authentication shell collapses without horizontal overflow', async ({
  page,
}) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await gotoJourney(page, authJourney('signin'));

  await expect(page.getByRole('banner')).toBeVisible();
  const footer = page.getByRole('contentinfo');
  await expect(footer).toBeVisible();
  await expect(footer.getByText('Support', { exact: true })).toBeHidden();
  await expect(footer.getByText('Powered by Rapida AI')).toBeHidden();
  await expect(
    page.getByRole('main', { name: 'Authentication' }),
  ).toBeVisible();

  const main = page.getByRole('main', { name: 'Authentication' });
  const accountAccess = page.getByRole('region', { name: 'Account access' });
  await expect(accountAccess).toBeVisible();
  await expect(page.getByTestId('auth-center-box')).toHaveCount(0);
  await expect(page.getByRole('complementary')).toHaveCount(0);
  await expect(page.getByTestId('auth-form-rail')).toHaveCount(0);
  await expect(page.getByTestId('auth-illustration')).toHaveCount(0);

  const hasHorizontalOverflow = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth,
  );
  const mainBounds = await main.boundingBox();
  const accountAccessBounds = await accountAccess.boundingBox();
  const brandBounds = await page
    .getByRole('banner')
    .getByAltText('Rapida AI')
    .boundingBox();
  const emailInputBounds = await accountAccess
    .getByLabel('Email address')
    .boundingBox();
  const submitButtonBounds = await accountAccess
    .getByRole('button', { name: 'Log in', exact: true })
    .boundingBox();
  const googleButtonBounds = await accountAccess
    .getByRole('button', { name: 'Log in with Google' })
    .boundingBox();

  expect(hasHorizontalOverflow).toBe(false);
  expect(mainBounds).not.toBeNull();
  expect(accountAccessBounds).not.toBeNull();
  expect(brandBounds).not.toBeNull();
  expect(emailInputBounds).not.toBeNull();
  expect(submitButtonBounds).not.toBeNull();
  expect(googleButtonBounds).not.toBeNull();
  expect(mainBounds.width).toBe(375);
  expect(accountAccessBounds.x).toBe(16);
  expect(brandBounds.x).toBeGreaterThanOrEqual(16);
  expect(brandBounds.x).toBeLessThanOrEqual(24);
  expect(accountAccessBounds.width).toBe(343);
  expect(emailInputBounds.x).toBe(submitButtonBounds.x);
  expect(emailInputBounds.width).toBe(submitButtonBounds.width);
  expect(googleButtonBounds.x).toBe(submitButtonBounds.x);
  expect(googleButtonBounds.width).toBe(submitButtonBounds.width);
});

test('sign-in submits through the authentication client and opens the dashboard', async ({
  page,
}) => {
  await gotoJourney(page, authJourney('signin'));
  const request = page.waitForRequest(
    candidate =>
      candidate.method() === 'POST' &&
      candidate.url().endsWith(AUTHENTICATION_PATHS.signIn),
  );

  await submitSignIn(page);

  await request;
  await expect(page).toHaveURL(/\/dashboard$/);
});

test('sign-in displays the authentication client error', async ({ page }) => {
  await mockAuthenticationOperation(page, 'signIn', {
    success: false,
    message: 'Invalid email or password.',
  });
  await gotoJourney(page, authJourney('signin'));

  await submitSignIn(page);

  await expect(page.getByText('Invalid email or password.')).toBeVisible();
  await expect(page).toHaveURL(/\/auth\/signin$/);
});

test('sign-up submits through the authentication client and opens the dashboard', async ({
  page,
}) => {
  await gotoJourney(page, authJourney('signup'));
  const termsRow = page
    .getByRole('region', { name: 'Account access' })
    .locator('p')
    .filter({ hasText: 'By creating an account' });
  const submitButton = page.getByRole('button', { name: 'Create account' });
  const termsRowBounds = await termsRow.boundingBox();
  const submitButtonBounds = await submitButton.boundingBox();

  expect(termsRowBounds).not.toBeNull();
  expect(submitButtonBounds).not.toBeNull();
  expect(
    submitButtonBounds.y - (termsRowBounds.y + termsRowBounds.height),
  ).toBe(24);

  const request = page.waitForRequest(
    candidate =>
      candidate.method() === 'POST' &&
      candidate.url().endsWith(AUTHENTICATION_PATHS.signUp),
  );

  await submitSignUp(page);

  await request;
  await expect(page).toHaveURL(/\/dashboard$/);
});

test('sign-up displays the authentication client error', async ({ page }) => {
  await mockAuthenticationOperation(page, 'signUp', {
    success: false,
    message: 'An account already exists for this email.',
  });
  await gotoJourney(page, authJourney('signup'));

  await submitSignUp(page);

  await expect(
    page.getByText('An account already exists for this email.'),
  ).toBeVisible();
  await expect(page).toHaveURL(/\/auth\/signup$/);
});

test('forgot password confirms a successful client response', async ({
  page,
}) => {
  await gotoJourney(page, authJourney('forgotPassword'));
  const request = page.waitForRequest(
    candidate =>
      candidate.method() === 'POST' &&
      candidate.url().endsWith(AUTHENTICATION_PATHS.forgotPassword),
  );

  await page
    .getByLabel('Email address', { exact: true })
    .fill('user@example.test');
  await page.getByRole('button', { name: 'Send reset link' }).click();

  await request;
  await expect(page.getByText('Email sent')).toBeVisible();
  await expect(page.getByText(/Thanks! An email was sent/)).toBeVisible();
});

test('forgot password displays the authentication client error', async ({
  page,
}) => {
  await mockAuthenticationOperation(page, 'forgotPassword', {
    success: false,
    message: 'No account was found for this email.',
  });
  await gotoJourney(page, authJourney('forgotPassword'));

  await page.getByLabel('Email address').fill('missing@example.test');
  await page.getByRole('button', { name: 'Send reset link' }).click();

  await expect(
    page.getByText('No account was found for this email.'),
  ).toBeVisible();
  await expect(page.getByText('Email sent')).toHaveCount(0);
});

test('change password rejects mismatched passwords without calling the client', async ({
  page,
}) => {
  let requestCount = 0;
  page.on('request', request => {
    if (request.url().endsWith(AUTHENTICATION_PATHS.createPassword)) {
      requestCount += 1;
    }
  });
  await gotoJourney(page, authJourney('changePassword'));

  await page.locator('#new-password').fill('first password');
  await page.locator('#confirm-password').fill('second password');
  await page.getByRole('button', { name: 'Save new password' }).click();

  await expect(page.getByText('Passwords do not match.')).toBeVisible();
  expect(requestCount).toBe(0);
});

test('change password submits through the client and returns to sign-in', async ({
  page,
}) => {
  await gotoJourney(page, authJourney('changePassword'));
  const request = page.waitForRequest(
    candidate =>
      candidate.method() === 'POST' &&
      candidate.url().endsWith(AUTHENTICATION_PATHS.createPassword),
  );

  await page.locator('#new-password').fill('new password');
  await page.locator('#confirm-password').fill('new password');
  await page.getByRole('button', { name: 'Save new password' }).click();

  await request;
  await expect(page).toHaveURL(/\/auth\/signin$/);
});

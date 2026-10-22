const {
  COLOR_MODES,
  SCREENSHOT_VIEWPORTS,
  authenticatePage,
  expect,
  setColorMode,
  test,
} = require('../../fixtures');
const {
  ASSISTANT_ID,
  mockDebuggerAssistant,
} = require('../../mocks/clients/assistant');

for (const legacyRoute of [
  {
    path: `/preview/chat/${ASSISTANT_ID}`,
    destination: `/deployment/assistant/${ASSISTANT_ID}/playground/chat`,
    heading: 'Delivery concierge',
  },
  {
    path: `/preview/v2/chat/${ASSISTANT_ID}`,
    destination: `/deployment/assistant/${ASSISTANT_ID}/playground/chat`,
    heading: 'Delivery concierge',
  },
  {
    path: `/preview/call/${ASSISTANT_ID}`,
    destination: `/deployment/assistant/${ASSISTANT_ID}/playground/phone`,
    heading: 'Debug Phone Call',
  },
]) {
  test(`legacy debugger route ${legacyRoute.path} redirects to the assistant playground`, async ({
    page,
  }) => {
    await authenticatePage(page);
    await mockDebuggerAssistant(page);

    await page.goto(legacyRoute.path);

    await expect(page).toHaveURL(legacyRoute.destination);
    await expect(
      page.getByRole('heading', { name: legacyRoute.heading }),
    ).toBeVisible();
  });
}

for (const viewport of SCREENSHOT_VIEWPORTS) {
  for (const colorMode of COLOR_MODES) {
    test(`debugger v2 success screenshot at ${viewport.name} in ${colorMode} mode`, async ({
      page,
    }) => {
      await page.setViewportSize(viewport);
      await setColorMode(page, colorMode);
      await authenticatePage(page);
      await mockDebuggerAssistant(page);

      await page.goto(`/deployment/assistant/${ASSISTANT_ID}/playground/chat`);
      await expect(
        page.getByRole('heading', {
          name: 'Delivery concierge',
          exact: true,
        }),
      ).toBeVisible();
      await expect(
        page.getByText('Hello! How can I help you today?', { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole('button', { name: 'Start voice conversation' }),
      ).toBeEnabled();
      await expect(
        page.getByRole('button', { name: 'Back to assistant' }),
      ).toHaveCount(0);

      if (viewport.width >= 1280) {
        const assistantNavBounds = await page
          .getByRole('navigation', { name: 'Assistant actions' })
          .boundingBox();
        const conversationBounds = await page
          .getByRole('region', { name: 'Conversation' })
          .boundingBox();
        expect(assistantNavBounds).not.toBeNull();
        expect(conversationBounds).not.toBeNull();
        expect(conversationBounds.x).toBeGreaterThanOrEqual(
          assistantNavBounds.x + assistantNavBounds.width - 1,
        );
      }

      await page.getByRole('button', { name: 'Configuration' }).click();
      const languageModel = page.getByRole('button', {
        name: /Language model, OpenAI, gpt-4\.1-mini/,
      });
      await expect(page.getByText('rapida.credential_id')).toHaveCount(0);
      await expect(page.getByText('credential-secret-123')).toHaveCount(0);
      await expect(
        page.getByRole('button', { name: 'Copy Model' }).first(),
      ).toBeVisible();
      if ((await languageModel.getAttribute('aria-expanded')) === 'true') {
        await languageModel.click();
      }
      await page
        .getByRole('button', { name: 'Arguments, 3 variables' })
        .click();

      const customerName = page.getByRole('textbox', {
        name: '{{customer_name}}',
      });
      const deliveryInstructions = page.getByRole('textbox', {
        name: '{{delivery_instructions}}',
      });
      const orderContext = page.getByRole('textbox', {
        name: '{{order_context}}',
      });
      await expect(customerName).toHaveValue('Prashant');
      await expect(deliveryInstructions).toHaveValue(
        'Leave the package with reception if no one answers.',
      );
      await expect(orderContext).toHaveValue('{ "priority": "express" }');
      expect(
        await deliveryInstructions.evaluate(element => {
          const label = document.querySelector(`label[for="${element.id}"]`);
          if (!label) return true;
          return (
            label.getBoundingClientRect().bottom >
            element.getBoundingClientRect().top
          );
        }),
      ).toBe(false);
      await orderContext.scrollIntoViewIfNeeded();
      await page.evaluate(() => document.fonts.ready);

      await expect(page).toHaveScreenshot(
        [
          'preview',
          'debugger-v2',
          'success',
          `${viewport.name}-${colorMode}.png`,
        ],
        { fullPage: true, maxDiffPixelRatio: 0.01 },
      );
    });
  }
}

const configurationViewport = SCREENSHOT_VIEWPORTS.find(
  viewport => viewport.name === 'xl',
);

for (const colorMode of COLOR_MODES) {
  test(`phone debugger configuration in ${colorMode} mode`, async ({
    page,
  }) => {
    await page.setViewportSize(configurationViewport);
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await mockDebuggerAssistant(page);

    await page.goto(`/deployment/assistant/${ASSISTANT_ID}/playground/phone`);
    await page.getByRole('button', { name: 'Configuration' }).click();
    await expect(
      page.getByRole('complementary', { name: 'Session configuration' }),
    ).toBeVisible();
    await expect(page.getByText('Twilio')).toBeVisible();
    await expect(
      page.getByRole('heading', { name: 'Arguments' }),
    ).toBeVisible();
    await page.evaluate(() => document.fonts.ready);

    await expect(page).toHaveScreenshot(
      ['preview', 'phone-debugger', 'configuration', `${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });
}

for (const colorMode of COLOR_MODES) {
  test(`debugger v2 copyable configuration in ${colorMode} mode`, async ({
    page,
  }) => {
    await page.setViewportSize(configurationViewport);
    await setColorMode(page, colorMode);
    await authenticatePage(page);
    await mockDebuggerAssistant(page);

    await page.goto(`/deployment/assistant/${ASSISTANT_ID}/playground/chat`);
    await page.getByRole('button', { name: 'Configuration' }).click();
    await page
      .getByRole('button', { name: /Speech to text, Deepgram, nova-3/ })
      .click();
    await page
      .getByRole('button', { name: /Text to speech, Cartesia, sonic-2/ })
      .click();

    await expect(page.getByText('rapida.credential_id')).toHaveCount(0);
    await expect(page.getByText('credential-secret-123')).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Copy Model' })).toHaveCount(
      3,
    );
    await expect(
      page.getByRole('button', { name: 'Copy Voice' }),
    ).toBeVisible();
    await expect(
      page.getByRole('button', { name: 'Copy Provider' }),
    ).toHaveCount(0);
    await page.evaluate(() => document.fonts.ready);

    await expect(page).toHaveScreenshot(
      ['preview', 'debugger-v2', 'configuration', `${colorMode}.png`],
      { fullPage: true, maxDiffPixelRatio: 0.01 },
    );
  });
}

for (const viewport of SCREENSHOT_VIEWPORTS) {
  for (const colorMode of COLOR_MODES) {
    test(`debugger v2 voice unavailable screenshot at ${viewport.name} in ${colorMode} mode`, async ({
      page,
    }) => {
      await page.setViewportSize(viewport);
      await setColorMode(page, colorMode);
      await authenticatePage(page);
      await mockDebuggerAssistant(page, { withSpeechToText: false });

      await page.goto(`/deployment/assistant/${ASSISTANT_ID}/playground/chat`);
      await expect(
        page.getByText('Hello! How can I help you today?', { exact: true }),
      ).toBeVisible();
      const unavailableVoice = page.getByRole('button', {
        name: 'Voice is not configured',
      });
      await unavailableVoice.click();
      await expect(unavailableVoice).toHaveAttribute('aria-expanded', 'true');
      await expect(
        page.getByText(
          'Configure speech to text to start a voice conversation.',
          { exact: true },
        ),
      ).toBeVisible();
      await expect(
        page.getByRole('button', { name: 'Configure voice' }),
      ).toBeVisible();
      await page.evaluate(() => document.fonts.ready);

      await expect(page).toHaveScreenshot(
        [
          'preview',
          'debugger-v2',
          'voice-unavailable',
          `${viewport.name}-${colorMode}.png`,
        ],
        { fullPage: true, maxDiffPixelRatio: 0.01 },
      );
    });
  }
}

for (const viewport of SCREENSHOT_VIEWPORTS) {
  for (const colorMode of COLOR_MODES) {
    test(`phone debugger success screenshot at ${viewport.name} in ${colorMode} mode`, async ({
      page,
    }) => {
      await page.setViewportSize(viewport);
      await setColorMode(page, colorMode);
      await authenticatePage(page);
      await mockDebuggerAssistant(page);

      await page.goto(`/deployment/assistant/${ASSISTANT_ID}/playground/phone`);
      await expect(
        page.getByRole('heading', { name: 'Debug Phone Call' }),
      ).toBeVisible();
      const startCallButton = page.getByRole('button', { name: 'Start Call' });
      await expect(startCallButton).toBeEnabled();
      const startCallBox = await startCallButton.boundingBox();
      expect(startCallBox).not.toBeNull();
      expect(
        Math.abs(startCallBox.y + startCallBox.height - viewport.height),
      ).toBeLessThanOrEqual(1);
      await expect(
        page.getByRole('link', { name: 'Chat', exact: true }),
      ).toBeVisible();
      await page.evaluate(() => document.fonts.ready);

      await expect(page).toHaveScreenshot(
        [
          'preview',
          'phone-debugger',
          'success',
          `${viewport.name}-${colorMode}.png`,
        ],
        { fullPage: true, maxDiffPixelRatio: 0.01 },
      );
    });
  }
}

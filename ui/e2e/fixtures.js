const { test: base, expect } = require('@playwright/test');
const { evaluateA11yViolations } = require('./a11y-baseline');
const { installGrpcWebPreflightMock } = require('./mocks/grpc-web');
const {
  ASSISTANT_DETAIL_PATH,
  ASSISTANT_LIST_PATH,
  installAssistantClientMock,
} = require('./mocks/clients/assistant');
const {
  AUTHENTICATION_PATHS,
  installAuthenticationClientMock,
  mockAuthenticationOperation,
} = require('./mocks/clients/authentication');
const {
  ENDPOINT_LIST_PATH,
  installEndpointClientMock,
} = require('./mocks/clients/endpoint');
const {
  DASHBOARD_PATH,
  installDashboardClientMock,
} = require('./mocks/clients/dashboard');
const {
  ACTIVITY_LIST_PATHS,
  installActivityClientMock,
} = require('./mocks/clients/activity');
const {
  CREDENTIAL_LIST_PATH,
  installCredentialClientMock,
} = require('./mocks/clients/credential');
const {
  ORGANIZATION_PATH,
  WORKSPACE_LIST_PATHS,
  installWorkspaceClientMock,
} = require('./mocks/clients/workspace');

const COLOR_MODES = ['light', 'dark'];
const SCREENSHOT_VIEWPORTS = [
  { name: 'sm', width: 640, height: 900 },
  { name: 'lg', width: 1024, height: 900 },
  { name: 'xl', width: 1280, height: 900 },
  { name: '2xl', width: 1536, height: 960 },
];

const authState = {
  state: {
    currentUser: {
      id: '101',
      name: 'E2E User',
      email: 'e2e@example.test',
    },
    token: {
      token: 'token-e2e',
    },
    organizationRole: {
      id: '201',
      organizationid: '202',
      role: 'admin',
    },
    projectRoles: [
      {
        id: '301',
        projectid: '302',
        role: 'admin',
      },
    ],
    currentProjectRole: {
      id: '301',
      projectid: '302',
      role: 'admin',
    },
    featurePermissions: [],
  },
  version: 1,
};

async function installBrowserFixtures(page) {
  await page.route('https://www.googletagmanager.com/**', route =>
    route.abort('blockedbyclient'),
  );
  await page.route('http://localhost:8080/**', route => route.abort('failed'));
  await page.route('http://127.0.0.1:8080/**', route => route.abort('failed'));
  await installAssistantClientMock(page);
  await installAuthenticationClientMock(page);
  await installEndpointClientMock(page);
  await installDashboardClientMock(page);
  await installActivityClientMock(page);
  await installCredentialClientMock(page);
  await installWorkspaceClientMock(page);
  await installGrpcWebPreflightMock(page);
}

async function authenticatePage(page) {
  await page.addInitScript(state => {
    window.localStorage.setItem('rpd::__user', JSON.stringify(state));
    window.localStorage.setItem('rapida-sidebar-open', 'true');
  }, authState);
}

async function gotoJourney(page, journey) {
  if (journey.auth) await authenticatePage(page);

  const viewport = page.viewportSize();
  if (viewport) {
    await page.mouse.move(viewport.width - 1, viewport.height - 1);
  }

  await page.goto(journey.path);
  await page.waitForLoadState('domcontentloaded');

  if (viewport?.width < 768) {
    await page.mouse.move(viewport.width - 1, viewport.height - 1);
    const sidebar = page
      .locator('aside')
      .filter({
        has: page.getByRole('button', { name: 'Expand sidebar' }),
      })
      .first();
    if ((await sidebar.count()) > 0) {
      await expect(sidebar).toHaveCSS('width', '48px');
    }
  }

  await expect(page.locator('body')).toContainText(journey.expectText);
  if (journey.readyText) {
    await expect(page.locator('body')).toContainText(journey.readyText);
  }
  if (!journey.allowNotFound) {
    await expect(page.locator('body')).not.toContainText(
      "Sorry we couldn't find this page.",
    );
  }
}

async function setColorMode(page, colorMode) {
  await page.addInitScript(mode => {
    window.localStorage.setItem('ui-theme-mode', mode);
  }, colorMode);
}

async function expandDashboardForScreenshot(page) {
  await page.locator('[data-dashboard-page]').evaluate(dashboardPage => {
    dashboardPage.style.overflow = 'visible';
    dashboardPage.style.height = 'auto';
    dashboardPage.style.flex = 'none';

    let ancestor = dashboardPage.parentElement;
    while (ancestor && ancestor !== document.body) {
      ancestor.style.overflow = 'visible';
      ancestor.style.height = 'auto';
      ancestor = ancestor.parentElement;
    }
    document.documentElement.style.overflow = 'visible';
    document.body.style.overflow = 'visible';
  });
}

async function checkA11y(page, journey) {
  await page.addScriptTag({ path: require.resolve('axe-core/axe.min.js') });
  const results = await page.evaluate(async () =>
    window.axe.run(document, {
      runOnly: {
        type: 'tag',
        values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'],
      },
    }),
  );
  const violations = results.violations.filter(violation =>
    ['critical', 'serious'].includes(violation.impact),
  );
  const assessment = evaluateA11yViolations(journey.id, violations);

  expect(assessment).toEqual({ unapprovedViolations: [], overages: [] });
}

function toScreenshotSlug(value) {
  return value
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/[^a-z0-9]+/gi, '-')
    .toLowerCase();
}

function screenshotFileParts(journey, colorMode) {
  const [routeArea, ...journeyName] = journey.id.split('.');
  return [
    toScreenshotSlug(routeArea),
    `${toScreenshotSlug(journeyName.join('-'))}-${colorMode}.png`,
  ];
}

function screenshotBaselinePath(journey, colorMode) {
  return screenshotFileParts(journey, colorMode);
}

const test = base.extend({
  page: async ({ page }, use) => {
    await installBrowserFixtures(page);
    await use(page);
  },
});

module.exports = {
  test,
  expect,
  COLOR_MODES,
  SCREENSHOT_VIEWPORTS,
  ASSISTANT_LIST_PATH,
  ASSISTANT_DETAIL_PATH,
  ACTIVITY_LIST_PATHS,
  AUTHENTICATION_PATHS,
  CREDENTIAL_LIST_PATH,
  DASHBOARD_PATH,
  ENDPOINT_LIST_PATH,
  ORGANIZATION_PATH,
  WORKSPACE_LIST_PATHS,
  authenticatePage,
  checkA11y,
  expandDashboardForScreenshot,
  gotoJourney,
  mockAuthenticationOperation,
  setColorMode,
  screenshotBaselinePath,
};

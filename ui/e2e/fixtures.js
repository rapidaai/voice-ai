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
  WORKSPACE_CREATION_PATHS,
  WORKSPACE_LIST_PATHS,
  installWorkspaceClientMock,
  mockWorkspaceCreation,
} = require('./mocks/clients/workspace');
const {
  ACCESS_MANAGEMENT_PATHS,
  installAccessManagementClientMock,
  mockRoleListFailureAfterInitialLoad,
  mockRoleNotFound,
} = require('./mocks/clients/access-management');

const COLOR_MODES = ['light', 'dark'];
const SCREENSHOT_VIEWPORTS = [
  { name: 'sm', width: 640, height: 900 },
  { name: 'lg', width: 1024, height: 900 },
  { name: 'xl', width: 1280, height: 900 },
  { name: '2xl', width: 1536, height: 960 },
];

function createAuthState({ organization = true, project = true } = {}) {
  const hasProject = organization && project;
  const organizationRole = {
    id: '102',
    key: 'organization_admin',
    name: 'Organization Admin',
    scope: 'organization',
    permissionsList: [
      'organization.members.read',
      'organization.members.invite',
      'organization.members.remove',
      'organization.access.manage',
      'organization.roles.read',
      'organization.project_access_tokens.revoke',
    ].map((key, index) => ({
      id: String(1000 + index),
      key,
      scope: 'organization',
    })),
  };
  const projectRole = {
    id: '110',
    key: 'project_admin',
    name: 'Project Admin',
    scope: 'project',
    permissionsList: ['project.read', 'project.access_tokens.revoke'].map(
      (key, index) => ({
        id: String(2000 + index),
        key,
        scope: 'project',
      }),
    ),
  };

  return {
    state: {
      authentication: {
        principal: {
          subject: { type: 'user', id: '101' },
          selectedtenant: {
            organizationid: organization ? '202' : '0',
            projectid: hasProject ? '302' : '0',
          },
          proof: {
            method: 'user_session',
            reference: { type: 'token', value: '1' },
          },
        },
        user: {
          id: '101',
          name: 'E2E User',
          email: 'e2e@example.test',
          status: 'active',
        },
        organizationsList: organization
          ? [
              {
                id: '202',
                name: 'E2E Organization',
                rolesList: [organizationRole],
                projectsList: hasProject
                  ? [
                      {
                        id: '302',
                        name: 'E2E Project',
                        rolesList: [projectRole],
                      },
                    ]
                  : [],
              },
            ]
          : [],
      },
      token: {
        token: 'token-e2e',
      },
      selectedOrganizationID: organization ? '202' : '',
      selectedProjectID: hasProject ? '302' : '',
    },
    version: 3,
  };
}

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
  await installAccessManagementClientMock(page);
  await installGrpcWebPreflightMock(page);
}

async function authenticatePage(page, options) {
  await page.addInitScript(state => {
    window.localStorage.setItem('rpd::__user', JSON.stringify(state));
    window.localStorage.setItem('rapida-sidebar-open', 'true');
  }, createAuthState(options));
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
    await expect(page.locator('body')).not.toContainText('Page not found');
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
  await page.evaluate(async () => {
    const finiteAnimations = document
      .getAnimations()
      .filter(animation =>
        Number.isFinite(animation.effect?.getComputedTiming().endTime),
      );
    await Promise.all(
      finiteAnimations.map(animation => animation.finished.catch(() => {})),
    );
  });
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
  ACCESS_MANAGEMENT_PATHS,
  ASSISTANT_LIST_PATH,
  ASSISTANT_DETAIL_PATH,
  ACTIVITY_LIST_PATHS,
  AUTHENTICATION_PATHS,
  CREDENTIAL_LIST_PATH,
  DASHBOARD_PATH,
  ENDPOINT_LIST_PATH,
  ORGANIZATION_PATH,
  WORKSPACE_CREATION_PATHS,
  WORKSPACE_LIST_PATHS,
  authenticatePage,
  checkA11y,
  expandDashboardForScreenshot,
  gotoJourney,
  mockAuthenticationOperation,
  mockRoleListFailureAfterInitialLoad,
  mockRoleNotFound,
  mockWorkspaceCreation,
  setColorMode,
  screenshotBaselinePath,
};

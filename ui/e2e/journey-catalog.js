const covered = (id, parentJourney, evidence) => ({
  id,
  parentJourney,
  status: 'covered',
  evidence,
});

const deferred = (id, parentJourney, reason) => ({
  id,
  parentJourney,
  status: 'deferred',
  reason,
});

const disabled = (id, reason) => ({ id, status: 'disabled', reason });

const journeySubjourneys = [
  covered(
    'auth.signin.submit',
    'auth.signin',
    'journeys/auth/authentication.spec.js',
  ),
  covered(
    'auth.signin.serviceError',
    'auth.signin',
    'journeys/auth/authentication.spec.js',
  ),
  covered(
    'auth.signup.submit',
    'auth.signup',
    'journeys/auth/authentication.spec.js',
  ),
  covered(
    'auth.forgotPassword.submit',
    'auth.forgotPassword',
    'journeys/auth/authentication.spec.js',
  ),
  covered(
    'auth.changePassword.submit',
    'auth.changePassword',
    'journeys/auth/authentication.spec.js',
  ),
  covered(
    'dashboard.loading',
    'dashboard.home',
    'journeys/dashboard/state-screenshots.spec.js',
  ),
  covered(
    'dashboard.success',
    'dashboard.home',
    'journeys/dashboard/state-screenshots.spec.js',
  ),
  covered(
    'dashboard.error',
    'dashboard.home',
    'journeys/dashboard/state-screenshots.spec.js',
  ),
  covered(
    'dashboard.responsiveLayout',
    'dashboard.home',
    'journeys/dashboard/responsive.spec.js',
  ),
  covered(
    'deployment.assistant.empty',
    'deployment.assistant',
    'journeys/deployment/assistant-listing.spec.js',
  ),
  covered(
    'deployment.assistant.initialError',
    'deployment.assistant',
    'journeys/deployment/assistant-listing.spec.js',
  ),
  covered(
    'deployment.assistant.createChooser',
    'deployment.createAssistant',
    'journey-viewport-screenshots.spec.js',
  ),
  covered(
    'deployment.endpoint.overview',
    'deployment.endpointOverview',
    'journeys/deployment/endpoint-detail.spec.js',
  ),
  covered(
    'deployment.endpoint.emptyActivity',
    'deployment.endpointOverview',
    'journeys/deployment/endpoint-detail.spec.js',
  ),
  covered(
    'deployment.endpoint.playground',
    'deployment.endpointPlayground',
    'journeys/deployment/endpoint-detail.spec.js',
  ),
  covered(
    'deployment.endpoint.logs',
    'deployment.endpointLogs',
    'journey-viewport-screenshots.spec.js',
  ),
  covered(
    'deployment.endpoint.versions',
    'deployment.endpointVersions',
    'journey-viewport-screenshots.spec.js',
  ),
  covered(
    'deployment.endpoint.createVersion',
    'deployment.createEndpointVersion',
    'journey-viewport-screenshots.spec.js',
  ),
  covered(
    'deployment.endpoint.settings',
    'deployment.endpointSettings',
    'journeys/deployment/endpoint-detail.spec.js',
  ),
  covered(
    'integration.provider.catalog',
    'integration.models',
    'journeys/integration/provider-catalog.spec.js',
  ),
  covered(
    'integration.provider.search',
    'integration.models',
    'journeys/integration/provider-catalog.spec.js',
  ),
  covered(
    'organization.users.empty',
    'organization.users',
    'journeys/organization/listings.spec.js',
  ),
  covered(
    'organization.projects.empty',
    'organization.projects',
    'journeys/organization/listings.spec.js',
  ),

  deferred(
    'deployment.assistant.overview',
    'deployment.assistant',
    'Assistant entity, dashboard, and version fixtures are required.',
  ),
  deferred(
    'deployment.assistant.sessions',
    'deployment.assistant',
    'Assistant conversation list fixtures are required.',
  ),
  deferred(
    'deployment.assistant.sessionDetail',
    'deployment.assistant',
    'Conversation and message fixtures are required.',
  ),
  deferred(
    'deployment.assistant.versionHistory',
    'deployment.assistant',
    'Assistant version fixtures are required.',
  ),
  deferred(
    'deployment.assistant.createPromptVersion',
    'deployment.assistant',
    'Assistant provider and credential fixtures are required.',
  ),
  deferred(
    'deployment.assistant.createWebsocketVersion',
    'deployment.assistant',
    'WebSocket assistant fixtures are required.',
  ),
  deferred(
    'deployment.assistant.createAgentkitVersion',
    'deployment.assistant',
    'AgentKit assistant fixtures are required.',
  ),
  deferred(
    'deployment.assistant.createAgentflowVersion',
    'deployment.assistant',
    'Agentflow assistant fixtures are required.',
  ),
  deferred(
    'deployment.assistant.generalSettings',
    'deployment.assistant',
    'Assistant entity fixtures are required.',
  ),
  deferred(
    'deployment.assistant.agentflowEditor',
    'deployment.assistant',
    'Agentflow graph fixtures are required.',
  ),
  ...[
    'analysis.list',
    'analysis.create',
    'analysis.edit',
    'tools.list',
    'tools.create',
    'tools.edit',
    'webhooks.list',
    'webhooks.create',
    'webhooks.edit',
    'authentication.list',
    'authentication.create',
    'authentication.edit',
    'telemetry.list',
    'telemetry.create',
    'telemetry.edit',
    'storage.list',
    'storage.create',
    'storage.edit',
  ].map(name =>
    deferred(
      `deployment.assistant.${name}`,
      'deployment.assistant',
      'Assistant configuration entity fixtures are required.',
    ),
  ),
  ...[
    'deployments.list',
    'deployments.web.create',
    'deployments.web.edit',
    'deployments.call.create',
    'deployments.call.edit',
    'deployments.api.create',
    'deployments.api.edit',
    'deployments.debugger.create',
    'deployments.debugger.edit',
  ].map(name =>
    deferred(
      `deployment.assistant.${name}`,
      'deployment.assistant',
      'Assistant deployment fixtures are required.',
    ),
  ),
  deferred(
    'deployment.endpoint.configureExisting',
    'deployment.endpoint',
    'The routed ConfigureEndpointPage currently renders no UI.',
  ),
  deferred(
    'deployment.endpoint.notFound',
    'deployment.endpointOverview',
    'The endpoint layout has no persistent not-found state after load failure.',
  ),
  deferred(
    'deployment.assistant.notFound',
    'deployment.assistant',
    'The assistant layout does not set its persistent error state for transport failures.',
  ),
  ...['atlassian', 'github', 'hubspot', 'gmail', 'slack'].map(provider =>
    deferred(
      `connections.${provider}.callback`,
      'integration.models',
      'OAuth callback fixtures and redirect assertions are required.',
    ),
  ),
  ...['chat', 'phone', 'public'].map(channel =>
    deferred(
      `preview.assistant.${channel}`,
      'deployment.assistant',
      'Assistant-specific realtime media fixtures are required.',
    ),
  ),
  deferred(
    'integration.provider.unsupportedDetail',
    'integration.models',
    'The catalog links providers that do not have a detail route implementation.',
  ),
  covered(
    'organization.users.inviteOrganization',
    'organization.users',
    'journeys/organization/listings.spec.js',
  ),
  deferred(
    'organization.users.inviteProject',
    'organization.users',
    'The dialog needs deterministic project membership fixtures.',
  ),
  covered(
    'organization.projects.create',
    'organization.projects',
    'journeys/organization/listings.spec.js',
  ),
  deferred(
    'integration.credentials.create',
    'integration.projectCredential',
    'The dialog needs deterministic credential mutation fixtures.',
  ),
  covered(
    'account.notifications',
    'account.settings',
    'journeys/account/settings.spec.js',
  ),
  ...[
    'list',
    'create',
    'detail',
    'document.add',
    'structure.add',
    'logs',
    'googleDrive.callback',
    'oneDrive.callback',
    'confluence.callback',
    'sharePoint.callback',
    'github.callback',
    'notion.callback',
  ].map(name =>
    disabled(
      `knowledge.${name}`,
      'Knowledge is disabled by workspace.features.knowledge in the development configuration.',
    ),
  ),
];

module.exports = { journeySubjourneys };

const routeRoots = [
  { root: '/auth', journey: 'auth.signin' },
  {
    root: '/knowledge',
    disabledFeature: 'workspace.features.knowledge',
    reason: 'Knowledge is disabled in the default development config.',
  },
  { root: '/onboarding', journey: 'onboarding.organization' },
  { root: '/dashboard', journey: 'dashboard.home' },
  { root: '/deployment', journey: 'deployment.assistant' },
  { root: '/integration', journey: 'integration.models' },
  { root: '/account', journey: 'account.settings' },
  { root: '/logs', journey: 'logs.request' },
  { root: '/organization', journey: 'organization.overview' },
  {
    root: '/preview',
    deferredReason:
      'Preview pages need assistant-specific realtime fixtures before browser coverage is stable.',
  },
  {
    root: '/connect-common',
    deferredReason:
      'OAuth callback routes need provider redirect fixtures before browser coverage is stable.',
  },
  {
    root: '/connect-knowledge',
    disabledFeature: 'workspace.features.knowledge',
    reason: 'Knowledge connection routes are disabled with knowledge.',
  },
  {
    root: '/connect-crm',
    deferredReason:
      'OAuth callback routes need CRM redirect fixtures before browser coverage is stable.',
  },
  {
    root: '/connect-action',
    deferredReason:
      'OAuth callback routes need action-provider redirect fixtures before browser coverage is stable.',
  },
  { root: '/static', journey: 'static.privacy' },
];

const routeJourneys = [
  {
    id: 'navigation.root',
    path: '/',
    auth: false,
    expectText: 'Sign in',
    readyText: 'Sign in with GitHub',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'auth.root',
    path: '/auth',
    auth: false,
    expectText: 'Sign in',
    readyText: 'Sign in with GitHub',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'auth.signin',
    path: '/auth/signin',
    auth: false,
    expectText: 'Sign in',
    readyText: 'Sign in with GitHub',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'auth.signup',
    path: '/auth/signup',
    auth: false,
    expectText: 'Create account',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'auth.forgotPassword',
    path: '/auth/forgot-password',
    auth: false,
    expectText: 'Reset your password',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'auth.changePassword',
    path: '/auth/change-password/token-1',
    auth: false,
    expectText: 'Set a new password',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'static.privacy',
    path: '/static/privacy-policy',
    auth: false,
    expectText: 'Privacy policy',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'static.terms',
    path: '/static/terms-conditions',
    auth: false,
    expectText: 'Terms of Service',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'dashboard.home',
    path: '/dashboard',
    auth: true,
    expectText: 'Project overview',
    readyText: 'Top assistants',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.assistant',
    path: '/deployment/assistant',
    auth: true,
    expectText: 'Assistants',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.root',
    path: '/deployment',
    auth: true,
    expectText: 'Assistants',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.endpoint',
    path: '/deployment/endpoint',
    auth: true,
    expectText: 'Hosted Endpoints',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.createAssistant',
    path: '/deployment/assistant/create-assistant',
    auth: true,
    expectText: 'Select a usecase template',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.createEndpoint',
    path: '/deployment/endpoint/create-endpoint',
    auth: true,
    expectText: 'Complete all steps to create a new endpoint.',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.connectWebsocket',
    path: '/deployment/assistant/connect-websocket',
    auth: true,
    expectText: 'Complete all steps to connect with a websocket',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.connectAgentkit',
    path: '/deployment/assistant/connect-agentkit',
    auth: true,
    expectText: 'Complete all steps to connect a new AgentKit.',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.createAgentflow',
    path: '/deployment/assistant/create-agentflow',
    auth: true,
    expectText: 'Quick start',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'integration.models',
    path: '/integration/models',
    auth: true,
    expectText: 'Providers and Models',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'integration.deepgram',
    path: '/integration/models/deepgram',
    auth: true,
    expectText: 'Deepgram',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'integration.googleSpeechService',
    path: '/integration/models/google-speech-service',
    auth: true,
    expectText: 'Google Speech Service',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'integration.azureSpeechService',
    path: '/integration/models/azure-speech-service',
    auth: true,
    expectText: 'Azure Cognitive Services',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'integration.cartesia',
    path: '/integration/models/cartesia',
    auth: true,
    expectText: 'Cartesia',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'integration.elevenLabs',
    path: '/integration/models/elevenlabs',
    auth: true,
    expectText: 'ElevenLabs',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.endpointOverview',
    path: '/deployment/endpoint/2301664620831571968/overview',
    auth: true,
    expectText: 'Endpoint activity',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.endpointPlayground',
    path: '/deployment/endpoint/2301664620831571968/playground',
    auth: true,
    expectText: 'Playground',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.endpointSettings',
    path: '/deployment/endpoint/2301664620831571968/settings',
    auth: true,
    expectText: 'General Settings',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.endpointLogs',
    path: '/deployment/endpoint/2301664620831571968/logs',
    auth: true,
    expectText: 'Filter table',
    readyText: 'vrsn_2301664620831571999',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.endpointVersions',
    path: '/deployment/endpoint/2301664620831571968/versions',
    auth: true,
    expectText: 'No versions found',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'deployment.createEndpointVersion',
    path: '/deployment/endpoint/2301664620831571968/create-endpoint-version',
    auth: true,
    expectText: 'Create a new version of this endpoint.',
    readyText: 'Configure instruction',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'integration.projectCredential',
    path: '/integration/project-credential',
    auth: true,
    expectText: 'Project Developer Keys',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'integration.personalCredential',
    path: '/integration/personal-credential',
    auth: true,
    expectText: 'Personal Tokens',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'logs.llm',
    path: '/logs',
    auth: true,
    expectText: 'LLM Logs',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'logs.request',
    path: '/logs/request',
    auth: true,
    expectText: 'Request Logs',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'logs.tool',
    path: '/logs/tool',
    auth: true,
    expectText: 'Tool Logs',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'logs.conversation',
    path: '/logs/conversation',
    auth: true,
    expectText: 'Conversation Logs',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'logs.traces',
    path: '/logs/traces',
    auth: true,
    expectText: 'No traces found',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'organization.overview',
    path: '/organization',
    auth: true,
    expectText: 'Organization Profile',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'organization.users',
    path: '/organization/users',
    auth: true,
    expectText: 'Users',
    readyText: 'No organization users',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'organization.projects',
    path: '/organization/projects',
    auth: true,
    expectText: 'Projects',
    readyText: 'No projects',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'organization.security',
    path: '/organization/security',
    auth: true,
    expectText: 'Organization Security',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'account.settings',
    path: '/account',
    auth: true,
    expectText: 'Account Information',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'onboarding.organization',
    path: '/onboarding/organization',
    auth: true,
    expectText: 'Set up your organization',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'onboarding.root',
    path: '/onboarding',
    auth: true,
    expectText: 'Set up your organization',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'onboarding.project',
    path: '/onboarding/project',
    auth: true,
    expectText: 'Create your first project',
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'navigation.globalNotFound',
    path: '/missing-journey',
    auth: false,
    expectText: "Sorry we couldn't find this page.",
    allowNotFound: true,
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'navigation.deploymentNotFound',
    path: '/deployment/missing-journey',
    auth: true,
    expectText: "Sorry we couldn't find this page.",
    allowNotFound: true,
    checks: ['smoke', 'a11y', 'screenshot'],
  },
  {
    id: 'navigation.staticNotFound',
    path: '/static/missing-journey',
    auth: false,
    expectText: "Sorry we couldn't find this page.",
    allowNotFound: true,
    checks: ['smoke', 'a11y', 'screenshot'],
  },
];

const sourceRouteInventory = {
  'account.tsx': [{ path: '', journey: 'account.settings' }],
  'auth.tsx': [
    { path: 'signup', journey: 'auth.signup' },
    { path: 'signin', journey: 'auth.signin' },
    { path: 'forgot-password', journey: 'auth.forgotPassword' },
    { path: 'change-password/:token', journey: 'auth.changePassword' },
  ],
  'connect-action.tsx': [
    {
      path: '/',
      deferredReason: 'OAuth callback layout route needs provider fixtures.',
    },
    {
      path: 'gmail',
      deferredReason: 'OAuth callback route needs action-provider fixtures.',
    },
    {
      path: 'slack',
      deferredReason: 'OAuth callback route needs action-provider fixtures.',
    },
  ],
  'connect-knowledge.tsx': [
    {
      path: '/',
      count: 3,
      deferredReason: 'Connect route layouts need provider fixtures.',
    },
    {
      path: 'atlassian',
      deferredReason: 'OAuth callback route needs provider fixtures.',
    },
    {
      path: 'github',
      count: 2,
      deferredReason: 'OAuth callback route needs provider fixtures.',
    },
    {
      path: 'hubspot',
      deferredReason: 'OAuth callback route needs CRM provider fixtures.',
    },
    {
      path: 'google-drive',
      disabledFeature: 'workspace.features.knowledge',
    },
    { path: 'one-drive', disabledFeature: 'workspace.features.knowledge' },
    { path: 'confluence', disabledFeature: 'workspace.features.knowledge' },
    { path: 'share-point', disabledFeature: 'workspace.features.knowledge' },
    { path: 'notion', disabledFeature: 'workspace.features.knowledge' },
  ],
  'dashboard.tsx': [{ path: '/', journey: 'dashboard.home' }],
  'deployment.tsx': [
    {
      path: '/',
      journey: 'deployment.assistant',
    },
    { path: 'endpoint', journey: 'deployment.endpoint' },
    {
      path: 'endpoint/create-endpoint',
      journey: 'deployment.createEndpoint',
    },
    {
      path: 'endpoint/configure-endpoint',
      journey: 'deployment.createEndpoint',
    },
    {
      path: 'endpoint/configure-endpoint/:endpointId',
      deferredReason: 'Endpoint detail fixtures are not available yet.',
    },
    {
      path: 'endpoint/:endpointId',
      journey: 'deployment.endpointOverview',
    },
    {
      path: 'create-endpoint-version',
      journey: 'deployment.createEndpointVersion',
    },
    {
      path: ':tab',
      count: 2,
      deferredReason:
        'Assistant detail tabs still need entity fixtures; endpoint tabs use the endpoint detail journeys.',
    },
    { path: 'assistant', journey: 'deployment.assistant' },
    {
      path: 'assistant/:assistantId',
      deferredReason: 'Assistant detail fixtures are not available yet.',
    },
    {
      path: 'sessions/:sessionId',
      deferredReason: 'Conversation detail fixtures are not available yet.',
    },
    {
      path: 'create-new-version',
      deferredReason: 'Assistant version fixtures are not available yet.',
    },
    {
      path: 'create-websocket-version',
      deferredReason: 'Assistant version fixtures are not available yet.',
    },
    {
      path: 'create-agentkit-version',
      deferredReason: 'Assistant version fixtures are not available yet.',
    },
    {
      path: 'create-agentflow-version',
      deferredReason: 'Assistant version fixtures are not available yet.',
    },
    {
      path: 'edit-assistant/',
      deferredReason: 'Assistant detail fixtures are not available yet.',
    },
    {
      path: 'edit-agentflow/',
      deferredReason: 'Agentflow fixtures are not available yet.',
    },
    {
      path: 'configure-analysis/',
      deferredReason: 'Assistant analysis fixtures are not available yet.',
    },
    {
      path: 'configure-analysis/create',
      deferredReason: 'Assistant analysis fixtures are not available yet.',
    },
    {
      path: 'configure-analysis/:analysisId',
      deferredReason: 'Assistant analysis fixtures are not available yet.',
    },
    {
      path: 'configure-tool',
      deferredReason: 'Assistant tool fixtures are not available yet.',
    },
    {
      path: 'configure-tool/create',
      deferredReason: 'Assistant tool fixtures are not available yet.',
    },
    {
      path: 'configure-tool/:assistantToolId',
      deferredReason: 'Assistant tool fixtures are not available yet.',
    },
    {
      path: 'configure-webhook/',
      deferredReason: 'Assistant webhook fixtures are not available yet.',
    },
    {
      path: 'configure-webhook/create',
      deferredReason: 'Assistant webhook fixtures are not available yet.',
    },
    {
      path: 'configure-webhook/:webhookId',
      deferredReason: 'Assistant webhook fixtures are not available yet.',
    },
    {
      path: 'configure-authentication/',
      deferredReason:
        'Assistant authentication fixtures are not available yet.',
    },
    {
      path: 'configure-authentication/create',
      deferredReason:
        'Assistant authentication fixtures are not available yet.',
    },
    {
      path: 'configure-authentication/edit',
      deferredReason:
        'Assistant authentication fixtures are not available yet.',
    },
    {
      path: 'configure-telemetry/',
      deferredReason: 'Assistant telemetry fixtures are not available yet.',
    },
    {
      path: 'configure-telemetry/create',
      deferredReason: 'Assistant telemetry fixtures are not available yet.',
    },
    {
      path: 'configure-telemetry/:telemetryId',
      deferredReason: 'Assistant telemetry fixtures are not available yet.',
    },
    {
      path: 'configure-storage/',
      deferredReason: 'Assistant storage fixtures are not available yet.',
    },
    {
      path: 'configure-storage/create',
      deferredReason: 'Assistant storage fixtures are not available yet.',
    },
    {
      path: 'configure-storage/:storageId',
      deferredReason: 'Assistant storage fixtures are not available yet.',
    },
    {
      path: 'deployment/',
      deferredReason: 'Assistant deployment fixtures are not available yet.',
    },
    {
      path: 'deployment/web/',
      deferredReason: 'Assistant deployment fixtures are not available yet.',
    },
    {
      path: 'deployment/web/:deploymentId/',
      deferredReason: 'Assistant deployment fixtures are not available yet.',
    },
    {
      path: 'deployment/call/',
      deferredReason: 'Assistant deployment fixtures are not available yet.',
    },
    {
      path: 'deployment/call/:deploymentId/',
      deferredReason: 'Assistant deployment fixtures are not available yet.',
    },
    {
      path: 'deployment/api/',
      deferredReason: 'Assistant deployment fixtures are not available yet.',
    },
    {
      path: 'deployment/api/:deploymentId/',
      deferredReason: 'Assistant deployment fixtures are not available yet.',
    },
    {
      path: 'deployment/debugger/',
      deferredReason: 'Assistant deployment fixtures are not available yet.',
    },
    {
      path: 'deployment/debugger/:deploymentId/',
      deferredReason: 'Assistant deployment fixtures are not available yet.',
    },
    {
      path: 'assistant/create-assistant',
      journey: 'deployment.createAssistant',
    },
    {
      path: 'assistant/connect-websocket',
      journey: 'deployment.connectWebsocket',
    },
    {
      path: 'assistant/connect-agentkit',
      journey: 'deployment.connectAgentkit',
    },
    {
      path: 'assistant/create-agentflow',
      journey: 'deployment.createAgentflow',
    },
  ],
  'integration.tsx': [
    { path: '/', journey: 'integration.models' },
    { path: '', journey: 'integration.models' },
    { path: 'models', journey: 'integration.models' },
    {
      path: 'models/:provider',
      journey: 'integration.deepgram',
    },
    {
      path: 'project-credential',
      journey: 'integration.projectCredential',
    },
    { path: 'personal-credential', journey: 'integration.personalCredential' },
  ],
  'knowledge.tsx': [
    { path: '', disabledFeature: 'workspace.features.knowledge' },
    {
      path: 'create-knowledge',
      disabledFeature: 'workspace.features.knowledge',
    },
    { path: ':id', disabledFeature: 'workspace.features.knowledge' },
    {
      path: ':id/add-knowledge-file',
      disabledFeature: 'workspace.features.knowledge',
    },
    {
      path: ':id/add-structure-file',
      disabledFeature: 'workspace.features.knowledge',
    },
  ],
  'observability.tsx': [
    {
      path: '/',
      count: 2,
      journey: 'logs.llm',
    },
    { path: '/request', journey: 'logs.request' },
    { path: '/knowledge', disabledFeature: 'workspace.features.knowledge' },
    { path: '/tool', journey: 'logs.tool' },
    { path: '/conversation', journey: 'logs.conversation' },
    { path: '/traces', journey: 'logs.traces' },
  ],
  'onborading.tsx': [
    { path: '/', journey: 'onboarding.organization' },
    { path: 'organization', journey: 'onboarding.organization' },
    { path: 'project', journey: 'onboarding.project' },
  ],
  'organization.tsx': [
    { path: '/', journey: 'organization.overview' },
    { path: '', journey: 'organization.overview' },
    { path: 'users', journey: 'organization.users' },
    { path: 'projects', journey: 'organization.projects' },
    { path: 'security', journey: 'organization.security' },
  ],
  'preview.tsx': [
    {
      path: '/',
      deferredReason:
        'Preview layout needs assistant-specific realtime fixtures.',
    },
    {
      path: 'call/:assistantId/',
      deferredReason:
        'Phone preview needs assistant-specific realtime fixtures.',
    },
    {
      path: 'chat/:assistantId',
      deferredReason:
        'Chat preview needs assistant-specific realtime fixtures.',
    },
    {
      path: 'public/assistant/:assistantId',
      deferredReason:
        'Public preview needs assistant-specific realtime fixtures.',
    },
  ],
  'static.tsx': [
    { path: 'privacy-policy', journey: 'static.privacy' },
    { path: 'terms-conditions', journey: 'static.terms' },
  ],
};

const journeysWithCheck = check =>
  routeJourneys.filter(journey => journey.checks.includes(check));

module.exports = {
  routeRoots,
  routeJourneys,
  sourceRouteInventory,
  journeysWithCheck,
};

import { matchRoutes, type RouteObject } from 'react-router-dom';
import { accountRoute } from '../account';
import type { AppRouteHandle } from '../breadcrumb-handle';
import { dashboardRoute } from '../dashboard';
import { deploymentRoute } from '../deployment';
import { integrationRoute } from '../integration';
import { knowledgeRoute } from '../knowledge';
import { observabilityRoute } from '../observability';
import { organizationRoute } from '../organization';

const workspaceRoutes: RouteObject[] = [
  dashboardRoute,
  deploymentRoute,
  integrationRoute,
  knowledgeRoute,
  accountRoute,
  observabilityRoute,
  organizationRoute,
];

function getBreadcrumbs(pathname: string) {
  const matches = matchRoutes(workspaceRoutes, pathname);
  expect(matches).not.toBeNull();

  return matches!
    .map(match => (match.route.handle as AppRouteHandle)?.breadcrumb)
    .filter(definition => definition !== undefined);
}

describe('workspace route breadcrumb metadata', () => {
  it.each([
    ['/dashboard', ['Dashboard']],
    ['/organization/users', ['Organization', 'Users']],
    [
      '/integration/project-credential',
      ['Integrations', 'Project credentials'],
    ],
    ['/logs/conversation', ['Logs', 'Conversations']],
    [
      '/deployment/assistant/create-assistant',
      ['Deployment', 'Assistants', 'Create assistant'],
    ],
  ])('declares static metadata for %s', (pathname, expectedLabels) => {
    const labels = getBreadcrumbs(pathname).map(definition =>
      'label' in definition! ? definition.label : 'dynamic',
    );

    expect(labels).toEqual(expectedLabels);
  });

  it('declares dynamic entity metadata without exposing route parameters', () => {
    const definitions = getBreadcrumbs(
      '/deployment/assistant/assistant-1/sessions/session-1',
    );

    expect(definitions.map(definition => Object.keys(definition!)[0])).toEqual([
      'label',
      'label',
      'render',
      'label',
      'label',
    ]);
    expect(
      definitions.some(
        definition =>
          'label' in definition! && definition.label === 'assistant-1',
      ),
    ).toBe(false);
  });

  it('uses explicit not-found metadata for unknown deployment routes', () => {
    const labels = getBreadcrumbs('/deployment/missing').map(definition =>
      'label' in definition! ? definition.label : 'dynamic',
    );

    expect(labels).toEqual(['Deployment', 'Not found']);
  });
});

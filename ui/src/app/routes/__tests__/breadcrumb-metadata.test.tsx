import { render, screen } from '@testing-library/react';
import { matchRoutes, type RouteObject } from 'react-router-dom';
import { accountRoute } from '../account';
import type { AppRouteHandle } from '../breadcrumb-handle';
import { dashboardRoute } from '../dashboard';
import { deploymentRoute } from '../deployment';
import { integrationRoute } from '../integration';
import { knowledgeRoute } from '../knowledge';
import { observabilityRoute } from '../observability';
import { organizationRoute } from '../organization';
import { useAssistantPageStore } from '@/stores/assistant/assistant.store';
import { useKnowledgePageStore } from '@/stores/knowledge/knowledge.store';

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
  beforeEach(() => {
    useAssistantPageStore.setState({
      currentAssistant: null,
      currentAssistantStatus: 'idle',
    });
    useKnowledgePageStore.setState({
      currentKnowledge: null,
      currentKnowledgeStatus: 'idle',
    });
  });

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

  it('renders the matching assistant name from the authoritative store', () => {
    useAssistantPageStore.setState({
      currentAssistant: {
        getId: () => 'assistant-1',
        getName: () => 'Support bot',
      } as any,
      currentAssistantStatus: 'ready',
    });
    const definition = getBreadcrumbs(
      '/deployment/assistant/assistant-1/overview',
    ).find(candidate => 'render' in candidate!);
    expect(definition).toBeDefined();

    if (definition && 'render' in definition) {
      const DynamicLabel = definition.render;
      render(<DynamicLabel params={{ assistantId: 'assistant-1' }} />);
    }

    expect(screen.getByText('Support bot')).toBeInTheDocument();
    expect(screen.queryByText('assistant-1')).not.toBeInTheDocument();
  });

  it('renders an explicit label when the assistant lookup fails', () => {
    useAssistantPageStore.setState({
      currentAssistant: null,
      currentAssistantStatus: 'error',
    });
    const definition = getBreadcrumbs(
      '/deployment/assistant/assistant-1/overview',
    ).find(candidate => 'render' in candidate!);
    expect(definition).toBeDefined();

    if (definition && 'render' in definition) {
      const DynamicLabel = definition.render;
      render(<DynamicLabel params={{ assistantId: 'assistant-1' }} />);
    }

    expect(screen.getByText('Assistant unavailable')).toBeInTheDocument();
  });

  it('renders an explicit label when the knowledge lookup fails', () => {
    useKnowledgePageStore.setState({ currentKnowledgeStatus: 'error' });
    const definition = getBreadcrumbs(
      '/knowledge/knowledge-1/add-knowledge-file',
    ).find(candidate => 'render' in candidate!);
    expect(definition).toBeDefined();

    if (definition && 'render' in definition) {
      const DynamicLabel = definition.render;
      render(<DynamicLabel params={{ id: 'knowledge-1' }} />);
    }

    expect(screen.getByText('Knowledge unavailable')).toBeInTheDocument();
  });

  it('uses explicit not-found metadata for unknown deployment routes', () => {
    const labels = getBreadcrumbs('/deployment/missing').map(definition =>
      'label' in definition! ? definition.label : 'dynamic',
    );

    expect(labels).toEqual(['Deployment', 'Not found']);
  });
});

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { Dashboard } from '../dashboard';
import { Deployment } from '../deployment';
import { ExternalTool } from '../external-tools';
import { Knowledge } from '../knowledge';
import { Observability } from '../observability';
import { Project } from '../project';
import { Team } from '../team';
import { Vault } from '../vault';

let mockOpen = true;
let mockWorkspaceFeatures: Record<string, boolean> = {};

jest.mock('@/context/sidebar-context', () => ({
  useSidebar: () => ({ open: mockOpen }),
}));

jest.mock('@/workspace', () => ({
  useWorkspace: () => ({ features: mockWorkspaceFeatures }),
}));

jest.mock('@/app/components/ui/primitives/text', () => ({
  Text: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const renderAtPath = (path: string, element: React.ReactElement) =>
  render(<MemoryRouter initialEntries={[path]}>{element}</MemoryRouter>);

describe('sidebar navigation items', () => {
  beforeEach(() => {
    mockOpen = true;
    mockWorkspaceFeatures = {};
  });

  it('renders dashboard, deployment, knowledge, integration, and organization links', () => {
    renderAtPath(
      '/deployment/assistant',
      <>
        <Dashboard />
        <Deployment />
        <Knowledge />
        <ExternalTool />
        <Team />
        <Project />
      </>,
    );

    expect(screen.getByRole('link', { name: /Dashboard/i })).toHaveAttribute(
      'href',
      '/dashboard',
    );
    expect(
      screen.getByRole('link', { name: /Dashboard/i }).closest('li'),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Assistants/i })).toHaveAttribute(
      'href',
      '/deployment/assistant',
    );
    expect(screen.getByRole('link', { name: /Endpoints/i })).toHaveAttribute(
      'href',
      '/deployment/endpoint',
    );
    expect(screen.getByRole('link', { name: /Knowledge/i })).toHaveAttribute(
      'href',
      '/knowledge',
    );
    expect(
      screen.getByRole('link', { name: /External integrations/i }),
    ).toHaveAttribute('href', '/integration/models');
    expect(
      screen.getByRole('link', { name: /Users and Teams/i }),
    ).toHaveAttribute('href', '/organization/users');
    expect(screen.getByRole('link', { name: /Projects/i })).toHaveAttribute(
      'href',
      '/organization/projects',
    );
    expect(
      screen.getByRole('link', { name: /Assistants/i }).firstChild,
    ).toHaveClass('text-foreground');
  });

  it('renders observability links', () => {
    renderAtPath('/logs/traces', <Observability />);

    expect(screen.getByRole('link', { name: /Trace/i })).toHaveAttribute(
      'href',
      '/logs/traces',
    );
    expect(
      screen.getByRole('link', { name: /Knowledge logs/i }),
    ).toHaveAttribute('href', '/logs/knowledge');
    expect(screen.getByRole('link', { name: /LLM logs/i })).toHaveAttribute(
      'href',
      '/logs',
    );
    expect(screen.getByRole('link', { name: /Tool logs/i })).toHaveAttribute(
      'href',
      '/logs/tool',
    );
    expect(screen.getByRole('link', { name: /Request logs/i })).toHaveAttribute(
      'href',
      '/logs/request',
    );
    expect(
      screen.getByRole('link', { name: /Conversation logs/i }),
    ).toHaveAttribute('href', '/logs/conversation');
  });

  it('hides observability links for disabled feature flags', () => {
    mockWorkspaceFeatures = { telemetry: false, knowledge: false };

    renderAtPath('/logs', <Observability />);

    expect(
      screen.queryByRole('link', { name: /Trace/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: /Knowledge logs/i }),
    ).not.toBeInTheDocument();
  });

  it('expands credential links from the active vault route and can collapse them', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderAtPath('/integration/personal-credential', <Vault />);

    expect(
      screen.getByRole('link', { name: /Credentials/i }).firstChild,
    ).toHaveClass('text-foreground');
    expect(
      screen.getByRole('link', { name: /Project Credential/i }),
    ).toHaveAttribute('href', '/integration/project-credential');
    expect(
      screen.getByRole('link', { name: /Personal Token/i }),
    ).toHaveAttribute('href', '/integration/personal-credential');

    await user.click(screen.getByText('Credentials'));

    expect(
      screen.queryByRole('link', { name: /Project Credential/i }),
    ).not.toBeInTheDocument();
  });

  it('keeps expanded credential links visually hidden when the sidebar is collapsed', () => {
    mockOpen = false;

    renderAtPath('/integration/project-credential', <Vault />);

    expect(
      screen.getByRole('link', { name: /Project Credential/i }).parentElement,
    ).toHaveClass('hidden');
  });
});

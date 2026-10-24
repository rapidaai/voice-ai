import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { AssistantWebWidgetlDeploymentDialog } from '../assistant-web-widget-deployment-modal';

jest.mock('@rapidaai/react', () => {
  class AssistantWebpluginDeployment {
    getId() {
      return 'deployment-1';
    }
    getAssistantid() {
      return 'assistant-123';
    }
    getInputaudio() {
      return undefined;
    }
    getOutputaudio() {
      return undefined;
    }
  }

  return { AssistantWebpluginDeployment };
});

jest.mock('@/configs', () => ({
  CONFIG: {
    deployment: {
      webWidgetScriptUrl: 'https://assets.client.example/widget/app.min.js',
    },
  },
}));

jest.mock('@/app/components/dialogs/shared', () => ({
  RightSideModal: ({ children, label, title }: any) => (
    <section>
      <div>{label}</div>
      <h2>{title}</h2>
      {children}
    </section>
  ),
  DeploymentRow: ({ children, label }: any) => (
    <div>
      <span>{label}</span>
      {children}
    </div>
  ),
  DeploymentSectionHeader: ({ label }: any) => <h3>{label}</h3>,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  CopyButton: ({ children }: any) => <button>{children}</button>,
  Tabs: ({ tabs, children, selectedIndex = 0 }: any) => (
    <div>
      {tabs.map((tab: string) => (
        <button key={tab}>{tab}</button>
      ))}
      {Array.isArray(children) ? children[selectedIndex] : children}
    </div>
  ),
}));

jest.mock('@/app/components/ui/editor/code-highlighting', () => ({
  CodeHighlighting: ({ code }: any) => <pre>{code}</pre>,
}));

jest.mock('@/app/components/layout/container/message/notice-block', () => ({
  YellowNoticeBlock: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/domain/pills/provider-model-pill', () => ({
  ProviderPill: ({ provider }: any) => <span>{provider}</span>,
}));

describe('AssistantWebWidgetlDeploymentDialog', () => {
  it('shows tenant-neutral web widget snippets when viewing deployments', () => {
    const { AssistantWebpluginDeployment } =
      jest.requireMock('@rapidaai/react');

    render(
      <AssistantWebWidgetlDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        deployment={new AssistantWebpluginDeployment()}
      />,
    );

    const snippetText = screen
      .getAllByText((_, node) => node?.tagName === 'PRE')
      .map(snippet => snippet.textContent)
      .join('\n');

    expect(snippetText).toContain(
      'https://assets.client.example/widget/app.min.js',
    );
    expect(snippetText).toContain('assistant_id: "assistant-123"');
    expect(snippetText).toContain('token: "{PROJECT_KEY}"');
    expect(snippetText).not.toMatch(/rapida/i);
  });
});

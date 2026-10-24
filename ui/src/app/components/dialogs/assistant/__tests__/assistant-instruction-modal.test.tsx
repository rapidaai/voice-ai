import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { AssistantWebwidgetDeploymentDialog } from '../assistant-instruction-modal';

jest.mock('@/app/components/ui/primitives/modal', () => ({
  Modal: ({ open, children }: any) => (open ? <div>{children}</div> : null),
  ModalHeader: ({ title, children }: any) => (
    <header>
      <h2>{title}</h2>
      {children}
    </header>
  ),
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalFooter: ({ children }: any) => <footer>{children}</footer>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({ children, renderIcon: Icon, ...props }: any) => (
    <button {...props}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  Launch: () => <svg data-testid="documentation-icon" />,
}));

jest.mock('@/app/components/ui/editor/code-highlighting', () => ({
  CodeHighlighting: ({ code }: any) => <pre>{code}</pre>,
}));

jest.mock(
  '@/app/components/dialogs/shared/deployment-modal-primitives',
  () => ({
    DeploymentSectionHeader: ({ label }: any) => <h3>{label}</h3>,
  }),
);

jest.mock('@/theme/documentation-url', () => ({
  useDocumentationUrl: () => 'https://docs.example.test',
}));

jest.mock('@/configs', () => ({
  CONFIG: {
    deployment: {
      webWidgetScriptUrl: 'https://assets.client.example/widget/app.min.js',
    },
  },
}));

describe('AssistantWebwidgetDeploymentDialog', () => {
  const originalOpen = window.open;

  beforeEach(() => {
    window.open = jest.fn();
  });

  afterEach(() => {
    window.open = originalOpen;
  });

  it('renders deployment snippets with the assistant id', () => {
    render(
      <AssistantWebwidgetDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        assistantId="assistant-123"
      />,
    );

    expect(
      screen.getByRole('heading', { name: 'Deployment completed' }),
    ).toBeInTheDocument();
    expect(screen.getByText(/app.min.js/)).toBeInTheDocument();
    expect(screen.getByText(/assistant-123/)).toBeInTheDocument();
  });

  it('keeps web widget snippets tenant neutral', () => {
    render(
      <AssistantWebwidgetDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        assistantId="assistant-123"
      />,
    );

    const snippets = screen.getAllByText((_, node) => node?.tagName === 'PRE');
    const snippetText = snippets.map(snippet => snippet.textContent).join('\n');

    expect(snippetText).toContain(
      'https://assets.client.example/widget/app.min.js',
    );
    expect(snippetText).toContain('token: "{PROJECT_KEY}"');
    expect(snippetText).not.toMatch(/rapida/i);
  });

  it('opens documentation with the standard action icon', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <AssistantWebwidgetDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        assistantId="assistant-123"
      />,
    );

    const documentationButton = screen.getByRole('button', {
      name: 'View Documentation',
    });
    expect(screen.getByTestId('documentation-icon')).toBeInTheDocument();

    await user.click(documentationButton);

    expect(window.open).toHaveBeenCalledWith(
      'https://docs.example.test',
      '_blank',
    );
  });

  it('closes from the secondary action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();

    render(
      <AssistantWebwidgetDeploymentDialog
        modalOpen
        setModalOpen={setModalOpen}
        assistantId="assistant-123"
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Close' }));

    expect(setModalOpen).toHaveBeenCalledWith(false);
  });
});

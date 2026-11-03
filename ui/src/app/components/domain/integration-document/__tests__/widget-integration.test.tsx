import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { WidgetIntegration } from '../widget-integration';

jest.mock('@carbon/react', () => ({
  Button: ({ children, onClick }: any) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  CodeSnippet: ({ children, copyText, ...props }: any) => (
    <pre aria-label={props['aria-label']} data-copy-text={copyText}>
      {children}
    </pre>
  ),
  Tag: ({ children }: any) => <span>{children}</span>,
}));

jest.mock('@carbon/icons-react', () => ({
  Launch: () => <svg />,
}));

jest.mock('@/app/components/domain/cards/rapida-credential-card', () => ({
  RapidaCredentialCard: () => <div>Project credential card</div>,
}));

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

describe('WidgetIntegration', () => {
  const originalOpen = window.open;

  beforeEach(() => {
    window.open = jest.fn();
  });

  afterEach(() => {
    window.open = originalOpen;
  });

  it('renders one credential-aware sequence for deploying the widget', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<WidgetIntegration assistantId="assistant-123" />);

    expect(screen.getByText('Step 1')).toBeInTheDocument();
    expect(screen.getByText('Step 2')).toBeInTheDocument();
    expect(screen.getByText('Step 3')).toBeInTheDocument();
    expect(screen.getByText('Step 4')).toBeInTheDocument();
    expect(screen.getByText('Project credential card')).toBeInTheDocument();
    expect(screen.getByText(/app.min.js/)).toBeInTheDocument();
    expect(
      screen.getByText(/assistant_id: "assistant-123"/),
    ).toBeInTheDocument();
    expect(screen.getByText(/layout: "docked-right"/)).toBeInTheDocument();
    expect(screen.getByText(/mode: "light"/)).toBeInTheDocument();
    expect(screen.queryByText(/color: "black"/)).not.toBeInTheDocument();

    await user.click(
      screen.getByRole('button', { name: 'View documentation' }),
    );
    expect(window.open).toHaveBeenCalledWith(
      'https://docs.example.test',
      '_blank',
    );
  });

  it('uses an explicit assistant placeholder when deployment data is incomplete', () => {
    render(<WidgetIntegration assistantId="" />);

    expect(
      screen.getByText(/assistant_id: "YOUR_ASSISTANT_ID"/),
    ).toBeInTheDocument();
  });
});

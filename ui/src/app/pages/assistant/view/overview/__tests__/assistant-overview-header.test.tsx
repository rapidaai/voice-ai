import React from 'react';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { Overview } from '@/app/pages/assistant/view/overview';

const mockGoToCreateAssistantVersion = jest.fn();
let mockLoading = false;

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: mockLoading,
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToCreateAssistantVersion: mockGoToCreateAssistantVersion,
    goToDeploymentAssistant: jest.fn(),
  }),
}));

jest.mock('@/app/components/ui/feedback/loaders/section-loader', () => ({
  SectionLoader: () => <div>Loading section</div>,
}));

jest.mock('@/app/components/ui/feedback/notification', () => ({
  ActionNotification: ({ title }: any) => <div>{title}</div>,
}));

jest.mock('@/app/pages/assistant/view/overview/assistant-analytics', () => ({
  AssistantAnalytics: () => <section>Assistant analytics</section>,
}));

jest.mock('@carbon/icons-react', () => ({
  SourceControl: () => <span>source-control</span>,
}));

jest.mock('@carbon/react', () => ({
  Breadcrumb: ({ children }: any) => <nav>{children}</nav>,
  BreadcrumbItem: ({ children, href, isCurrentPage }: any) =>
    isCurrentPage ? <span>{children}</span> : <a href={href}>{children}</a>,
  Button: ({
    children,
    className,
    kind,
    renderIcon: Icon,
    size,
    ...props
  }: any) => (
    <button className={className} data-kind={kind} data-size={size} {...props}>
      {Icon ? <Icon size={16} /> : null}
      {children}
    </button>
  ),
  HeaderGlobalBar: ({ children, className, ...props }: any) => (
    <div className={className} role="toolbar" {...props}>
      {children}
    </div>
  ),
}));

const makeAssistant = () =>
  ({
    getId: () => 'assistant-1',
    getName: () => 'Production assistant',
    getApideployment: () => ({ getId: () => 'api-deployment-1' }),
    getDebuggerdeployment: () => undefined,
    getWebplugindeployment: () => undefined,
    getPhonedeployment: () => undefined,
  }) as any;

describe('Assistant overview header', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockLoading = false;
  });

  it('shows the Carbon-style overview header with a flush full-height primary create version action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<Overview currentAssistant={makeAssistant()} />);

    expect(screen.getByText('Assistants')).toHaveAttribute(
      'href',
      '/deployment/assistant',
    );
    expect(screen.getByText('Production assistant')).toBeInTheDocument();

    const toolbar = screen.getByRole('toolbar', {
      name: 'Assistant overview header actions',
    });
    expect(toolbar).toHaveClass('h-full');
    expect(toolbar).toHaveClass('items-center');
    expect(toolbar).not.toHaveClass('pr-4');
    expect(within(toolbar).getByText('source-control')).toBeInTheDocument();

    const createVersionButton = screen.getByRole('button', {
      name: 'Create new version',
    });
    expect(createVersionButton).toHaveAttribute('data-kind', 'primary');
    expect(createVersionButton).toHaveAttribute('data-size', 'lg');
    expect(createVersionButton).toHaveClass('h-full!');
    expect(createVersionButton).toHaveClass('min-h-full!');
    expect(createVersionButton).toHaveClass('items-center');
    expect(createVersionButton).toHaveClass('justify-center');

    await user.click(createVersionButton);

    expect(mockGoToCreateAssistantVersion).toHaveBeenCalledWith('assistant-1');
  });

  it('keeps the overview header hidden while the overview is loading', () => {
    mockLoading = true;

    render(<Overview currentAssistant={makeAssistant()} />);

    expect(screen.getByText('Loading section')).toBeInTheDocument();
    expect(
      screen.queryByRole('toolbar', {
        name: 'Assistant overview header actions',
      }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('Assistant analytics')).not.toBeInTheDocument();
  });
});

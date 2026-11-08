import React from 'react';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { Overview } from '@/app/pages/assistant/view/overview';

const mockGoToCreateAssistantVersion = jest.fn();
const mockGoToDeploymentAssistant = jest.fn();
let mockLoading = false;

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: mockLoading,
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToCreateAssistantVersion: mockGoToCreateAssistantVersion,
    goToDeploymentAssistant: mockGoToDeploymentAssistant,
  }),
}));

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => ({ resolvedMode: 'light' }),
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
  CheckmarkOutline: () => <span>complete</span>,
  ChevronUp: () => <span>chevron-up</span>,
  CircleDash: () => <span>not-started</span>,
  Incomplete: () => <span>in-progress</span>,
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
  Heading: ({ children, className, ...props }: any) => (
    <h2 className={className} {...props}>
      {children}
    </h2>
  ),
  HeaderGlobalBar: ({ children, className, ...props }: any) => (
    <div className={className} role="toolbar" {...props}>
      {children}
    </div>
  ),
  IconButton: ({ children, label, ...props }: any) => (
    <button title={label} {...props}>
      {children}
    </button>
  ),
  Section: ({ as: Component = 'section', children, ...props }: any) => (
    <Component {...props}>{children}</Component>
  ),
}));

const makeAssistant = ({
  deployed = true,
  deploymentStatus = 'ACTIVE',
} = {}) =>
  ({
    getId: () => 'assistant-1',
    getName: () => 'Production assistant',
    getApideployment: () =>
      deployed
        ? {
            getId: () => 'api-deployment-1',
            getStatus: () => deploymentStatus,
          }
        : undefined,
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
    const { container } = render(
      <Overview currentAssistant={makeAssistant()} />,
    );

    const resourceHeader = container.querySelector(
      '[data-resource-detail-header]',
    );
    expect(resourceHeader).toHaveClass('h-12', 'bg-shell');

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

  it('shows a floating deployment checklist only when no deployment is enabled', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const { rerender } = render(
      <Overview currentAssistant={makeAssistant({ deployed: false })} />,
    );

    const checklist = screen.getByLabelText('Assistant deployment checklist');
    expect(checklist).toHaveClass('fixed', 'bottom-4', 'right-4');
    expect(within(checklist).getByText('Deploy your assistant')).toBeVisible();
    expect(within(checklist).getByText('25% complete')).toBeVisible();
    expect(within(checklist).getByText('Assistant created')).toBeVisible();
    expect(
      within(checklist).getByRole('button', {
        name: 'Choose a deployment channel',
      }),
    ).toBeVisible();
    expect(
      within(checklist).getByRole('button', {
        name: 'Configure deployment settings',
      }),
    ).toBeVisible();
    expect(
      within(checklist).getByRole('button', {
        name: 'Enable your deployment',
      }),
    ).toBeVisible();

    const toggle = within(checklist).getByRole('button', {
      name: 'Toggle assistant deployment checklist',
    });
    expect(toggle).toHaveAttribute(
      'title',
      'Collapse or expand deployment setup',
    );
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    await user.click(toggle);
    expect(toggle).toHaveAttribute('aria-expanded', 'false');

    await user.click(
      within(checklist).getByRole('button', {
        name: 'View deployment options',
      }),
    );
    expect(mockGoToDeploymentAssistant).toHaveBeenCalledWith('assistant-1');

    rerender(<Overview currentAssistant={makeAssistant()} />);
    expect(
      screen.queryByLabelText('Assistant deployment checklist'),
    ).not.toBeInTheDocument();
  });

  it('shows deployment setup as 75% complete while deployment is inactive', () => {
    render(
      <Overview
        currentAssistant={makeAssistant({ deploymentStatus: 'INACTIVE' })}
      />,
    );

    const checklist = screen.getByLabelText('Assistant deployment checklist');
    expect(within(checklist).getByText('75% complete')).toBeVisible();
    expect(within(checklist).getByText('Assistant created')).toBeVisible();
    expect(
      within(checklist).getByText('Choose a deployment channel'),
    ).toBeVisible();
    expect(
      within(checklist).getByText('Configure deployment settings'),
    ).toBeVisible();
    expect(
      within(checklist).getByRole('button', {
        name: 'Enable your deployment',
      }),
    ).toBeVisible();
  });
});

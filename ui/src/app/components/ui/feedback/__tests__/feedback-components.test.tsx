import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Renew, Search } from '@carbon/icons-react';

import { EmptyState } from '../empty-state';
import { ErrorContainer } from '../error-container';
import { ErrorMessage } from '../error-message';
import {
  CarbonIconIndicator,
  recordStateToIconIndicator,
} from '../icon-indicator';
import { Loader } from '../loader';
import {
  ActionNotification,
  LinkNotification,
  Notification,
} from '../notification';
import {
  CarbonShapeIndicator,
  recordStateToShapeIndicator,
} from '../shape-indicator';
import { Toast, ToastNotification } from '../toast';
import { PageLoading } from '../loading';
import { AnimatedLine, LineLoader } from '../loaders/line-loader';
import { PageLoader } from '../loaders/page-loader';
import { SectionLoader } from '../loaders/section-loader';
import { Spinner } from '../loaders/spinner';

let mockLoading = false;
let mockLoadingType = 'line';
let mockToasts: Array<{
  id: string;
  message?: unknown;
  type?: string;
  height?: number;
}> = [];
const mockEndPause = jest.fn();
const mockStartPause = jest.fn();
const mockToastRemove = jest.fn();
const mockUpdateHeight = jest.fn();

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: mockLoading,
    loadingType: mockLoadingType,
  }),
}));

jest.mock('react-hot-toast/headless', () => ({
  __esModule: true,
  default: {
    remove: (...args: unknown[]) => mockToastRemove(...args),
  },
  useToaster: () => ({
    toasts: mockToasts,
    handlers: {
      endPause: mockEndPause,
      startPause: mockStartPause,
      updateHeight: mockUpdateHeight,
    },
  }),
}));

jest.mock('@carbon/react', () => ({
  ActionableNotification: ({
    actionButtonLabel,
    children,
    className,
    hideCloseButton,
    inline,
    kind,
    lowContrast,
    onActionButtonClick,
    onCloseButtonClick,
    role,
    subtitle,
    title,
  }: any) => (
    <section
      className={className}
      data-hide-close={hideCloseButton}
      data-inline={inline}
      data-kind={kind}
      data-low-contrast={lowContrast}
      role={role}
    >
      <strong>{title}</strong>
      {subtitle ? <p>{subtitle}</p> : null}
      <button onClick={onActionButtonClick}>{actionButtonLabel}</button>
      <button onClick={onCloseButtonClick}>Close</button>
      {children}
    </section>
  ),
  Button: ({ children, onClick, renderIcon: Icon, size, kind }: any) => (
    <button data-kind={kind} data-size={size} onClick={onClick}>
      {Icon ? <Icon data-testid="button-icon" /> : null}
      {children}
    </button>
  ),
  InlineNotification: ({
    className,
    hideCloseButton,
    kind,
    lowContrast,
    onCloseButtonClick,
    subtitle,
    title,
  }: any) => (
    <section
      className={className}
      data-hide-close={hideCloseButton}
      data-kind={kind}
      data-low-contrast={lowContrast}
    >
      <strong>{title}</strong>
      {subtitle ? <p>{subtitle}</p> : null}
      <button onClick={onCloseButtonClick}>Close</button>
    </section>
  ),
  Loading: ({ active, className, description, small, withOverlay }: any) => (
    <div
      aria-label={description}
      className={className}
      data-active={active}
      data-small={small}
      data-with-overlay={withOverlay}
      role="status"
    />
  ),
  ToastNotification: ({
    caption,
    className,
    hideCloseButton,
    kind,
    lowContrast,
    onClose,
    onCloseButtonClick,
    role,
    statusIconDescription,
    subtitle,
    timeout,
    title,
  }: any) => (
    <section
      aria-label={statusIconDescription}
      className={className}
      data-hide-close={hideCloseButton}
      data-kind={kind}
      data-low-contrast={lowContrast}
      data-timeout={timeout}
      role={role}
    >
      <strong>{title}</strong>
      {subtitle ? <p>{subtitle}</p> : null}
      {caption ? <small>{caption}</small> : null}
      <button onClick={onClose}>Close toast</button>
      <button onClick={onCloseButtonClick}>Close toast button</button>
    </section>
  ),
  preview__IconIndicator: ({
    align,
    iconDescription,
    kind,
    label,
    size,
  }: any) => (
    <span
      data-align={align}
      data-icon-description={iconDescription}
      data-icon-kind={kind}
      data-size={size}
    >
      {label}
    </span>
  ),
  preview__ShapeIndicator: ({ kind, label, textSize }: any) => (
    <span data-kind={kind} data-text-size={textSize}>
      {label}
    </span>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  ArrowLeft: (props: any) => <svg data-testid="arrow-left" {...props} />,
  Renew: (props: any) => <svg data-testid="renew-icon" {...props} />,
  Search: (props: any) => <svg data-testid="search-icon" {...props} />,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({ children, onClick, renderIcon: Icon, size }: any) => (
    <button data-size={size} onClick={onClick}>
      {Icon ? <Icon /> : null}
      {children}
    </button>
  ),
}));

describe('feedback components', () => {
  beforeEach(() => {
    mockLoading = false;
    mockLoadingType = 'line';
    mockToasts = [];
    mockEndPause.mockClear();
    mockStartPause.mockClear();
    mockToastRemove.mockClear();
    mockUpdateHeight.mockClear();
  });

  it('renders an accessible empty state with icon and action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onAction = jest.fn();

    render(
      <EmptyState
        icon={Search}
        title="No deployments"
        subtitle="Create a deployment to start testing."
        action="Create deployment"
        actionIcon={Renew}
        onAction={onAction}
      />,
    );

    expect(screen.getByLabelText('No deployments')).toBeInTheDocument();
    expect(screen.getByTestId('search-icon')).toBeInTheDocument();
    expect(
      screen.getByText('Create a deployment to start testing.'),
    ).toBeInTheDocument();

    await user.click(
      screen.getByRole('button', { name: /create deployment/i }),
    );

    expect(onAction).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('button-icon')).toBeInTheDocument();
  });

  it('supports custom empty-state action content without default action', () => {
    render(
      <EmptyState
        title="No results"
        actionComponent={<button>Reset filters</button>}
      />,
    );

    expect(screen.getByRole('button', { name: 'Reset filters' })).toBeVisible();
    expect(
      screen.queryByRole('button', { name: 'No results' }),
    ).not.toBeInTheDocument();
  });

  it('renders page error content and executes its recovery action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onAction = jest.fn();

    render(
      <ErrorContainer
        code="403"
        title="Access denied"
        description="You do not have access to this workspace."
        actionLabel="Go back"
        className="custom-error"
        onAction={onAction}
      />,
    );

    expect(
      screen.getByRole('heading', { name: 'Access denied' }),
    ).toBeVisible();
    expect(screen.getByText('403')).toBeVisible();
    expect(screen.getByText(/do not have access/i)).toBeVisible();
    expect(screen.getByLabelText('Access denied')).toHaveClass('custom-error');
    expect(screen.getByTestId('arrow-left')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Go back' }));

    expect(onAction).toHaveBeenCalledTimes(1);
  });

  it('renders error messages only when a message exists', () => {
    const { rerender } = render(<ErrorMessage />);

    expect(screen.queryByText('Error')).not.toBeInTheDocument();

    rerender(<ErrorMessage message="Request failed" />);

    expect(screen.getByText('Error')).toBeVisible();
    expect(screen.getByText('Request failed')).toBeVisible();
  });

  it('passes notification behavior through Carbon wrappers', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onClose = jest.fn();
    const onAction = jest.fn();
    const onLink = jest.fn();

    render(
      <>
        <Notification
          kind="success"
          title="Saved"
          subtitle="Configuration updated"
          onClose={onClose}
        />
        <ActionNotification
          kind="warning"
          title="Review"
          actionButtonLabel="Open"
          onActionButtonClick={onAction}
        />
        <LinkNotification
          kind="info"
          title="Docs"
          linkText="Read docs"
          onLinkClick={onLink}
        />
      </>,
    );

    expect(screen.getByText('Saved').closest('section')).toHaveAttribute(
      'data-kind',
      'success',
    );
    expect(screen.getByText('Saved').closest('section')).toHaveClass(
      '!max-w-full',
    );

    await user.click(screen.getAllByRole('button', { name: 'Close' })[0]);
    await user.click(screen.getByRole('button', { name: 'Open' }));
    await user.click(screen.getByRole('button', { name: 'Read docs' }));

    expect(onClose).toHaveBeenCalledTimes(1);
    expect(onAction).toHaveBeenCalledTimes(1);
    expect(onLink).toHaveBeenCalledTimes(1);
  });

  it('passes toast notification options and close callbacks', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onClose = jest.fn();
    const onCloseButtonClick = jest.fn();

    render(
      <ToastNotification
        kind="error"
        title="Save failed"
        subtitle="Try again"
        caption="Now"
        timeout={3000}
        onClose={onClose}
        onCloseButtonClick={onCloseButtonClick}
      />,
    );

    const toast = screen.getByRole('status', { name: 'notification' });
    expect(toast).toHaveAttribute('data-kind', 'error');
    expect(toast).toHaveAttribute('data-timeout', '3000');
    expect(screen.getByText('Try again')).toBeVisible();
    expect(screen.getByText('Now')).toBeVisible();

    await user.click(screen.getByRole('button', { name: 'Close toast' }));
    await user.click(
      screen.getByRole('button', { name: 'Close toast button' }),
    );

    expect(onClose).toHaveBeenCalledTimes(1);
    expect(onCloseButtonClick).toHaveBeenCalledTimes(1);
  });

  it('renders loading states with Carbon loading props', () => {
    const { container } = render(
      <>
        <PageLoading className="page-wait" />
        <Spinner size="xs" className="tiny-spinner" />
      </>,
    );

    const statuses = screen.getAllByRole('status', { name: 'Loading' });
    expect(container.querySelector('.page-wait')).toBeInTheDocument();
    expect(statuses[0]).toHaveAttribute('data-with-overlay', 'false');
    expect(statuses[1]).toHaveClass('tiny-spinner');
    expect(statuses[1]).toHaveAttribute('data-small', 'true');
  });

  it('renders section, page, and line loaders with caller props preserved', () => {
    const { container } = render(
      <>
        <AnimatedLine animate="infinite" className="line-shell" />
        <PageLoader data-testid="page-loader" className="page-shell" />
        <SectionLoader data-testid="section-loader" className="section-shell" />
      </>,
    );

    expect(container.querySelector('.line-shell > div')).toHaveStyle(
      'animation: fill 2s linear infinite',
    );
    expect(screen.getByTestId('page-loader')).toHaveClass('page-shell');
    expect(screen.getByTestId('section-loader')).toHaveClass('section-shell');
  });

  it('uses store loading state for the global line loader', () => {
    const { rerender } = render(<LineLoader data-testid="line-loader" />);

    expect(screen.getByTestId('line-loader').firstElementChild).toHaveStyle(
      'animation: fill 2s linear',
    );

    mockLoading = true;
    rerender(<LineLoader data-testid="line-loader" />);

    expect(screen.getByTestId('line-loader').firstElementChild).toHaveStyle(
      'animation: fill 2s linear infinite',
    );
  });

  it('selects global loader style from the store', () => {
    const { rerender } = render(<Loader />);

    expect(document.querySelector('.overflow-hidden')).toBeInTheDocument();

    mockLoadingType = 'overlay';
    rerender(<Loader />);

    expect(document.querySelector('.backdrop-blur-xs')).toBeInTheDocument();
  });

  it('maps icon indicator states and explicit props', () => {
    expect(recordStateToIconIndicator.RECORD_CONNECTED).toEqual({
      kind: 'succeeded',
      label: 'Connected',
    });

    const { rerender } = render(
      <CarbonIconIndicator state="RECORD_FAILED" size={20} />,
    );

    expect(screen.getByText('Failed')).toHaveAttribute(
      'data-icon-kind',
      'failed',
    );
    expect(screen.getByText('Failed')).toHaveAttribute('data-size', '20');

    rerender(
      <CarbonIconIndicator
        align="bottom"
        iconDescription="Health"
        kind="informative"
        label="Healthy"
      />,
    );

    expect(screen.getByText('Healthy')).toHaveAttribute(
      'data-icon-kind',
      'informative',
    );
    expect(screen.getByText('Healthy')).toHaveAttribute(
      'data-icon-description',
      'Health',
    );

    rerender(<CarbonIconIndicator state="UNEXPECTED" />);

    expect(screen.getByText('Unknown')).toHaveAttribute(
      'data-icon-kind',
      'unknown',
    );

    rerender(<CarbonIconIndicator />);

    expect(screen.getByText('Unknown')).toHaveAttribute(
      'data-icon-kind',
      'unknown',
    );
  });

  it('maps shape indicator states and explicit props', () => {
    expect(recordStateToShapeIndicator.RECORD_FAILED).toEqual({
      kind: 'failed',
      label: 'Failed',
    });

    const { rerender } = render(
      <CarbonShapeIndicator state="RECORD_ACTIVE" textSize={14} />,
    );

    expect(screen.getByText('Active')).toHaveAttribute('data-kind', 'stable');
    expect(screen.getByText('Active')).toHaveAttribute('data-text-size', '14');

    rerender(
      <CarbonShapeIndicator kind="informative" label="Running checks" />,
    );

    expect(screen.getByText('Running checks')).toHaveAttribute(
      'data-kind',
      'informative',
    );

    rerender(<CarbonShapeIndicator state="UNEXPECTED" />);

    expect(screen.getByText('Unknown')).toHaveAttribute(
      'data-kind',
      'undefined',
    );

    rerender(<CarbonShapeIndicator />);

    expect(screen.getByText('Unknown')).toHaveAttribute(
      'data-kind',
      'undefined',
    );
  });

  it('renders headless toasts and reports measured heights', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockToasts = [
      { id: 'success-toast', message: 'Saved', type: 'success' },
      { id: 'error-toast', message: 'Failed', type: 'error', height: 48 },
      { id: 'blank-toast', type: 'blank' },
    ];

    render(<Toast />);

    expect(screen.getByText('Saved').closest('section')).toHaveAttribute(
      'data-kind',
      'success',
    );
    expect(screen.getByText('Failed').closest('section')).toHaveAttribute(
      'data-kind',
      'error',
    );

    const wrapper = screen.getByText('Saved').closest('.absolute');
    expect(wrapper).toBeInTheDocument();
    await user.hover(wrapper!);
    await user.unhover(wrapper!);

    expect(mockStartPause).toHaveBeenCalledTimes(1);
    expect(mockEndPause).toHaveBeenCalledTimes(1);
    expect(mockUpdateHeight).toHaveBeenCalledWith(
      'success-toast',
      expect.any(Number),
    );
    expect(mockUpdateHeight).toHaveBeenCalledWith(
      'blank-toast',
      expect.any(Number),
    );
    expect(mockUpdateHeight).not.toHaveBeenCalledWith(
      'error-toast',
      expect.any(Number),
    );

    await user.click(
      screen.getAllByRole('button', { name: 'Close toast button' })[0],
    );

    expect(mockToastRemove).toHaveBeenCalledWith('success-toast');
  });

  it('uses default toast notification options', () => {
    const { rerender } = render(<ToastNotification title="Queued" />);

    expect(screen.getByText('Queued').closest('section')).toHaveAttribute(
      'data-kind',
      'info',
    );

    rerender(<ToastNotification kind="warning" title="Queued" />);

    expect(screen.getByText('Queued').closest('section')).toHaveAttribute(
      'data-kind',
      'warning',
    );
  });
});

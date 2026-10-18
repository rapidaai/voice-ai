import React from 'react';
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ConversationDirectionIndicator } from '../conversation-direction';
import { HttpStatusSpanIndicator } from '../http-status';
import { OrganizationRoleIndicator } from '../organization-role';
import { RoleIndicator } from '../role';
import { SourceIndicator } from '../source';
import { StatusIndicator } from '../status';
import { VersionIndicator } from '../version';

const makeIcon = (name: string) =>
  function MockIcon({ size, className }: any) {
    return <svg data-icon={name} data-size={size} className={className} />;
  };

jest.mock('@carbon/icons-react', () => ({
  Activity: makeIcon('activity'),
  Application: makeIcon('application'),
  Archive: makeIcon('archive'),
  ArrowDown: makeIcon('arrow-down'),
  ArrowUp: makeIcon('arrow-up'),
  Checkmark: makeIcon('checkmark'),
  CheckmarkFilled: makeIcon('checkmark-filled'),
  Close: makeIcon('close'),
  Code: makeIcon('code'),
  ConnectionSignal: makeIcon('connection-signal'),
  Copy: makeIcon('copy'),
  Debug: makeIcon('debug'),
  Edit: makeIcon('edit'),
  Email: makeIcon('email'),
  Globe: makeIcon('globe'),
  InProgress: makeIcon('in-progress'),
  LogoPython: makeIcon('python'),
  LogoReact: makeIcon('react'),
  Pending: makeIcon('pending'),
  Phone: makeIcon('phone'),
  SubtractAlt: makeIcon('subtract-alt'),
  Time: makeIcon('time'),
  UserAdmin: makeIcon('user-admin'),
  UserMultiple: makeIcon('user-multiple'),
  UserRole: makeIcon('user-role'),
  View: makeIcon('view'),
}));

jest.mock('@carbon/react', () => ({
  Tag: ({ children, renderIcon: Icon, size, type }: any) => (
    <span data-size={size} data-type={type}>
      {Icon ? <Icon /> : null}
      {children}
    </span>
  ),
  Tooltip: ({ children, label }: any) => (
    <span aria-label={label}>{children}</span>
  ),
  unstable__ShapeIndicator: ({ kind, label, textSize }: any) => (
    <span data-kind={kind} data-text-size={textSize}>
      {label}
    </span>
  ),
}));

jest.mock('@/app/components/ui/primitives', () => ({
  IconOnlyButton: ({ iconDescription, onClick, renderIcon: Icon }: any) => (
    <button type="button" aria-label={iconDescription} onClick={onClick}>
      {Icon ? <Icon /> : null}
    </button>
  ),
}));

describe('domain indicators', () => {
  const originalClipboard = navigator.clipboard;

  beforeEach(() => {
    jest.useFakeTimers();
    Object.assign(navigator, {
      clipboard: {
        writeText: jest.fn(),
      },
    });
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  afterAll(() => {
    Object.assign(navigator, { clipboard: originalClipboard });
  });

  it('renders status states with Carbon tags and icons', () => {
    const { rerender } = render(
      <StatusIndicator state="SUCCESS" size="small" />,
    );

    expect(screen.getByText('Success')).toHaveAttribute('data-type', 'green');
    expect(
      document.querySelector('[data-icon="checkmark-filled"]'),
    ).toBeInTheDocument();

    rerender(<StatusIndicator state="IN_PROGRESS" />);
    expect(screen.getByText('In progress')).toHaveAttribute(
      'data-type',
      'blue',
    );
    expect(
      document.querySelector('[data-icon="in-progress"]'),
    ).toBeInTheDocument();

    rerender(<StatusIndicator state="UNKNOWN" size="large" />);
    expect(screen.getByText('Inactive')).toHaveAttribute('data-type', 'gray');
    expect(
      document.querySelector('[data-icon="subtract-alt"]'),
    ).toBeInTheDocument();
  });

  it('renders role and organization role indicators', () => {
    render(
      <>
        <RoleIndicator role="SUPER_ADMIN" size="small" />
        <RoleIndicator role="reader" />
        <RoleIndicator role="unexpected" />
        <OrganizationRoleIndicator role="OWNER" size="large" />
        <OrganizationRoleIndicator role="admin" />
        <OrganizationRoleIndicator />
      </>,
    );

    expect(screen.getByText('Super Admin')).toHaveAttribute(
      'data-type',
      'purple',
    );
    expect(screen.getByText('Reader')).toHaveAttribute('data-type', 'cyan');
    expect(screen.getByText('User')).toHaveAttribute('data-type', 'gray');
    expect(screen.getByText('Owner')).toHaveAttribute('data-type', 'red');
    expect(screen.getByText('Admin')).toHaveAttribute('data-type', 'blue');
    expect(screen.getByText('Member')).toHaveAttribute('data-type', 'gray');
  });

  it('renders conversation direction, source, and HTTP status indicators', () => {
    render(
      <>
        <ConversationDirectionIndicator direction="inbound" />
        <ConversationDirectionIndicator direction="outbound" />
        <SourceIndicator source="react-sdk" />
        <SourceIndicator source="unknown-source" withLabel={false} />
        <HttpStatusSpanIndicator status={204} textSize={14} />
        <HttpStatusSpanIndicator status={302} />
        <HttpStatusSpanIndicator status={404} />
        <HttpStatusSpanIndicator status={503} />
        <HttpStatusSpanIndicator status={102} />
      </>,
    );

    expect(screen.getByText('Inbound').closest('[data-type]')).toHaveAttribute(
      'data-type',
      'green',
    );
    expect(screen.getByText('Outbound').closest('[data-type]')).toHaveAttribute(
      'data-type',
      'warm-gray',
    );
    expect(
      screen.getByText('React SDK').closest('[data-type]'),
    ).toHaveAttribute('data-type', 'blue');
    expect(screen.getByLabelText('Unknown')).toBeInTheDocument();
    expect(screen.getByText('204 OK')).toHaveAttribute('data-kind', 'stable');
    expect(screen.getByText('302 Redirect')).toHaveAttribute(
      'data-kind',
      'informative',
    );
    expect(screen.getByText('404 Client Error')).toHaveAttribute(
      'data-kind',
      'cautious',
    );
    expect(screen.getByText('503 Server Error')).toHaveAttribute(
      'data-kind',
      'failed',
    );
    expect(screen.getByText('102')).toHaveAttribute('data-kind', 'undefined');
  });

  it('copies version ids and shows copied state temporarily', async () => {
    const user = userEvent.setup({
      advanceTimers: jest.advanceTimersByTime,
      writeToClipboard: false,
    });
    const writeText = jest.spyOn(navigator.clipboard, 'writeText');
    render(<VersionIndicator id="version-1" />);

    await user.click(screen.getByRole('button', { name: 'Copy version' }));

    expect(writeText).toHaveBeenCalledWith('vrsn_version-1');
    expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument();

    act(() => {
      jest.advanceTimersByTime(2000);
    });

    expect(
      screen.getByRole('button', { name: 'Copy version' }),
    ).toBeInTheDocument();
  });
});

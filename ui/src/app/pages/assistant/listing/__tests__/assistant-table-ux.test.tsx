import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import SingleAssistant from '@/app/pages/assistant/listing/single-assistant';

const mockGoToAssistant = jest.fn();
const mockGoToManageAssistant = jest.fn();
const mockGoToCreateAssistantVersion = jest.fn();

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToAssistant: mockGoToAssistant,
    goToManageAssistant: mockGoToManageAssistant,
    goToCreateAssistantVersion: mockGoToCreateAssistantVersion,
  }),
}));

jest.mock('@carbon/react', () => ({
  Link: ({ children, href, className, ...props }: any) => (
    <a href={href} className={className} {...props}>
      {children}
    </a>
  ),
  TableRow: ({ children }: any) => <tr>{children}</tr>,
  TableCell: ({ children, className }: any) => (
    <td className={className}>{children}</td>
  ),
  Tag: ({ children, type, size }: any) => (
    <span data-size={size} data-type={type}>
      {children}
    </span>
  ),
}));

jest.mock('@/app/components/indicators/source', () => ({
  __esModule: true,
  default: ({ source }: any) => <span>Deployment: {source}</span>,
}));

jest.mock('@/app/components/carbon/record-status-indicator', () => ({
  RecordStatusIndicator: ({ state }: any) => <span>Status: {state}</span>,
}));

jest.mock('@/app/components/carbon/button/copy-button', () => ({
  CopyButton: ({ children }: any) => <button>Copy {children}</button>,
}));

jest.mock('@/app/components/indicators/version', () => ({
  VersionIndicator: ({ id }: any) => (
    <span>
      Version: vrsn_{id}
      <button>Copy version</button>
    </span>
  ),
}));

jest.mock('@/app/components/carbon/button', () => ({
  IconOnlyButton: ({ iconDescription, onClick }: any) => (
    <button aria-label={iconDescription} onClick={onClick}>
      {iconDescription}
    </button>
  ),
}));

jest.mock('@/utils/date', () => ({
  toHumanReadableDateTime: (timestamp: any) => `date:${timestamp.getSeconds()}`,
}));

const makeTimestamp = (seconds: number) => ({
  getSeconds: () => seconds,
  getNanos: () => 0,
});

const makeAssistant = (overrides: Record<string, any> = {}) =>
  ({
    getId: () => 'assistant-1',
    getName: () => 'Support assistant',
    getStatus: () => 'ACTIVE',
    getAssistantprovider: () => 'agent_flow',
    getAssistantproviderid: () => 'provider-version-1',
    getAssistantprovidermodel: () => null,
    getAssistantprovideragentkit: () => null,
    getAssistantproviderwebsocket: () => null,
    getAssistantprovideragentflow: () => null,
    getAssistanttag: () => ({
      getTagList: () => ['support', 'production', 'billing'],
    }),
    getApideployment: () => ({}),
    getDebuggerdeployment: () => ({}),
    getWebplugindeployment: () => null,
    getPhonedeployment: () => null,
    getUpdateddate: () => makeTimestamp(400),
    getCreateduser: () => ({ getName: () => 'Prashant' }),
    ...overrides,
  }) as any;

const renderAssistantRow = (assistant = makeAssistant()) =>
  render(
    <table>
      <tbody>
        <SingleAssistant assistant={assistant} />
      </tbody>
    </table>,
  );

describe('assistant listing table UX', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders assistant resource-console cells with compact Carbon metadata', () => {
    renderAssistantRow();

    expect(
      screen.getByRole('link', { name: /Support assistant/i }),
    ).toHaveAttribute('href', '/deployment/assistant/assistant-1');
    expect(screen.getByText('assistant-1')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Copy assistant-1' }),
    ).toBeInTheDocument();
    expect(screen.getByText('agent flow')).toBeInTheDocument();
    expect(
      screen.getByText(/Version: vrsn_provider-version-1/),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Copy version' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Status: ACTIVE')).toBeInTheDocument();
    expect(screen.getByText('Deployment: react-sdk')).toBeInTheDocument();
    expect(screen.getByText('Deployment: debugger')).toBeInTheDocument();
    expect(screen.getByText('support')).toBeInTheDocument();
    expect(screen.getByText('production')).toBeInTheDocument();
    expect(screen.queryByText('billing')).not.toBeInTheDocument();
    expect(screen.getByText('+1')).toBeInTheDocument();
    expect(screen.getByText('date:400')).toBeInTheDocument();
    expect(screen.getByText('Prashant')).toBeInTheDocument();
  });

  it('keeps assistants without deployments actionable from the table row', () => {
    renderAssistantRow(
      makeAssistant({
        getApideployment: () => null,
        getDebuggerdeployment: () => null,
        getWebplugindeployment: () => null,
        getPhonedeployment: () => null,
        getAssistanttag: () => null,
        getStatus: () => '',
        getAssistantprovider: () => '',
        getAssistantprovidermodel: () => ({}),
        getUpdateddate: () => undefined,
        getCreateduser: () => undefined,
      }),
    );

    expect(screen.getByText('Not configured')).toBeInTheDocument();
    expect(screen.getByText('prompt')).toBeInTheDocument();
    expect(screen.getAllByText('-')).toHaveLength(4);

    fireEvent.click(screen.getByRole('button', { name: 'Set up deployment' }));
    expect(mockGoToManageAssistant).toHaveBeenCalledWith('assistant-1');
  });

  it('uses flat icon actions for common assistant workflows', () => {
    renderAssistantRow();

    fireEvent.click(screen.getByRole('button', { name: 'View detail' }));
    expect(mockGoToAssistant).toHaveBeenCalledWith('assistant-1');

    fireEvent.click(screen.getByRole('button', { name: 'Manage deployments' }));
    expect(mockGoToManageAssistant).toHaveBeenCalledWith('assistant-1');

    fireEvent.click(screen.getByRole('button', { name: 'Create new version' }));
    expect(mockGoToCreateAssistantVersion).toHaveBeenCalledWith('assistant-1');
  });
});

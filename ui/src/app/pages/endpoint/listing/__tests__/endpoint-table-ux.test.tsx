import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { SingleEndpoint } from '@/app/pages/endpoint/listing/single-endpoint';
import { useEndpointPageStore } from '@/hooks/use-endpoint-page-store';

const mockNavigate = jest.fn();
const mockWriteText = jest.fn();

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
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

jest.mock('@/app/components/carbon/record-status-indicator', () => ({
  RecordStatusIndicator: ({ state }: any) => <span>Status: {state}</span>,
}));

jest.mock('@/app/components/carbon/provider-tag', () => ({
  ProviderTag: ({ provider }: any) => <span>Provider: {provider}</span>,
}));

jest.mock('@/app/components/indicators/version', () => ({
  VersionIndicator: ({ id }: any) => <span>Version: vrsn_{id}</span>,
}));

jest.mock('@/app/components/carbon/button/copy-button', () => ({
  CopyButton: ({ children }: any) => <button>Copy {children}</button>,
}));

jest.mock('@/app/components/carbon/button', () => ({
  IconOnlyButton: ({ iconDescription, onClick }: any) => (
    <button aria-label={iconDescription} onClick={onClick}>
      {iconDescription}
    </button>
  ),
}));

const defaultColumns = useEndpointPageStore.getState().columns;

const makeTimestamp = (date: Date) => ({
  toDate: () => date,
  getSeconds: () => Math.floor(date.getTime() / 1000),
  getNanos: () => 0,
});

const makeEndpoint = ({
  errorCount = '0',
  totalCount = '100',
}: {
  errorCount?: string;
  totalCount?: string;
} = {}) =>
  ({
    getId: () => 'endpoint-1',
    getName: () => 'Production completion',
    getStatus: () => 'ACTIVE',
    getEndpointtag: () => ({
      getTagList: () => ['production', 'finance', 'chat', 'premium'],
    }),
    getEndpointprovidermodel: () => ({
      getId: () => 'model-1',
      getStatus: () => 'DEPLOYED',
      getModelprovidername: () => 'openai',
      getCreateduser: () => ({ getName: () => 'Prashant' }),
    }),
    getEndpointanalytics: () => ({
      getCount: () => totalCount,
      getTotaltoken: () => '123456',
      getErrorcount: () => errorCount,
      getTotalinputcost: () => 1.25,
      getTotaloutputcost: () => 2.75,
      getP50latency: () => 120_000_000,
      getP99latency: () => 990_000_000,
      getLastactivity: () => makeTimestamp(new Date()),
    }),
  }) as any;

const renderEndpointRow = (endpoint = makeEndpoint()) =>
  render(
    <table>
      <tbody>
        <SingleEndpoint endpoint={endpoint} />
      </tbody>
    </table>,
  );

describe('endpoint listing table UX', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText,
      },
    });
    useEndpointPageStore.setState({
      columns: defaultColumns,
    } as any);
  });

  it('defaults to platform resource-console columns', () => {
    expect(
      useEndpointPageStore
        .getState()
        .columns.filter(column => column.visible)
        .map(column => column.name),
    ).toEqual([
      'Endpoint',
      'Endpoint ID',
      'Status',
      'Provider',
      'Version',
      'Tags',
      'Runs (7D)',
      'Error rate (7D)',
      'P50 latency',
      'P99 latency',
      'Cost',
      'Last activity',
      'Owner',
      'Actions',
    ]);
  });

  it('renders restrained Carbon-style cells without grouping metrics into custom blocks', () => {
    renderEndpointRow();

    expect(
      screen.getByRole('link', { name: /Production completion/i }),
    ).toHaveAttribute('href', '/deployment/endpoint/endpoint-1');
    expect(screen.getByText('endpoint-1')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Copy endpoint-1' }),
    ).toBeInTheDocument();
    expect(screen.getByText('production')).toBeInTheDocument();
    expect(screen.getByText('finance')).toBeInTheDocument();
    expect(screen.queryByText('chat')).not.toBeInTheDocument();
    expect(screen.getByText('+2')).toBeInTheDocument();
    expect(screen.getByText('Status: ACTIVE')).toBeInTheDocument();
    expect(screen.getByText('Provider: openai')).toBeInTheDocument();
    expect(screen.getByText('Version: vrsn_model-1')).toBeInTheDocument();
    expect(screen.getByText('100')).toBeInTheDocument();
    expect(screen.queryByText('runs')).not.toBeInTheDocument();
    expect(screen.queryByText('123,456 tokens')).not.toBeInTheDocument();
    expect(screen.getByText('0%')).toHaveClass('text-green-600');
    expect(screen.queryByText('P50')).not.toBeInTheDocument();
    expect(screen.getByText('120ms')).toBeInTheDocument();
    expect(screen.queryByText('P99')).not.toBeInTheDocument();
    expect(screen.getByText('990ms')).toBeInTheDocument();
    expect(screen.getByText('$4.0000')).toBeInTheDocument();
    expect(screen.getByText('Prashant')).toBeInTheDocument();
  });

  it('uses severity-aware reliability styling instead of making healthy endpoints red', () => {
    const { rerender } = render(
      <table>
        <tbody>
          <SingleEndpoint endpoint={makeEndpoint({ errorCount: '0' })} />
        </tbody>
      </table>,
    );

    expect(screen.getByText('0%')).toHaveClass('text-green-600');
    expect(screen.getByText('0%')).not.toHaveClass('text-red-600');

    rerender(
      <table>
        <tbody>
          <SingleEndpoint
            endpoint={makeEndpoint({ errorCount: '12', totalCount: '100' })}
          />
        </tbody>
      </table>,
    );

    expect(screen.getByText('12%')).toHaveClass('text-red-600');
  });

  it('uses flat icon actions like the activity tables', () => {
    renderEndpointRow();

    fireEvent.click(screen.getByRole('button', { name: 'View detail' }));
    expect(mockNavigate).toHaveBeenCalledWith(
      '/deployment/endpoint/endpoint-1',
    );

    fireEvent.click(screen.getByRole('button', { name: 'Create new version' }));
    expect(mockNavigate).toHaveBeenCalledWith(
      '/deployment/endpoint/endpoint-1/create-endpoint-version',
    );

    expect(
      screen.queryByLabelText('Actions for Production completion'),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('View traces')).not.toBeInTheDocument();
    expect(screen.queryByText('Manage versions')).not.toBeInTheDocument();
  });
});

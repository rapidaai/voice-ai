import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { AssistantPage } from '@/app/pages/assistant/listing';
import { EndpointPage } from '@/app/pages/endpoint/listing';

const mockNavigate = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockAssistantGetAll = jest.fn();
const mockEndpointGetAll = jest.fn();
const mockAssistantAddCriteria = jest.fn();
const mockEndpointAddCriteria = jest.fn();
const mockAssistantSetCriterias = jest.fn();
const mockEndpointSetCriterias = jest.fn();
const mockAssistantSetPage = jest.fn();
const mockAssistantSetPageSize = jest.fn();
const mockEndpointSetPage = jest.fn();
const mockEndpointSetPageSize = jest.fn();

let mockLoading = false;
let mockAssistantState: any;
let mockEndpointState: any;

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
  useSearchParams: () => [new URLSearchParams()],
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/hooks/use-assistant-page-store', () => ({
  useAssistantPageStore: () => mockAssistantState,
}));

jest.mock('@/hooks', () => ({
  useEndpointPageStore: () => mockEndpointState,
  useRapidaStore: () => ({
    loading: mockLoading,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/app/components/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/app/components/carbon/loading', () => ({
  PageLoading: () => <div>Loading page</div>,
}));

jest.mock('@/app/components/carbon/modal', () => ({
  Modal: ({ children, open }: any) =>
    open ? <div role="dialog">{children}</div> : null,
  ModalBody: ({ children }: any) => <div>{children}</div>,
  ModalHeader: ({ title }: any) => <h2>{title}</h2>,
}));

jest.mock('@/app/components/carbon/pagination', () => ({
  Pagination: ({ onChange, pageSize }: any) => (
    <button onClick={() => onChange({ page: 2, pageSize })}>Next page</button>
  ),
}));

jest.mock('@/app/pages/assistant/listing/single-assistant', () => ({
  __esModule: true,
  default: ({ assistant }: any) => (
    <tr>
      <td>{assistant.getName()}</td>
    </tr>
  ),
}));

jest.mock('@/app/pages/endpoint/listing/single-endpoint', () => ({
  SingleEndpoint: ({ endpoint }: any) => (
    <tr>
      <td>{endpoint.getName()}</td>
    </tr>
  ),
}));

jest.mock('@carbon/react', () => {
  const React = require('react');
  const Div = ({ children }: any) => <div>{children}</div>;
  return {
    Button: ({
      children,
      hasIconOnly: _hasIconOnly,
      renderIcon: _renderIcon,
      iconDescription,
      tooltipPosition: _tooltipPosition,
      ...props
    }: any) => (
      <button aria-label={iconDescription} {...props}>
        {children || iconDescription}
      </button>
    ),
    ButtonSkeleton: ({ className }: any) => (
      <div className={className}>Button loading</div>
    ),
    ClickableTile: ({ children, onClick, className }: any) => (
      <button className={className} onClick={onClick}>
        {children}
      </button>
    ),
    IconButton: ({
      children,
      label,
      renderIcon: _renderIcon,
      ...props
    }: any) => (
      <button aria-label={label} {...props}>
        {children || label}
      </button>
    ),
    Link: ({ children, href, ...props }: any) => (
      <a href={href} {...props}>
        {children}
      </a>
    ),
    Table: ({ children }: any) => <table>{children}</table>,
    TableHead: ({ children }: any) => <thead>{children}</thead>,
    TableRow: ({ children }: any) => <tr>{children}</tr>,
    TableHeader: ({ children, className }: any) => (
      <th className={className}>{children}</th>
    ),
    TableBody: ({ children }: any) => <tbody>{children}</tbody>,
    TableCell: ({ children, className }: any) => (
      <td className={className}>{children}</td>
    ),
    DataTableSkeleton: ({ className, headers, rowCount }: any) => (
      <div className={className}>
        Loading table {headers?.length} columns {rowCount} rows
      </div>
    ),
    SkeletonPlaceholder: ({ className }: any) => (
      <span className={className} data-testid="skeleton-placeholder" />
    ),
    SkeletonText: ({ className, width }: any) => (
      <span
        className={className}
        data-testid="skeleton-text"
        style={{ width }}
      />
    ),
    TableToolbar: Div,
    TableToolbarContent: Div,
    TableToolbarSearch: ({ placeholder }: any) => (
      <input placeholder={placeholder} />
    ),
  };
});

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
}));

const makeAssistant = (id: string, name: string) =>
  ({
    getId: () => id,
    getName: () => name,
  }) as any;

const makeEndpoint = (id: string, name: string) =>
  ({
    getId: () => id,
    getName: () => name,
  }) as any;

function resetState() {
  mockLoading = false;
  mockAssistantState = {
    assistants: [],
    criteria: [],
    page: 1,
    pageSize: 20,
    totalCount: 0,
    addCriteria: mockAssistantAddCriteria,
    setCriterias: mockAssistantSetCriterias,
    onGetAllAssistant: mockAssistantGetAll,
    setPage: mockAssistantSetPage,
    setPageSize: mockAssistantSetPageSize,
  };
  mockEndpointState = {
    endpoints: [],
    criteria: [],
    columns: [],
    page: 1,
    pageSize: 20,
    totalCount: 0,
    addCriteria: mockEndpointAddCriteria,
    setCriterias: mockEndpointSetCriterias,
    onGetAllEndpoint: mockEndpointGetAll,
    setPage: mockEndpointSetPage,
    setPageSize: mockEndpointSetPageSize,
  };
}

describe('listing page create CTAs', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    resetState();
  });

  it('uses aligned primary create actions on assistant and endpoint listings', () => {
    mockAssistantState.assistants = [makeAssistant('a-1', 'Support assistant')];
    mockAssistantState.totalCount = 4;
    mockEndpointState.endpoints = [makeEndpoint('e-1', 'Support endpoint')];
    mockEndpointState.totalCount = 5;

    const { unmount } = render(<AssistantPage />);

    expect(
      screen.getAllByRole('button', { name: 'Create new assistant' }),
    ).toHaveLength(1);
    expect(screen.getByText('1/4')).toBeInTheDocument();
    expect(screen.queryByText('Create new Assistant')).not.toBeInTheDocument();

    unmount();
    render(<EndpointPage />);

    expect(
      screen.getAllByRole('button', { name: 'Create new endpoint' }),
    ).toHaveLength(1);
    expect(screen.getByText('1/5')).toBeInTheDocument();
    expect(screen.queryByText('Add new endpoint')).not.toBeInTheDocument();
  });

  it('keeps empty-state create copy and filtered no-results copy consistent', () => {
    mockAssistantState.criteria = [{ key: 'name', value: 'missing' }];
    mockEndpointState.criteria = [{ key: 'name', value: 'missing' }];

    const { unmount } = render(<AssistantPage />);

    expect(screen.getByText('No assistants found')).toBeInTheDocument();
    expect(
      screen.getByText('No assistants match your current filters.'),
    ).toBeInTheDocument();
    expect(
      screen.getAllByRole('button', { name: 'Create new assistant' }),
    ).toHaveLength(2);

    unmount();
    render(<EndpointPage />);

    expect(screen.getByText('No endpoints found')).toBeInTheDocument();
    expect(
      screen.getByText('No endpoints match your current filters.'),
    ).toBeInTheDocument();
    expect(
      screen.getAllByRole('button', { name: 'Create new endpoint' }),
    ).toHaveLength(2);
  });

  it('keeps the endpoint create action visible while the listing reloads', () => {
    mockLoading = true;

    render(<EndpointPage />);

    expect(screen.getByText('Loading page')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Create new endpoint' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Button loading')).not.toBeInTheDocument();

    fireEvent.click(
      screen.getByRole('button', { name: 'Create new endpoint' }),
    );
    expect(mockNavigate).toHaveBeenCalledWith(
      '/deployment/endpoint/create-endpoint',
    );
  });

  it('uses a table skeleton while the assistant listing reloads', () => {
    mockLoading = true;
    mockAssistantState.pageSize = 20;

    render(<AssistantPage />);

    expect(screen.getByRole('columnheader', { name: 'Assistant' })).toHaveClass(
      'min-w-56',
      'whitespace-nowrap',
    );
    expect(screen.getByRole('columnheader', { name: 'Actions' })).toHaveClass(
      'min-w-28',
      'whitespace-nowrap',
    );
    expect(screen.getAllByTestId('skeleton-text')).toHaveLength(60);
    expect(screen.getAllByTestId('skeleton-placeholder')).toHaveLength(100);
    expect(
      screen.getByRole('button', { name: 'Create new assistant' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Loading page')).not.toBeInTheDocument();
  });

  it('keeps endpoint metric headers on one line with fixed minimum widths', () => {
    mockEndpointState.endpoints = [makeEndpoint('e-1', 'Support endpoint')];
    mockEndpointState.columns = [
      { name: 'Runs (7D)', key: 'getCount', visible: true },
      { name: 'Error rate (7D)', key: 'getErrorRate', visible: true },
      { name: 'P50 latency', key: 'getP50', visible: true },
      { name: 'P99 latency', key: 'getP99', visible: true },
      { name: 'Cost', key: 'getCost', visible: true },
    ];

    render(<EndpointPage />);

    expect(screen.getByRole('columnheader', { name: 'Runs (7D)' })).toHaveClass(
      'min-w-28',
      'whitespace-nowrap',
    );
    expect(
      screen.getByRole('columnheader', { name: 'Error rate (7D)' }),
    ).toHaveClass('min-w-36', 'whitespace-nowrap');
    expect(
      screen.getByRole('columnheader', { name: 'P50 latency' }),
    ).toHaveClass('min-w-32', 'whitespace-nowrap');
    expect(
      screen.getByRole('columnheader', { name: 'P99 latency' }),
    ).toHaveClass('min-w-32', 'whitespace-nowrap');
    expect(screen.getByRole('columnheader', { name: 'Cost' })).toHaveClass(
      'min-w-28',
      'whitespace-nowrap',
    );
  });

  it('keeps assistant resource table headers on one line with fixed minimum widths', () => {
    mockAssistantState.assistants = [makeAssistant('a-1', 'Support assistant')];
    mockAssistantState.totalCount = 1;

    render(<AssistantPage />);

    expect(screen.getByRole('columnheader', { name: 'Assistant' })).toHaveClass(
      'min-w-56',
      'whitespace-nowrap',
    );
    expect(
      screen.getByRole('columnheader', { name: 'Assistant ID' }),
    ).toHaveClass('min-w-64', 'whitespace-nowrap');
    expect(screen.getByRole('columnheader', { name: 'Provider' })).toHaveClass(
      'min-w-36',
      'whitespace-nowrap',
    );
    expect(screen.getByRole('columnheader', { name: 'Version' })).toHaveClass(
      'min-w-44',
      'whitespace-nowrap',
    );
    expect(
      screen.getByRole('columnheader', { name: 'Deployments' }),
    ).toHaveClass('min-w-48', 'whitespace-nowrap');
    expect(
      screen.queryByRole('columnheader', { name: 'Sessions' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('columnheader', { name: 'Users' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('columnheader', { name: 'Last activity' }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Actions' })).toHaveClass(
      'min-w-28',
      'whitespace-nowrap',
    );

    const assistantHeaders = screen
      .getAllByRole('columnheader')
      .map(header => header.textContent);
    expect(assistantHeaders.indexOf('Actions')).toBeLessThan(
      assistantHeaders.indexOf('Tags'),
    );
  });
});

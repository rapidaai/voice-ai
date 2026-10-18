import React from 'react';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { EndpointOverviewPage } from '../overview-page';

const mockNavigate = jest.fn();
const mockListEndpointLogs = jest.fn();

jest.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/clients/endpoint.client', () => ({
  listEndpointLogs: (params: unknown) => mockListEndpointLogs(params),
}));

jest.mock('recharts', () => {
  const Container = ({ children }: any) => <div>{children}</div>;
  const Null = () => null;
  return {
    Area: Null,
    AreaChart: () => <div />,
    CartesianGrid: Null,
    ResponsiveContainer: Container,
    Tooltip: Null,
    XAxis: Null,
    YAxis: Null,
  };
});

const endpointLog = (props: {
  id: string;
  source: string;
  status: string;
  latency: string;
  tokens: string;
}) => ({
  getId: () => props.id,
  getSource: () => props.source,
  getStatus: () => props.status,
  getTimetaken: () => props.latency,
  getMetricsList: () => [
    {
      getName: () => 'agent_total_token',
      getValue: () => props.tokens,
    },
  ],
  getCreateddate: () => ({
    getSeconds: () => Math.floor(Date.parse('2026-09-29T10:00:00.000Z') / 1000),
    getNanos: () => 0,
  }),
});

const recentLogs = [
  endpointLog({
    id: 'log-3',
    source: 'playground',
    status: 'SUCCESS',
    latency: '140000000',
    tokens: '21',
  }),
  endpointLog({
    id: 'log-2',
    source: 'api',
    status: 'FAILED',
    latency: '210000000',
    tokens: '18',
  }),
  endpointLog({
    id: 'log-1',
    source: 'assistant',
    status: 'SUCCESS',
    latency: '95000000',
    tokens: '35',
  }),
] as any[];

const successfulLogResponse = {
  getSuccess: () => true,
  getDataList: () => recentLogs,
  getPaginated: () => ({ getTotalitem: () => 128 }),
};

const currentEndpoint = {
  getId: () => 'endpoint-1',
  getName: () => 'Customer support analysis',
  getStatus: () => 'Active',
} as any;

const currentEndpointProviderModel = {
  getId: () => 'provider-model-1',
  getModelprovidermodelname: () => 'gpt-4o-mini',
  getModelprovidername: () => 'OpenAI',
} as any;

describe('EndpointOverviewPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockListEndpointLogs.mockImplementation(({ callback }) =>
      callback(null, successfulLogResponse),
    );
  });

  it('builds an activity dashboard from endpoint logs', async () => {
    render(
      <EndpointOverviewPage
        currentEndpoint={currentEndpoint}
        currentEndpointProviderModel={currentEndpointProviderModel}
      />,
    );

    expect(screen.getByTestId('endpoint-page-header')).toHaveTextContent(
      'Customer support analysis',
    );
    expect(screen.getByText('Endpoints')).toHaveAttribute(
      'href',
      '/deployment/endpoint',
    );
    expect(
      await screen.findByRole('heading', { name: 'Endpoint activity' }),
    ).toBeInTheDocument();
    expect(screen.getAllByText('128')).toHaveLength(2);
    expect(screen.getByText('67')).toBeInTheDocument();
    expect(screen.getByText('140')).toBeInTheDocument();
    expect(screen.getByText('74')).toBeInTheDocument();
    expect(screen.getByText('Request details')).toBeInTheDocument();
    expect(screen.getByText('148')).toBeInTheDocument();
    expect(screen.getByText('Last 30 days')).toBeInTheDocument();
    expect(screen.getByText('Off')).toBeInTheDocument();
    expect(
      screen.getByRole('img', {
        name: 'Latency for 3 recent endpoint requests',
      }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('table', { name: 'Recent endpoint requests' }),
    ).not.toBeInTheDocument();
    expect(mockListEndpointLogs).toHaveBeenCalledWith(
      expect.objectContaining({
        endpointId: 'endpoint-1',
        page: 1,
        pageSize: 20,
        criteria: expect.arrayContaining([
          expect.objectContaining({ key: 'created_date', logic: '<=' }),
          expect.objectContaining({ key: 'created_date', logic: '>=' }),
        ]),
        auth: {
          userId: 'user-1',
          token: 'token-1',
          projectId: 'project-1',
        },
      }),
    );
  });

  it('keeps the failure visible and retries the log request', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockListEndpointLogs
      .mockImplementationOnce(({ callback }) =>
        callback(new Error('Log service unavailable'), null),
      )
      .mockImplementationOnce(({ callback }) =>
        callback(null, successfulLogResponse),
      );

    render(
      <EndpointOverviewPage
        currentEndpoint={currentEndpoint}
        currentEndpointProviderModel={currentEndpointProviderModel}
      />,
    );

    expect(
      await screen.findByText('Endpoint activity is unavailable'),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() => expect(mockListEndpointLogs).toHaveBeenCalledTimes(2));
    expect(await screen.findAllByText('128')).toHaveLength(2);
  });

  it('shows zero when recent requests report no token usage', async () => {
    mockListEndpointLogs.mockImplementation(({ callback }) =>
      callback(null, {
        getSuccess: () => true,
        getDataList: () => [
          endpointLog({
            id: 'log-1',
            source: 'api',
            status: 'SUCCESS',
            latency: '95000000',
            tokens: '0',
          }),
        ],
        getPaginated: () => ({ getTotalitem: () => 1 }),
      }),
    );

    render(
      <EndpointOverviewPage
        currentEndpoint={currentEndpoint}
        currentEndpointProviderModel={currentEndpointProviderModel}
      />,
    );

    const usageTile = (await screen.findByRole('heading', { name: 'Usage' }))
      .parentElement?.parentElement;
    expect(usageTile).not.toBeNull();
    expect(within(usageTile!).getByText('0')).toBeInTheDocument();
  });

  it('opens the create version flow from the overview header', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <EndpointOverviewPage
        currentEndpoint={currentEndpoint}
        currentEndpointProviderModel={currentEndpointProviderModel}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: 'Create new version' }),
    );
    expect(mockNavigate).toHaveBeenCalledWith(
      '/deployment/endpoint/endpoint-1/create-endpoint-version',
    );
  });
});

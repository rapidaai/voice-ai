import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ConversationTelemetryDialog } from '../conversation-telemetry-modal';

const mockListTelemetry = jest.fn();

jest.mock('@/clients', () => ({
  listTelemetry: (...args: unknown[]) => mockListTelemetry(...args),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    token: 'token',
    authId: 'user-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/app/components/ui/primitives', () => ({
  Modal: ({ open, children, onClose }: any) =>
    open ? (
      <div>
        <button type="button" onClick={onClose}>
          Modal close
        </button>
        {children}
      </div>
    ) : null,
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalHeader: ({ title, onClose }: any) => (
    <header>
      <h2>{title}</h2>
      <button type="button" onClick={onClose}>
        Header close
      </button>
    </header>
  ),
  Pagination: ({ totalItems }: any) => (
    <nav aria-label="pagination">{totalItems} total</nav>
  ),
  Tabs: ({ children, selectedIndex, onChange, tabs }: any) => {
    const React = require('react');
    return (
      <div>
        {tabs.map((tab: string, index: number) => (
          <button key={tab} type="button" onClick={() => onChange(index)}>
            {tab}
          </button>
        ))}
        {React.Children.toArray(children)[selectedIndex]}
      </div>
    );
  },
  TextInput: ({ labelText, onChange, value }: any) => (
    <label>
      {labelText}
      <input onChange={onChange} value={value} />
    </label>
  ),
}));

jest.mock('@carbon/react', () => ({
  CodeSnippet: ({ children }: any) => <pre>{children}</pre>,
  DismissibleTag: ({ text, onClose }: any) => (
    <button type="button" onClick={onClose}>
      {text}
    </button>
  ),
  Dropdown: ({ items, onChange }: any) => (
    <button type="button" onClick={() => onChange({ selectedItem: items[0] })}>
      Select metric scope
    </button>
  ),
  Loading: () => <div role="status">Loading</div>,
  MultiSelect: ({ items, onChange }: any) => (
    <button
      type="button"
      onClick={() => onChange({ selectedItems: [items[0]] })}
    >
      Select event name
    </button>
  ),
  Table: ({ children }: any) => <table>{children}</table>,
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableCell: ({ children }: any) => <td>{children}</td>,
  TableExpandedRow: ({ children }: any) => (
    <tr>
      <td>{children}</td>
    </tr>
  ),
  TableHead: ({ children }: any) => <thead>{children}</thead>,
  TableHeader: ({ children }: any) => <th>{children}</th>,
  TableRow: ({ children, onClick }: any) => (
    <tr onClick={onClick}>{children}</tr>
  ),
  TableToolbar: ({ children }: any) => <div>{children}</div>,
  TableToolbarContent: ({ children }: any) => <div>{children}</div>,
  TableToolbarSearch: ({ placeholder, value, onChange }: any) => (
    <input
      aria-label={placeholder}
      placeholder={placeholder}
      value={value}
      onChange={event => onChange(event, event.target.value)}
    />
  ),
  Tag: ({ children }: any) => <span>{children}</span>,
}));

jest.mock('@carbon/icons-react', () => ({
  ChevronRight: () => <svg data-testid="chevron-right" />,
}));

jest.mock('@/app/components/ui/composites/table-toolbar-filter', () => ({
  TableToolbarFilter: ({ extraContent, onApply, onReset }: any) => (
    <div>
      {extraContent}
      <button type="button" onClick={onApply}>
        Apply filters
      </button>
      <button type="button" onClick={onReset}>
        Reset filters
      </button>
    </div>
  ),
}));

jest.mock('../conversation-telemetry-latency-stack-chart', () => ({
  LatencyStackChart: ({ isLoading }: any) => (
    <div data-loading={String(isLoading)} data-testid="latency-chart" />
  ),
}));

const createMap = (entries: Array<[string, string]>) => ({
  toArray: () => entries,
});

const createEventRecord = () => ({
  getOccurredat: () => ({ toDate: () => new Date('2026-01-01T00:00:00.000Z') }),
  getEvent: () => 'sip.call.lifecycle',
  getComponent: () => 'sip',
  getScope: () => 'message',
  getAttributesMap: () => createMap([['type', 'initialized']]),
  getContextMap: () => createMap([['traceId', 'trace-1']]),
  getScopeattributesMap: () =>
    createMap([
      ['conversationId', 'conversation-1'],
      ['messageId', 'message-1'],
    ]),
});

const createMetricRecord = () => ({
  getTime: () => ({ toDate: () => new Date('2026-01-01T00:00:01.000Z') }),
  getScope: () => 'message',
  getName: () => 'stt.latency_ms',
  getValue: () => '120',
  getContextMap: () => createMap([['traceId', 'trace-1']]),
  getScopeattributesMap: () =>
    createMap([
      ['conversationId', 'conversation-1'],
      ['contextId', 'message-1'],
    ]),
});

const createTelemetryResponse = (records: any[]) => ({
  getDataList: () => records,
  getPaginated: () => ({ getTotalitem: () => records.length }),
});

describe('ConversationTelemetryDialog', () => {
  beforeEach(() => {
    mockListTelemetry.mockReset();
    mockListTelemetry.mockResolvedValue({
      getDataList: () => [],
      getPaginated: () => ({ getTotalitem: () => 0 }),
    });
  });

  it('does not fetch telemetry while closed', async () => {
    render(
      <ConversationTelemetryDialog
        modalOpen={false}
        setModalOpen={jest.fn()}
        assistantId="assistant-1"
      />,
    );

    await Promise.resolve();

    expect(
      screen.queryByRole('heading', { name: 'Telemetry Events' }),
    ).not.toBeInTheDocument();
    expect(mockListTelemetry).not.toHaveBeenCalled();
  });

  it('fetches telemetry when opened and closes through modal actions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();

    render(
      <ConversationTelemetryDialog
        modalOpen
        setModalOpen={setModalOpen}
        assistantId="assistant-1"
      />,
    );

    expect(
      screen.getByRole('heading', { name: 'Telemetry Events' }),
    ).toBeInTheDocument();

    await waitFor(() => expect(mockListTelemetry).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(screen.getByText('No events found')).toBeInTheDocument(),
    );

    await user.click(screen.getByRole('button', { name: 'Modal close' }));
    await user.click(screen.getByRole('button', { name: 'Header close' }));

    expect(setModalOpen).toHaveBeenCalledWith(false);
    expect(setModalOpen).toHaveBeenCalledTimes(2);
  });

  it('renders fetched event and metric rows across tabs', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockListTelemetry.mockResolvedValue(
      createTelemetryResponse([
        {
          getEvent: () => createEventRecord(),
          getMetric: () => undefined,
        },
        {
          getEvent: () => undefined,
          getMetric: () => createMetricRecord(),
        },
      ]),
    );

    render(
      <ConversationTelemetryDialog
        modalOpen
        setModalOpen={jest.fn()}
        assistantId="assistant-1"
      />,
    );

    expect(await screen.findByText('sip.call.lifecycle')).toBeInTheDocument();
    expect(screen.getByText(/initialized/)).toBeInTheDocument();

    await user.click(screen.getByText('sip.call.lifecycle').closest('tr')!);
    expect(screen.getByText(/"traceId": "trace-1"/)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Metrics' }));
    expect(mockListTelemetry).toHaveBeenCalledTimes(1);
    expect(await screen.findByText('metric·message')).toBeInTheDocument();
    expect(screen.getByText(/stt.latency_ms/)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Latency' }));
    await waitFor(() => expect(mockListTelemetry).toHaveBeenCalledTimes(2));
    await waitFor(() =>
      expect(screen.getByTestId('latency-chart')).toHaveAttribute(
        'data-loading',
        'false',
      ),
    );
  });

  it('renders initial criteria chips and removes them', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <ConversationTelemetryDialog
        modalOpen
        setModalOpen={jest.fn()}
        assistantId="assistant-1"
        criterias={
          [
            { getKey: () => 'conversationId', getValue: () => '123' },
            { getKey: () => 'contextId', getValue: () => 'ctx-1' },
            { getKey: () => 'scope', getValue: () => 'telephony' },
          ] as any
        }
      />,
    );

    expect(
      await screen.findByText('assistantConversationId: 123'),
    ).toBeInTheDocument();
    expect(screen.getByText('messageId/contextId: ctx-1')).toBeInTheDocument();
    expect(screen.getByText('scope: telephony')).toBeInTheDocument();

    await user.click(screen.getByText('assistantConversationId: 123'));
    await user.click(screen.getByText('messageId/contextId: ctx-1'));
    await user.click(screen.getByText('scope: telephony'));

    await waitFor(() =>
      expect(screen.queryByText('scope: telephony')).not.toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(screen.queryByRole('status')).not.toBeInTheDocument(),
    );
  });

  it('applies and resets event and metric filters', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <ConversationTelemetryDialog
        modalOpen
        setModalOpen={jest.fn()}
        assistantId="assistant-1"
      />,
    );

    await screen.findByText('No events found');

    await user.click(screen.getByRole('button', { name: 'Select event name' }));
    await user.click(screen.getByLabelText('MessageID / ContextID'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('ctx-1'));
    await user.click(screen.getByLabelText('Type'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('initialized'));
    await user.click(screen.getByRole('button', { name: 'Apply filters' }));

    expect(await screen.findByText('name: session')).toBeInTheDocument();
    expect(screen.getByText('messageId/contextId: ctx-1')).toBeInTheDocument();
    expect(screen.getByText('data.type: initialized')).toBeInTheDocument();

    await user.click(screen.getByText('name: session'));
    await user.click(screen.getByText('messageId/contextId: ctx-1'));
    await user.click(screen.getByText('data.type: initialized'));
    await user.click(screen.getByRole('button', { name: 'Reset filters' }));

    await user.click(screen.getByRole('button', { name: 'Metrics' }));
    await user.click(
      screen.getByRole('button', { name: 'Select metric scope' }),
    );
    await user.click(screen.getByRole('button', { name: 'Apply filters' }));

    expect(await screen.findByText('scope: message')).toBeInTheDocument();
    await user.click(screen.getByText('scope: message'));
    await user.click(screen.getByRole('button', { name: 'Reset filters' }));
    await waitFor(() =>
      expect(screen.queryByRole('status')).not.toBeInTheDocument(),
    );
  });
});

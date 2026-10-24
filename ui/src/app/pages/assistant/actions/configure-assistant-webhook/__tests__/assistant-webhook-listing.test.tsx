import React from 'react';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ConfigureAssistantWebhookPage } from '@/app/pages/assistant/actions/configure-assistant-webhook';
import { useAssistantWebhookPageStore } from '@/stores/assistant/actions';

const mockGetAssistantWebhook = jest.fn();
const mockDeleteAssistantWebhook = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ assistantId: 'assistant-1' }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({ authId: 'u1', token: 't1', projectId: 'p1' }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToCreateAssistantWebhook: jest.fn(),
    goToEditAssistantWebhook: jest.fn(),
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/app/components/ui/feedback/loaders/section-loader', () => ({
  SectionLoader: () => <div>loading</div>,
}));

jest.mock('@/app/components/layout/sections/table-section', () => ({
  TableSection: ({ children }: any) => <div>{children}</div>,
  ScrollableTableSection: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/ui/feedback/empty-state', () => ({
  EmptyState: ({ title }: any) => <div>{title}</div>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  IconOnlyButton: ({
    children,
    renderIcon: _renderIcon,
    iconDescription: _iconDescription,
    ...props
  }: any) => <button {...props}>{children || 'icon'}</button>,
  PrimaryButton: ({ children, renderIcon: _renderIcon, ...props }: any) => (
    <button {...props}>{children || 'primary'}</button>
  ),
}));

jest.mock('@/app/components/ui/primitives/pagination', () => ({
  Pagination: ({ onChange, pageSize }: any) => (
    <div>
      <button
        data-testid="change-page"
        onClick={() => onChange({ page: 2, pageSize })}
      >
        next
      </button>
      <button
        data-testid="change-page-size"
        onClick={() => onChange({ page: 3, pageSize: 50 })}
      >
        resize
      </button>
    </div>
  ),
}));

jest.mock('@carbon/react', () => {
  const React = require('react');
  const Div = ({ children }: any) => React.createElement('div', null, children);
  return {
    Header: ({ children, ...props }: any) =>
      React.createElement('header', props, children),
    HeaderName: ({ children, prefix: _prefix, ...props }: any) =>
      React.createElement('a', props, children),
    HeaderGlobalBar: ({ children, ...props }: any) =>
      React.createElement('div', props, children),
    HeaderGlobalAction: ({ children, ...props }: any) =>
      React.createElement('button', props, children),
    Table: Div,
    TableHead: Div,
    TableRow: Div,
    TableHeader: Div,
    TableBody: Div,
    TableCell: Div,
    TableToolbar: Div,
    TableToolbarContent: Div,
    TableToolbarSearch: ({ onChange }: any) =>
      React.createElement('input', { onChange }),
    Button: ({
      children,
      hasIconOnly: _hasIconOnly,
      renderIcon: _renderIcon,
      iconDescription,
      ...props
    }: any) =>
      React.createElement(
        'button',
        { 'aria-label': iconDescription, ...props },
        children || iconDescription || 'button',
      ),
    TableBatchActions: Div,
    TableBatchAction: ({ children, ...props }: any) =>
      React.createElement('button', props, children || 'action'),
    RadioButton: ({
      onChange,
      checked,
      labelText: _labelText,
      hideLabel: _hideLabel,
      ...props
    }: any) =>
      React.createElement('input', {
        ...props,
        type: 'radio',
        checked,
        onChange,
      }),
    Breadcrumb: Div,
    BreadcrumbItem: ({ children, ...props }: any) =>
      React.createElement('a', props, children),
    ComposedModal: ({ children, open }: any) =>
      open ? React.createElement('div', { role: 'dialog' }, children) : null,
    ModalBody: Div,
    ModalFooter: Div,
    ModalHeader: ({ title }: any) => React.createElement('div', null, title),
    Link: ({ children, href, ...props }: any) =>
      React.createElement('a', { href, ...props }, children),
    Tag: ({ children }: any) => React.createElement('span', null, children),
    OverflowMenu: Div,
    OverflowMenuItem: ({ itemText, onClick, disabled }: any) =>
      React.createElement(
        'button',
        { disabled, onClick },
        itemText || 'action',
      ),
    preview__ShapeIndicator: ({ label }: any) =>
      React.createElement('span', null, label),
  };
});

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
  success: jest.fn(),
}));

const makeWebhook = (id: string) =>
  ({
    getId: () => id,
    getOptionsList: () => [
      {
        getKey: () => 'assistant_events',
        getValue: () => '["conversation.begin"]',
      },
      {
        getKey: () => 'execution_priority',
        getValue: () => '1',
      },
    ],
    getEnabled: () => true,
    getStatus: () => 'ACTIVE',
    getCreateddate: () => undefined,
  }) as any;

describe('ConfigureAssistantWebhookPage listing', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useAssistantWebhookPageStore.setState({
      webhooks: [makeWebhook('w-1')],
      page: 1,
      pageSize: 20,
      totalCount: 1,
      criteria: [],
      getAssistantWebhook: mockGetAssistantWebhook as any,
      deleteAssistantWebhook: mockDeleteAssistantWebhook as any,
    });
  });

  it('refetches when page changes from pagination', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<ConfigureAssistantWebhookPage />);

    await waitFor(() =>
      expect(mockGetAssistantWebhook).toHaveBeenCalledTimes(1),
    );
    await user.click(screen.getByTestId('change-page'));

    await waitFor(() =>
      expect(mockGetAssistantWebhook).toHaveBeenCalledTimes(2),
    );
    expect(useAssistantWebhookPageStore.getState().page).toBe(2);
  });

  it('resets to first page and refetches when page size changes', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    useAssistantWebhookPageStore.setState({ page: 4 });
    render(<ConfigureAssistantWebhookPage />);

    await waitFor(() =>
      expect(mockGetAssistantWebhook).toHaveBeenCalledTimes(1),
    );
    await user.click(screen.getByTestId('change-page-size'));

    await waitFor(() =>
      expect(mockGetAssistantWebhook).toHaveBeenCalledTimes(2),
    );
    expect(useAssistantWebhookPageStore.getState().page).toBe(1);
    expect(useAssistantWebhookPageStore.getState().pageSize).toBe(50);
  });

  it('opens confirmation before deleting webhook', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<ConfigureAssistantWebhookPage />);

    await waitFor(() =>
      expect(mockGetAssistantWebhook).toHaveBeenCalledTimes(1),
    );
    await user.click(screen.getByRole('button', { name: 'Delete' }));

    expect(screen.getByRole('dialog')).toHaveTextContent('Delete webhook?');
    expect(mockDeleteAssistantWebhook).not.toHaveBeenCalled();

    await user.click(
      within(screen.getByRole('dialog')).getByRole('button', {
        name: 'Delete',
      }),
    );

    expect(mockDeleteAssistantWebhook).toHaveBeenCalledWith(
      'assistant-1',
      'w-1',
      'p1',
      't1',
      'u1',
      expect.any(Function),
      expect.any(Function),
    );
  });
});

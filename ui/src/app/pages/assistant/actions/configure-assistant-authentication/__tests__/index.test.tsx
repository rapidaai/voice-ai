import React from 'react';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import '@testing-library/jest-dom';
import {
  ConfigureAssistantAuthenticationPage,
  CreateAssistantAuthenticationPage,
  UpdateAssistantAuthenticationPage,
} from '../index';
import { AUTH_KEY_OPTIONS_BY_TYPE } from '../shared';
import {
  CreateAssistantConfiguration,
  DeleteAssistantConfiguration,
  GetAllAssistantConfiguration,
  UpdateAssistantConfiguration,
} from '@rapidaai/react';

jest.mock('@rapidaai/react', () => {
  class AssistantConfigurationRequest {
    id = '';
    assistantId = '';
    provider = '';
    configurationType = '';
    enabled = false;
    optionsList: unknown[] = [];
    setId(v: string) {
      this.id = v;
    }
    getId() {
      return this.id;
    }
    setAssistantid(v: string) {
      this.assistantId = v;
    }
    getAssistantid() {
      return this.assistantId;
    }
    setProvider(v: string) {
      this.provider = v;
    }
    getProvider() {
      return this.provider;
    }
    setConfigurationtype(v: string) {
      this.configurationType = v;
    }
    getConfigurationtype() {
      return this.configurationType;
    }
    setEnabled(v: boolean) {
      this.enabled = v;
    }
    getEnabled() {
      return this.enabled;
    }
    setOptionsList(v: unknown[]) {
      this.optionsList = v;
    }
    getOptionsList() {
      return this.optionsList;
    }
  }
  class GetAllAssistantConfigurationRequest {
    configurationType = '';
    paginate: unknown;
    setAssistantid(_: string) {}
    setConfigurationtype(v: string) {
      this.configurationType = v;
    }
    setPaginate(v: unknown) {
      this.paginate = v;
    }
  }
  class ConnectionConfig {}
  class Paginate {
    setPage(_: number) {}
    setPagesize(_: number) {}
  }
  class Metadata {
    key = '';
    value = '';
    setKey(v: string) {
      this.key = v;
    }
    setValue(v: string) {
      this.value = v;
    }
    getKey() {
      return this.key;
    }
    getValue() {
      return this.value;
    }
  }
  return {
    ConnectionConfig,
    CreateAssistantConfigurationRequest: AssistantConfigurationRequest,
    UpdateAssistantConfigurationRequest: AssistantConfigurationRequest,
    DeleteAssistantConfigurationRequest: AssistantConfigurationRequest,
    GetAllAssistantConfigurationRequest,
    Metadata,
    Paginate,
    CreateAssistantConfiguration: jest.fn(),
    DeleteAssistantConfiguration: jest.fn(),
    GetAllAssistantConfiguration: jest.fn(),
    UpdateAssistantConfiguration: jest.fn(),
  };
});

jest.mock('react-hot-toast/headless', () => ({
  __esModule: true,
  default: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ assistantId: 'assistant-1' }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'auth-1',
    token: 'token-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goBack: jest.fn(),
    goTo: jest.fn(),
  }),
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    showDialog: (cb: () => void) => cb(),
    ConfirmDialogComponent: () => null,
  }),
}));

jest.mock('@/app/components/input-group', () => ({
  InputGroup: ({ title, children }: any) => (
    <section>
      {title ? <div>{title}</div> : null}
      {children}
    </section>
  ),
}));

jest.mock('@/app/components/conditions/source-condition-rule', () => ({
  SourceConditionRule: () => <div>conditions</div>,
}));

jest.mock('@/app/components/external-api/api-header', () => ({
  APiStringHeader: () => <div>headers</div>,
}));

jest.mock('@/app/components/carbon/notification', () => ({
  Notification: ({ subtitle }: any) => <div>{subtitle}</div>,
}));

jest.mock('@/app/components/carbon/status-indicator', () => ({
  CarbonStatusIndicator: ({ state }: any) => <span>{state}</span>,
}));

jest.mock('@/app/components/carbon/overflow-menu', () => ({
  OverflowMenu: ({ children }: any) => <div>{children}</div>,
  OverflowMenuItem: ({ itemText, onClick, disabled }: any) => (
    <button disabled={disabled} onClick={onClick}>
      {itemText}
    </button>
  ),
}));

jest.mock('@/app/components/carbon/empty-state', () => ({
  EmptyState: ({
    title,
    subtitle,
    actionButtonText,
    onActionButtonClick,
  }: any) => (
    <div>
      <div>{title}</div>
      <div>{subtitle}</div>
      {actionButtonText ? (
        <button onClick={onActionButtonClick}>{actionButtonText}</button>
      ) : null}
    </div>
  ),
}));

jest.mock('@/app/components/loader/section-loader', () => ({
  SectionLoader: () => <div>loading</div>,
}));

jest.mock('@/app/components/sections/table-section', () => ({
  TableSection: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/carbon/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: ({ id, labelText, value, onChange, hideLabel }: any) => (
    <div>
      {!hideLabel && labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <input id={id} data-testid={id} value={value ?? ''} onChange={onChange} />
    </div>
  ),
  TextArea: ({ id, value, onChange }: any) => (
    <textarea id={id} value={value ?? ''} onChange={onChange} />
  ),
}));

jest.mock('@/app/components/carbon/button', () => ({
  PrimaryButton: ({
    children,
    isLoading: _isLoading,
    renderIcon: _renderIcon,
    hasIconOnly: _hasIconOnly,
    iconDescription: _iconDescription,
    ...props
  }: any) => <button {...props}>{children}</button>,
  SecondaryButton: ({
    children,
    isLoading: _isLoading,
    renderIcon: _renderIcon,
    hasIconOnly: _hasIconOnly,
    iconDescription: _iconDescription,
    ...props
  }: any) => <button {...props}>{children}</button>,
  TertiaryButton: ({
    children,
    isLoading: _isLoading,
    renderIcon: _renderIcon,
    hasIconOnly: _hasIconOnly,
    iconDescription: _iconDescription,
    ...props
  }: any) => <button {...props}>{children}</button>,
  IconOnlyButton: ({
    children,
    isLoading: _isLoading,
    renderIcon: _renderIcon,
    hasIconOnly: _hasIconOnly,
    iconDescription,
    ...props
  }: any) => (
    <button aria-label={iconDescription || children || 'button'} {...props}>
      {children || iconDescription || 'button'}
    </button>
  ),
}));

jest.mock('@carbon/react', () => ({
  Breadcrumb: ({ children }: any) => <div>{children}</div>,
  BreadcrumbItem: ({ children }: any) => <span>{children}</span>,
  ButtonSet: ({ children }: any) => <div>{children}</div>,
  ComposedModal: ({ children, open }: any) =>
    open ? <div role="dialog">{children}</div> : null,
  ModalBody: ({ children }: any) => <div>{children}</div>,
  ModalFooter: ({ children }: any) => <div>{children}</div>,
  ModalHeader: ({ title }: any) => <div>{title}</div>,
  Table: ({ children }: any) => <table>{children}</table>,
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableCell: ({ children, ...props }: any) => <td {...props}>{children}</td>,
  TableHead: ({ children }: any) => <thead>{children}</thead>,
  TableHeader: ({ children }: any) => <th>{children}</th>,
  TableRow: ({ children }: any) => <tr>{children}</tr>,
  TableToolbar: ({ children }: any) => <div>{children}</div>,
  TableToolbarContent: ({ children }: any) => <div>{children}</div>,
  OverflowMenu: ({ children }: any) => <div>{children}</div>,
  OverflowMenuItem: ({ itemText, onClick, disabled }: any) => (
    <button disabled={disabled} onClick={onClick}>
      {itemText}
    </button>
  ),
  Slider: ({ id, value, onChange }: any) => (
    <input
      id={id}
      type="range"
      value={value}
      onChange={e => onChange?.({ value: Number(e.target.value) })}
    />
  ),
  Select: ({ id, labelText, value, onChange, children, hideLabel }: any) => (
    <div>
      {!hideLabel && labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <select id={id} data-testid={id} value={value} onChange={onChange}>
        {children}
      </select>
    </div>
  ),
  SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
  Button: ({
    children,
    iconDescription,
    hasIconOnly: _hasIconOnly,
    renderIcon: _renderIcon,
    ...props
  }: any) => (
    <button aria-label={iconDescription || children || 'button'} {...props}>
      {children || 'button'}
    </button>
  ),
  Tooltip: ({ children }: any) => <span>{children}</span>,
}));

jest.mock('@/app/components/carbon/record-status-indicator', () => ({
  RecordStatusIndicator: ({ state }: any) => (
    <span>
      {state === 'ACTIVE'
        ? 'Active'
        : state === 'INACTIVE'
          ? 'Inactive'
          : 'Draft'}
    </span>
  ),
}));

jest.mock('@/app/components/carbon/url-table-cell', () => ({
  UrlTableCell: ({ url }: any) => <td>{url || '-'}</td>,
}));

describe('CreateAssistantAuthenticationPage', () => {
  it('uses the canonical assistant phone authentication key', () => {
    expect(
      AUTH_KEY_OPTIONS_BY_TYPE.client.find(
        option => option.name === 'Assistant Phone',
      ),
    ).toEqual({ value: 'assistant_phone', name: 'Assistant Phone' });
    expect(
      AUTH_KEY_OPTIONS_BY_TYPE.client.some(
        option => option.value === 'assistantPhone',
      ),
    ).toBe(false);
  });

  const makeOption = (key: string, value: string) => ({
    getKey: () => key,
    getValue: () => value,
  });

  const getSuccessLoadResponse = (failBehavior = 'BLOCK', enabled = true) => ({
    getSuccess: () => true,
    getDataList: () => [
      {
        getId: () => 'auth-config-1',
        getProvider: () => 'http',
        getEnabled: () => enabled,
        getStatus: () => (enabled ? 'ACTIVE' : 'INACTIVE'),
        getCreateddate: () => undefined,
        getOptionsList: () => [
          makeOption('http_method', 'POST'),
          makeOption('http_url', 'https://auth.example.com/loaded'),
          makeOption('http_headers', '{}'),
          makeOption(
            'http_body',
            '{"assistant.id":"assistantId","client.phone":"clientPhone"}',
          ),
          makeOption('fail_behavior', failBehavior),
          makeOption('timeout_ms', '5000'),
          makeOption(
            'authentication.condition',
            '[{"key":"source","condition":"=","value":"all"}]',
          ),
        ],
      },
    ],
  });

  beforeEach(() => {
    jest.clearAllMocks();
    (GetAllAssistantConfiguration as jest.Mock).mockResolvedValue(
      getSuccessLoadResponse(),
    );
    (CreateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({}),
    });
    (UpdateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({}),
    });
    (DeleteAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({}),
    });
  });

  const waitUntilReady = async () => {
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Save authentication' }),
      ).not.toBeDisabled(),
    );
  };

  it('keeps save enabled and validates on click', async () => {
    render(<CreateAssistantAuthenticationPage />);
    await waitUntilReady();
    expect(GetAllAssistantConfiguration).not.toHaveBeenCalled();

    const saveButton = screen.getByRole('button', {
      name: 'Save authentication',
    });
    expect(saveButton).not.toBeDisabled();

    fireEvent.click(saveButton);

    expect(
      screen.getByText('Please provide a server URL for authentication.'),
    ).toBeInTheDocument();
    expect(CreateAssistantConfiguration).not.toHaveBeenCalled();
  });

  it('supports add and edit for authentication parameter mapping', async () => {
    render(<CreateAssistantAuthenticationPage />);
    await waitUntilReady();
    expect(GetAllAssistantConfiguration).not.toHaveBeenCalled();

    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });

    const beforeCount = screen.getAllByTestId(/param-val-/).length;
    fireEvent.click(screen.getByRole('button', { name: 'Add parameter' }));
    await waitFor(() =>
      expect(screen.getAllByTestId(/param-val-/).length).toBe(beforeCount + 1),
    );
    const fields = screen.getAllByTestId(/param-val-/);
    const lastField = fields[fields.length - 1];
    fireEvent.change(lastField, {
      target: { value: 'assistantPrompt' },
    });

    expect(screen.getByText(/Mapping \(\d+\)/)).toBeInTheDocument();
    expect(lastField).toHaveValue('assistantPrompt');
  });

  it('creates authentication when valid', async () => {
    render(<CreateAssistantAuthenticationPage />);
    await waitUntilReady();
    expect(GetAllAssistantConfiguration).not.toHaveBeenCalled();

    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });
    fireEvent.click(
      screen.getByRole('button', { name: 'Save authentication' }),
    );

    await waitFor(() =>
      expect(CreateAssistantConfiguration).toHaveBeenCalledTimes(1),
    );
    await waitUntilReady();
    const createRequest = (CreateAssistantConfiguration as jest.Mock).mock
      .calls[0][1] as {
      provider: string;
      configurationType: string;
      enabled: boolean;
      optionsList: Array<{
        getKey: () => string;
        getValue: () => string;
      }>;
    };
    expect(createRequest.provider).toBe('http');
    expect(createRequest.configurationType).toBe('authentication');
    expect(createRequest.enabled).toBe(true);
    const optionMap = new Map(
      createRequest.optionsList.map(option => [
        option.getKey(),
        option.getValue(),
      ]),
    );
    expect(optionMap.get('http_method')).toBe('POST');
    expect(optionMap.get('http_url')).toBe('https://auth.example.com/resolve');
    expect(optionMap.get('http_headers')).toBe('{}');
    expect(optionMap.get('http_body')).toBe(
      '{"assistant.id":"assistantId","client.phone":"clientPhone"}',
    );
    expect(optionMap.get('fail_behavior')).toBe('BLOCK');
    expect(optionMap.get('timeout_ms')).toBe('5000');
    expect(optionMap.get('authentication.condition')).toBeDefined();
    expect(optionMap.get('auth.provider')).toBeUndefined();
    expect([...optionMap.keys()].sort()).toEqual(
      [
        'http_method',
        'http_url',
        'http_headers',
        'http_body',
        'fail_behavior',
        'timeout_ms',
        'authentication.condition',
      ].sort(),
    );
  });

  it('sends DO_NOTHING when on error is set to do nothing', async () => {
    render(<CreateAssistantAuthenticationPage />);
    await waitUntilReady();
    expect(GetAllAssistantConfiguration).not.toHaveBeenCalled();

    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });
    fireEvent.change(screen.getByTestId('assistant-auth-fail-behavior'), {
      target: { value: 'do_nothing' },
    });
    fireEvent.click(
      screen.getByRole('button', { name: 'Save authentication' }),
    );

    await waitFor(() =>
      expect(CreateAssistantConfiguration).toHaveBeenCalledTimes(1),
    );
    await waitUntilReady();
    const createRequest = (CreateAssistantConfiguration as jest.Mock).mock
      .calls[0][1] as {
      optionsList: Array<{
        getKey: () => string;
        getValue: () => string;
      }>;
    };
    const optionMap = new Map(
      createRequest.optionsList.map(option => [
        option.getKey(),
        option.getValue(),
      ]),
    );
    expect(optionMap.get('fail_behavior')).toBe('DO_NOTHING');
  });

  it('maps DO_NOTHING from API to do nothing option in UI', async () => {
    (GetAllAssistantConfiguration as jest.Mock).mockResolvedValueOnce(
      getSuccessLoadResponse('DO_NOTHING'),
    );

    render(<UpdateAssistantAuthenticationPage />);
    await waitFor(() =>
      expect(GetAllAssistantConfiguration).toHaveBeenCalled(),
    );
    await waitUntilReady();

    expect(screen.getByTestId('assistant-auth-fail-behavior')).toHaveValue(
      'do_nothing',
    );
  });

  it('maps legacy none from API to do nothing option in UI', async () => {
    (GetAllAssistantConfiguration as jest.Mock).mockResolvedValueOnce(
      getSuccessLoadResponse('none'),
    );

    render(<UpdateAssistantAuthenticationPage />);
    await waitFor(() =>
      expect(GetAllAssistantConfiguration).toHaveBeenCalled(),
    );
    await waitUntilReady();

    expect(screen.getByTestId('assistant-auth-fail-behavior')).toHaveValue(
      'do_nothing',
    );
  });

  it('saves DO_NOTHING when loaded legacy none without changing selection', async () => {
    (GetAllAssistantConfiguration as jest.Mock).mockResolvedValueOnce(
      getSuccessLoadResponse('none'),
    );

    render(<UpdateAssistantAuthenticationPage />);
    await waitFor(() =>
      expect(GetAllAssistantConfiguration).toHaveBeenCalled(),
    );
    await waitUntilReady();

    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });
    fireEvent.click(
      screen.getByRole('button', { name: 'Save authentication' }),
    );

    await waitFor(() =>
      expect(UpdateAssistantConfiguration).toHaveBeenCalledTimes(1),
    );
    await waitUntilReady();
    const updateRequest = (UpdateAssistantConfiguration as jest.Mock).mock
      .calls[0][1] as {
      id: string;
      configurationType: string;
      optionsList: Array<{
        getKey: () => string;
        getValue: () => string;
      }>;
    };
    expect(updateRequest.id).toBe('auth-config-1');
    expect(updateRequest.configurationType).toBe('authentication');
    const optionMap = new Map(
      updateRequest.optionsList.map(option => [
        option.getKey(),
        option.getValue(),
      ]),
    );
    expect(optionMap.get('fail_behavior')).toBe('DO_NOTHING');
  });

  it('falls back to add flow when initial load does not return authentication data', async () => {
    (GetAllAssistantConfiguration as jest.Mock).mockResolvedValueOnce({
      getSuccess: () => false,
      getError: () => ({
        getHumanmessage: () => 'Failed to load auth',
      }),
    });

    render(<UpdateAssistantAuthenticationPage />);

    await waitFor(() =>
      expect(screen.getByText('Add Authentication')).toBeInTheDocument(),
    );
    expect(screen.queryByText('Failed to load auth')).not.toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Save authentication' }),
      ).not.toBeDisabled(),
    );

    fireEvent.click(
      screen.getByRole('button', { name: 'Save authentication' }),
    );
    expect(
      screen.getByText('Please provide a server URL for authentication.'),
    ).toBeInTheDocument();
    expect(CreateAssistantConfiguration).not.toHaveBeenCalled();
  });

  it('disables authentication from the list after confirmation', async () => {
    (GetAllAssistantConfiguration as jest.Mock)
      .mockResolvedValueOnce(getSuccessLoadResponse('BLOCK', true))
      .mockResolvedValueOnce(getSuccessLoadResponse('BLOCK', false));

    render(<ConfigureAssistantAuthenticationPage />);

    await waitFor(() =>
      expect(
        screen.getByText('https://auth.example.com/loaded'),
      ).toBeInTheDocument(),
    );

    fireEvent.click(screen.getByRole('button', { name: 'Disable' }));
    expect(screen.getByRole('dialog')).toHaveTextContent(
      'Disable authentication?',
    );

    await act(async () => {
      fireEvent.click(
        within(screen.getByRole('dialog')).getByRole('button', {
          name: 'Disable',
        }),
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    await waitFor(() =>
      expect(UpdateAssistantConfiguration).toHaveBeenCalledTimes(1),
    );
    await waitFor(() =>
      expect(GetAllAssistantConfiguration).toHaveBeenCalledTimes(2),
    );
    await waitFor(() =>
      expect(screen.getByText('Inactive')).toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    const request = (UpdateAssistantConfiguration as jest.Mock).mock
      .calls[0][1] as {
      id: string;
      provider: string;
      configurationType: string;
      enabled: boolean;
      optionsList: unknown[];
    };
    expect(request.id).toBe('auth-config-1');
    expect(request.provider).toBe('http');
    expect(request.configurationType).toBe('authentication');
    expect(request.enabled).toBe(false);
    expect(request.optionsList).toHaveLength(7);
  });

  it('enables authentication from the list after confirmation', async () => {
    (GetAllAssistantConfiguration as jest.Mock)
      .mockResolvedValueOnce(getSuccessLoadResponse('BLOCK', false))
      .mockResolvedValueOnce(getSuccessLoadResponse('BLOCK', true));

    render(<ConfigureAssistantAuthenticationPage />);

    await waitFor(() =>
      expect(screen.getByText('Inactive')).toBeInTheDocument(),
    );

    fireEvent.click(screen.getByRole('button', { name: 'Enable' }));
    expect(screen.getByRole('dialog')).toHaveTextContent(
      'Enable authentication?',
    );

    await act(async () => {
      fireEvent.click(
        within(screen.getByRole('dialog')).getByRole('button', {
          name: 'Enable',
        }),
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    await waitFor(() =>
      expect(UpdateAssistantConfiguration).toHaveBeenCalledTimes(1),
    );
    await waitFor(() =>
      expect(GetAllAssistantConfiguration).toHaveBeenCalledTimes(2),
    );
    await waitFor(() => expect(screen.getByText('Active')).toBeInTheDocument());
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    const request = (UpdateAssistantConfiguration as jest.Mock).mock
      .calls[0][1] as {
      enabled: boolean;
      optionsList: unknown[];
    };
    expect(request.enabled).toBe(true);
    expect(request.optionsList).toHaveLength(7);
  });
});

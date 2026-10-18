import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { CreateAssistantAnalysis } from '@/app/pages/assistant/actions/configure-assistant-analysis/create-assistant-analysis';
import { CreateAssistantConfiguration } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';

const mockGoBack = jest.fn();
const mockGoToConfigureAssistantAnalysis = jest.fn();

let mockEndpointIdToPick = 'endpoint-1';

jest.mock('@rapidaai/react', () => ({
  ConnectionConfig: class ConnectionConfig {},
  Metadata: class Metadata {
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
  },
  CreateAssistantConfigurationRequest: class CreateAssistantConfigurationRequest {
    assistantId = '';
    provider = '';
    configurationType = '';
    enabled = false;
    optionsList: any[] = [];
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
    setOptionsList(v: any[]) {
      this.optionsList = v;
    }
    getOptionsList() {
      return this.optionsList;
    }
  },
  CreateAssistantConfiguration: jest.fn(),
}));

jest.mock('react-hot-toast/headless', () => ({
  __esModule: true,
  default: {
    success: jest.fn(),
  },
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goBack: mockGoBack,
    goToConfigureAssistantAnalysis: mockGoToConfigureAssistantAnalysis,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'auth-1',
    token: 'token-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/utils', () => ({
  cn: (...inputs: any[]) => inputs.filter(Boolean).join(' '),
  randomMeaningfullName: () => 'analysis-default',
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    showDialog: (cb: () => void) => cb(),
    ConfirmDialogComponent: () => null,
  }),
}));

jest.mock('@/app/components/domain/dropdowns/endpoint-dropdown', () => ({
  EndpointDropdown: ({ onChangeEndpoint }: any) => (
    <button
      type="button"
      onClick={() =>
        onChangeEndpoint({
          getId: () => mockEndpointIdToPick,
        })
      }
    >
      Pick endpoint
    </button>
  ),
}));

jest.mock('@/app/components/ui/composites/tab-form', () => ({
  TabForm: ({ form, activeTab, errorMessage, formHeading }: any) => {
    const React = require('react');
    const active = form.find((f: any) => f.code === activeTab) || form[0];
    return (
      <div>
        <h1>{formHeading}</h1>
        {errorMessage ? <div>{errorMessage}</div> : null}
        <div>{active.body}</div>
        <div>
          {Array.isArray(active.actions)
            ? active.actions.map((action: React.ReactElement, idx: number) => (
                <div key={idx}>{action}</div>
              ))
            : active.actions}
        </div>
      </div>
    );
  },
}));

jest.mock('@/app/components/ui/primitives/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: ({
    labelText: _l,
    helperText: _h,
    hideLabel: _hl,
    warn: _w,
    warnText: _wt,
    invalid: _inv,
    invalidText: _it,
    ...props
  }: any) => <input {...props} />,
  TextArea: ({
    labelText: _l,
    helperText: _h,
    hideLabel: _hl,
    warn: _w,
    warnText: _wt,
    invalid: _inv,
    invalidText: _it,
    ...props
  }: any) => <textarea {...props} />,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({
    children,
    isLoading: _,
    renderIcon: _r,
    hasIconOnly: _h,
    iconDescription: _d,
    ...props
  }: any) => <button {...props}>{children}</button>,
  SecondaryButton: ({
    children,
    isLoading: _,
    renderIcon: _r,
    hasIconOnly: _h,
    iconDescription: _d,
    ...props
  }: any) => <button {...props}>{children}</button>,
  TertiaryButton: ({
    children,
    isLoading: _,
    renderIcon: _r,
    hasIconOnly: _h,
    iconDescription: _d,
    ...props
  }: any) => <button {...props}>{children}</button>,
}));

jest.mock('@carbon/react', () => ({
  ButtonSet: ({ children }: any) => <div>{children}</div>,
  Button: ({
    children,
    hasIconOnly: _,
    renderIcon: _r,
    iconDescription: _d,
    ...props
  }: any) => <button {...props}>{children}</button>,
  Select: ({ children, labelText: _, hideLabel: _h, ...props }: any) => (
    <select {...props}>{children}</select>
  ),
  SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
  Table: ({ children }: any) => <table>{children}</table>,
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableCell: ({ children, colSpan }: any) => (
    <td colSpan={colSpan}>{children}</td>
  ),
  TableContainer: ({ children }: any) => <section>{children}</section>,
  TableHead: ({ children }: any) => <thead>{children}</thead>,
  TableHeader: ({ children }: any) => <th>{children}</th>,
  TableRow: ({ children }: any) => <tr>{children}</tr>,
  Tooltip: ({ children }: any) => <span>{children}</span>,
  NumberInput: ({
    label,
    helperText: _ht,
    onChange,
    value,
    hideLabel: _hl,
    ...rest
  }: any) => (
    <input
      aria-label={label}
      value={value}
      onChange={e => onChange(e, { value: Number(e.target.value) })}
    />
  ),
}));

describe('CreateAssistantAnalysis', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockEndpointIdToPick = 'endpoint-1';
  });

  it('shows endpoint validation error when continuing without endpoint', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<CreateAssistantAnalysis assistantId="assistant-1" />);

    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(
      screen.getByText(
        'Please select a valid endpoint to be executed for analysis.',
      ),
    ).toBeInTheDocument();
    expect(CreateAssistantConfiguration).not.toHaveBeenCalled();
  });

  it('creates analysis successfully and navigates back to analysis listing', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
    });

    render(<CreateAssistantAnalysis assistantId="assistant-1" />);

    await user.click(screen.getByRole('button', { name: 'Pick endpoint' }));
    await user.click(screen.getByRole('button', { name: 'Continue' }));
    await user.click(
      screen.getByRole('button', { name: 'Configure analysis' }),
    );

    await waitFor(() => {
      expect(CreateAssistantConfiguration).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith(
        'Analysis added to assistant successfully',
      );
    });
    expect(mockGoToConfigureAssistantAnalysis).toHaveBeenCalledWith(
      'assistant-1',
    );
  });

  it('shows human error message when create API returns unsuccessful response', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => false,
      getError: () => ({ getHumanmessage: () => 'Name already used' }),
    });

    render(<CreateAssistantAnalysis assistantId="assistant-1" />);

    await user.click(screen.getByRole('button', { name: 'Pick endpoint' }));
    await user.click(screen.getByRole('button', { name: 'Continue' }));
    await user.click(
      screen.getByRole('button', { name: 'Configure analysis' }),
    );

    expect(await screen.findByText('Name already used')).toBeInTheDocument();
  });

  it('supports add and edit for parameter mapping before create', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
    });

    render(<CreateAssistantAnalysis assistantId="assistant-1" />);

    await user.click(screen.getByRole('button', { name: 'Pick endpoint' }));
    await user.selectOptions(
      document.getElementById('tool-condition-key') as HTMLElement,
      String('conversation_mode'),
    );
    await user.selectOptions(
      document.getElementById('tool-condition-source-value') as HTMLElement,
      String('voice'),
    );
    await user.click(screen.getByRole('button', { name: 'Add parameter' }));
    await user.selectOptions(
      document.getElementById('param-type-1') as HTMLElement,
      String('assistant'),
    );
    await user.selectOptions(
      document.getElementById('param-key-1') as HTMLElement,
      String('name'),
    );
    await user.click(document.getElementById('param-val-1') as HTMLElement);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('assistantName'));

    await user.click(screen.getByRole('button', { name: 'Continue' }));
    await user.click(
      screen.getByRole('button', { name: 'Configure analysis' }),
    );

    await waitFor(() =>
      expect(CreateAssistantConfiguration).toHaveBeenCalled(),
    );
    const request = (CreateAssistantConfiguration as jest.Mock).mock
      .calls[0][1];
    expect(request.getAssistantid()).toBe('assistant-1');
    expect(request.getConfigurationtype()).toBe('analysis');
    expect(request.getProvider()).toBe('endpoint');
    expect(request.getEnabled()).toBe(true);
    const mappedParams = request.getOptionsList().map((option: any) => ({
      key: option.getKey(),
      value: option.getValue(),
    }));
    expect(mappedParams).toEqual(
      expect.arrayContaining([
        { key: 'name', value: 'analysis-default' },
        { key: 'description', value: '' },
        { key: 'execution_priority', value: '0' },
        { key: 'endpoint_id', value: 'endpoint-1' },
        { key: 'endpoint_version', value: 'latest' },
        {
          key: 'endpoint_parameters',
          value: JSON.stringify({
            'conversation.messages': 'messages',
            'assistant.name': 'assistantName',
          }),
        },
        {
          key: 'analysis.condition',
          value: JSON.stringify([
            {
              key: 'conversation_mode',
              condition: '=',
              value: 'voice',
            },
          ]),
        },
      ]),
    );
  });

  it('blocks reserved analysis option mapping key', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<CreateAssistantAnalysis assistantId="assistant-1" />);

    await user.click(screen.getByRole('button', { name: 'Pick endpoint' }));
    await user.selectOptions(
      document.getElementById('param-type-0') as HTMLElement,
      String('option'),
    );
    await user.click(document.getElementById('param-key-0') as HTMLElement);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('endpoint_parameters'));
    await user.click(document.getElementById('param-val-0') as HTMLElement);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('shouldFail'));
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(
      screen.getByText(
        'option.endpoint_parameters is reserved and managed by analysis options.',
      ),
    ).toBeInTheDocument();
    expect(CreateAssistantConfiguration).not.toHaveBeenCalled();
  });
});

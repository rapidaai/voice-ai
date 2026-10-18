import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { CreateAssistantWebhook } from '@/app/pages/assistant/actions/configure-assistant-webhook/create-assistant-webhook';
import { UpdateAssistantWebhook } from '@/app/pages/assistant/actions/configure-assistant-webhook/update-assistant-webhook';
import {
  CreateAssistantConfiguration,
  GetAssistantConfiguration,
  UpdateAssistantConfiguration,
} from '@rapidaai/react';

let mockParams: Record<string, string | undefined> = {
  assistantId: 'assistant-1',
  webhookId: 'webhook-1',
};

const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockNavigate = {
  goBack: jest.fn(),
  goToAssistantWebhook: jest.fn(),
};

jest.mock('@rapidaai/react', () => ({
  ConnectionConfig: class {
    static WithDebugger(config: unknown) {
      return config;
    }
  },
  Metadata: class {
    key = '';
    value = '';
    setKey(value: string) {
      this.key = value;
    }
    getKey() {
      return this.key;
    }
    setValue(value: string) {
      this.value = value;
    }
    getValue() {
      return this.value;
    }
  },
  CreateAssistantConfigurationRequest: class {
    assistantId = '';
    provider = '';
    configurationType = '';
    enabled = false;
    optionsList: any[] = [];
    setAssistantid(value: string) {
      this.assistantId = value;
    }
    getAssistantid() {
      return this.assistantId;
    }
    setProvider(value: string) {
      this.provider = value;
    }
    getProvider() {
      return this.provider;
    }
    setConfigurationtype(value: string) {
      this.configurationType = value;
    }
    getConfigurationtype() {
      return this.configurationType;
    }
    setEnabled(value: boolean) {
      this.enabled = value;
    }
    getEnabled() {
      return this.enabled;
    }
    setOptionsList(value: any[]) {
      this.optionsList = value;
    }
    getOptionsList() {
      return this.optionsList;
    }
  },
  UpdateAssistantConfigurationRequest: class {
    id = '';
    assistantId = '';
    provider = '';
    configurationType = '';
    enabled = false;
    optionsList: any[] = [];
    setId(value: string) {
      this.id = value;
    }
    getId() {
      return this.id;
    }
    setAssistantid(value: string) {
      this.assistantId = value;
    }
    getAssistantid() {
      return this.assistantId;
    }
    setProvider(value: string) {
      this.provider = value;
    }
    getProvider() {
      return this.provider;
    }
    setConfigurationtype(value: string) {
      this.configurationType = value;
    }
    getConfigurationtype() {
      return this.configurationType;
    }
    setEnabled(value: boolean) {
      this.enabled = value;
    }
    getEnabled() {
      return this.enabled;
    }
    setOptionsList(value: any[]) {
      this.optionsList = value;
    }
    getOptionsList() {
      return this.optionsList;
    }
  },
  GetAssistantConfigurationRequest: class {
    id = '';
    assistantId = '';
    setId(value: string) {
      this.id = value;
    }
    getId() {
      return this.id;
    }
    setAssistantid(value: string) {
      this.assistantId = value;
    }
    getAssistantid() {
      return this.assistantId;
    }
  },
  CreateAssistantConfiguration: jest.fn(),
  GetAssistantConfiguration: jest.fn(),
  UpdateAssistantConfiguration: jest.fn(),
}));

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => mockParams,
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => mockNavigate,
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({ authId: 'u1', token: 't1', projectId: 'p1' }),
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    showDialog: (cb: () => void) => cb(),
    ConfirmDialogComponent: () => null,
  }),
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

jest.mock('@/app/components/ui/primitives/form', () => {
  const React = require('react');
  return {
    Stack: ({ children }: any) => React.createElement('div', null, children),
    FormGroup: ({ children, legendText, messageText }: any) =>
      React.createElement(
        'fieldset',
        null,
        legendText ? React.createElement('legend', null, legendText) : null,
        messageText ? React.createElement('div', null, messageText) : null,
        children,
      ),
    TextInput: ({
      id,
      labelText,
      value,
      onChange,
      placeholder,
      hideLabel,
    }: any) =>
      React.createElement(
        'div',
        null,
        !hideLabel && labelText
          ? React.createElement('label', { htmlFor: id }, labelText)
          : null,
        React.createElement('input', {
          id,
          value: value ?? '',
          onChange,
          placeholder,
          'data-testid': id,
        }),
      ),
    TextArea: ({ id, labelText, value, onChange, placeholder }: any) =>
      React.createElement(
        'div',
        null,
        labelText
          ? React.createElement('label', { htmlFor: id }, labelText)
          : null,
        React.createElement('textarea', {
          id,
          value: value ?? '',
          onChange,
          placeholder,
          'data-testid': id,
        }),
      ),
  };
});

jest.mock('@/app/components/ui/primitives/input-group', () => {
  const React = require('react');
  return {
    InputGroup: ({ title, children }: any) =>
      React.createElement(
        'section',
        null,
        title ? React.createElement('div', null, title) : null,
        children,
      ),
  };
});

jest.mock('@/app/components/ui/primitives/slider', () => {
  const React = require('react');
  return {
    Slider: ({ value, onSlide }: any) =>
      React.createElement('input', {
        type: 'range',
        value,
        onChange: (e: any) => onSlide(Number(e.target.value)),
        'data-testid': 'webhook-timeout-slider',
      }),
  };
});

jest.mock('@/app/components/ui/primitives/button', () => {
  const React = require('react');
  return {
    PrimaryButton: ({
      children,
      isLoading: _,
      renderIcon: _r,
      hasIconOnly: _h,
      iconDescription: _d,
      ...props
    }: any) => React.createElement('button', props, children),
    SecondaryButton: ({
      children,
      isLoading: _,
      renderIcon: _r,
      hasIconOnly: _h,
      iconDescription: _d,
      ...props
    }: any) => React.createElement('button', props, children),
    TertiaryButton: ({
      children,
      isLoading: _,
      renderIcon: _r,
      hasIconOnly: _h,
      iconDescription: _d,
      ...props
    }: any) => React.createElement('button', props, children),
  };
});

jest.mock('@carbon/react', () => {
  const React = require('react');
  return {
    ButtonSet: ({ children }: any) =>
      React.createElement('div', null, children),
    Table: ({ children }: any) => React.createElement('table', null, children),
    TableBody: ({ children }: any) =>
      React.createElement('tbody', null, children),
    TableCell: ({ children, colSpan }: any) =>
      React.createElement('td', { colSpan }, children),
    TableContainer: ({ children }: any) =>
      React.createElement('div', null, children),
    TableHead: ({ children }: any) =>
      React.createElement('thead', null, children),
    TableHeader: ({ children }: any) =>
      React.createElement('th', null, children),
    TableRow: ({ children }: any) => React.createElement('tr', null, children),
    Tooltip: ({ children }: any) => React.createElement('span', null, children),
    Select: ({ id, labelText, value, onChange, children, hideLabel }: any) =>
      React.createElement(
        'div',
        null,
        !hideLabel && labelText
          ? React.createElement('label', { htmlFor: id }, labelText)
          : null,
        React.createElement(
          'select',
          { id, value, onChange, 'data-testid': id },
          children,
        ),
      ),
    SelectItem: ({ value, text }: any) =>
      React.createElement('option', { value }, text),
    NumberInput: ({ id, value, onChange, label, hideLabel }: any) =>
      React.createElement(
        'div',
        null,
        !hideLabel && label
          ? React.createElement('label', { htmlFor: id }, label)
          : null,
        React.createElement('input', {
          id,
          type: 'number',
          value,
          onChange: (e: any) => onChange?.(e, { value: e.target.value }),
          'data-testid': id,
        }),
      ),
    Checkbox: ({ id, labelText, checked, onChange }: any) =>
      React.createElement(
        'label',
        { htmlFor: id },
        React.createElement('input', {
          id,
          type: 'checkbox',
          checked: !!checked,
          onChange: (e: any) => onChange?.(e, { checked: e.target.checked }),
        }),
        labelText,
      ),
    StructuredListWrapper: ({
      children,
      isCondensed: _isCondensed,
      isFlush: _isFlush,
      selection: _selection,
      selectedInitialRow: _selectedInitialRow,
      ...props
    }: any) => React.createElement('div', props, children),
    StructuredListHead: ({ children }: any) =>
      React.createElement('div', null, children),
    StructuredListBody: ({ children }: any) =>
      React.createElement('div', null, children),
    StructuredListRow: ({
      children,
      head: _head,
      selection: _selection,
    }: any) => React.createElement('div', null, children),
    StructuredListCell: ({ children, head: _head, noWrap: _noWrap }: any) =>
      React.createElement('div', null, children),
    Button: ({
      iconDescription,
      children,
      hasIconOnly: _,
      renderIcon: _r,
      ...props
    }: any) =>
      React.createElement(
        'button',
        { ...props, 'aria-label': iconDescription || children || 'button' },
        children || 'button',
      ),
  };
});

jest.mock('react-hot-toast/headless', () => ({
  success: jest.fn(),
  error: jest.fn(),
}));

describe('Assistant webhook flows', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockParams = { assistantId: 'assistant-1', webhookId: 'webhook-1' };

    (CreateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
    });

    (GetAssistantConfiguration as jest.Mock).mockResolvedValue({
      getData: () => ({
        getOptionsList: () => [
          {
            getKey: () => 'http_method',
            getValue: () => 'POST',
          },
          {
            getKey: () => 'http_url',
            getValue: () => 'https://hooks.example.com/incoming',
          },
          {
            getKey: () => 'retry_status_codes',
            getValue: () => '["50X"]',
          },
          {
            getKey: () => 'max_retry_count',
            getValue: () => '2',
          },
          {
            getKey: () => 'timeout_seconds',
            getValue: () => '200',
          },
          {
            getKey: () => 'http_headers',
            getValue: () => '{"Authorization":"Bearer token"}',
          },
          {
            getKey: () => 'assistant_events',
            getValue: () => '["conversation.begin"]',
          },
          {
            getKey: () => 'execution_priority',
            getValue: () => '1',
          },
          {
            getKey: () => 'description',
            getValue: () => 'existing webhook',
          },
          {
            getKey: () => 'webhook.condition',
            getValue: () =>
              '[{"key":"source","condition":"=","value":"phone"}]',
          },
          {
            getKey: () => 'http_body',
            getValue: () =>
              '{"event.type":"event","assistant.id":"assistant_id"}',
          },
        ],
      }),
    });

    (UpdateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
    });
  });

  const selectFirstEventAndContinue = async () => {
    const user = userEvent.setup();
    const eventCheckbox = document.getElementById(
      'webhook-event-call.received',
    ) as HTMLInputElement;
    await user.click(eventCheckbox);
    await user.click(screen.getByRole('button', { name: 'Continue' }));
  };

  const selectWebRTCEventAndContinue = async () => {
    const user = userEvent.setup();
    const eventCheckbox = document.getElementById(
      'webhook-event-webrtc.connected',
    ) as HTMLInputElement;
    await user.click(eventCheckbox);
    await user.click(screen.getByRole('button', { name: 'Continue' }));
  };

  const continueWithLoadedEvent = async () => {
    const user = userEvent.setup();
    await waitFor(() => {
      expect(
        document.getElementById(
          'webhook-event-conversation.begin',
        ) as HTMLInputElement,
      ).toBeChecked();
    });
    await user.click(screen.getByRole('button', { name: 'Continue' }));
  };

  const continueWithLoadedAndWebRTCEvent = async () => {
    const user = userEvent.setup();
    await waitFor(() => {
      expect(
        document.getElementById(
          'webhook-event-conversation.begin',
        ) as HTMLInputElement,
      ).toBeChecked();
    });
    const eventCheckbox = document.getElementById(
      'webhook-event-webrtc.reconnecting',
    ) as HTMLInputElement;
    await user.click(eventCheckbox);
    await user.click(screen.getByRole('button', { name: 'Continue' }));
  };

  it('create webhook supports header add-delete and continues without payload mapping', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<CreateAssistantWebhook assistantId="assistant-1" />);

    await selectFirstEventAndContinue();

    await user.click(screen.getByTestId('webhook-endpoint'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('https://api.example.com/webhook'));

    expect(screen.getByText('Headers (0)')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Add header' }));
    expect(screen.getByText('Headers (1)')).toBeInTheDocument();
    const headerKeyInput = screen.getByTestId('api-header-key-0');
    const headerRow = headerKeyInput.closest('tr');
    const headerRemoveButton = headerRow?.querySelector('button');
    expect(headerRemoveButton).toBeTruthy();
    await user.click(headerRemoveButton!);
    expect(screen.getByText('Headers (0)')).toBeInTheDocument();
    expect(screen.getByText('Destination')).toBeInTheDocument();
  });

  it('create webhook validates headers when key is present without value', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<CreateAssistantWebhook assistantId="assistant-1" />);

    await selectFirstEventAndContinue();

    await user.click(screen.getByTestId('webhook-endpoint'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('https://api.example.com/webhook'));

    await user.click(screen.getByRole('button', { name: 'Add header' }));
    await user.click(screen.getByTestId('api-header-key-0'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Authorization'));
    await user.click(screen.getByRole('button', { name: 'Configure webhook' }));

    expect(
      screen.getByText('Headers with a key must also include a value.'),
    ).toBeInTheDocument();
  });

  it('create webhook submits with selected event and configuration', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<CreateAssistantWebhook assistantId="assistant-1" />);

    await selectWebRTCEventAndContinue();

    await user.click(screen.getByTestId('webhook-endpoint'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('https://api.example.com/webhook'));
    await user.selectOptions(screen.getByTestId('webhook-max-retries'), '2');
    await user.click(screen.getByTestId('webhook-timeout'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('220'));
    await user.click(screen.getByTestId('webhook-priority'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('4'));
    await user.click(screen.getByRole('button', { name: 'Configure webhook' }));

    await waitFor(() =>
      expect(CreateAssistantConfiguration).toHaveBeenCalledTimes(1),
    );
    const req = (CreateAssistantConfiguration as jest.Mock).mock.calls[0][1];
    expect(req.getAssistantid()).toBe('assistant-1');
    expect(req.getConfigurationtype()).toBe('webhook');
    expect(req.getProvider()).toBe('http');
    expect(req.getEnabled()).toBe(true);

    const optionMap = new Map(
      req
        .getOptionsList()
        .map((option: any) => [option.getKey(), option.getValue()]),
    );
    expect(optionMap.get('http_method')).toBe('POST');
    expect(optionMap.get('http_url')).toBe('https://api.example.com/webhook');
    expect(optionMap.get('max_retry_count')).toBe('2');
    expect(optionMap.get('timeout_seconds')).toBe('220');
    expect(optionMap.get('assistant_events')).toBe('["webrtc.connected"]');
    expect(optionMap.get('execution_priority')).toBe('4');
    expect(optionMap.get('description')).toBe('');
    expect(optionMap.has('http_body')).toBe(false);
    expect(optionMap.has('webhook.condition')).toBe(false);
  });

  it('update webhook validates invalid destination url', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<UpdateAssistantWebhook assistantId="assistant-1" />);

    await continueWithLoadedEvent();

    await waitFor(() => {
      expect(screen.getByTestId('webhook-endpoint')).toHaveValue(
        'https://hooks.example.com/incoming',
      );
    });

    await user.click(screen.getByTestId('webhook-endpoint'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('bad-url'));
    await user.click(screen.getByRole('button', { name: 'Update webhook' }));

    expect(
      screen.getByText('Please provide a valid server URL for the webhook.'),
    ).toBeInTheDocument();
  });

  it('update webhook validates headers when key is present without value', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<UpdateAssistantWebhook assistantId="assistant-1" />);

    await continueWithLoadedEvent();

    await waitFor(() => {
      expect(screen.getByTestId('webhook-endpoint')).toHaveValue(
        'https://hooks.example.com/incoming',
      );
    });

    await user.clear(screen.getByTestId('api-header-val-0'));
    await user.click(screen.getByRole('button', { name: 'Update webhook' }));

    expect(
      screen.getByText('Headers with a key must also include a value.'),
    ).toBeInTheDocument();
  });

  it('update webhook submits with loaded values', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<UpdateAssistantWebhook assistantId="assistant-1" />);

    await continueWithLoadedAndWebRTCEvent();

    await waitFor(() => {
      expect(screen.getByTestId('webhook-endpoint')).toHaveValue(
        'https://hooks.example.com/incoming',
      );
    });

    await user.click(screen.getByRole('button', { name: 'Update webhook' }));

    await waitFor(() =>
      expect(UpdateAssistantConfiguration).toHaveBeenCalledTimes(1),
    );
    const req = (UpdateAssistantConfiguration as jest.Mock).mock.calls[0][1];
    expect(req.getAssistantid()).toBe('assistant-1');
    expect(req.getId()).toBe('webhook-1');
    expect(req.getConfigurationtype()).toBe('webhook');
    expect(req.getProvider()).toBe('http');
    expect(req.getEnabled()).toBe(true);
    const optionMap = new Map(
      req
        .getOptionsList()
        .map((option: any) => [option.getKey(), option.getValue()]),
    );
    expect(optionMap.get('assistant_events')).toBe(
      '["conversation.begin","webrtc.reconnecting"]',
    );
    expect(optionMap.get('execution_priority')).toBe('1');
    expect(optionMap.get('description')).toBe('existing webhook');
    expect(optionMap.has('http_body')).toBe(false);
    expect(optionMap.has('webhook.condition')).toBe(false);
  });
});

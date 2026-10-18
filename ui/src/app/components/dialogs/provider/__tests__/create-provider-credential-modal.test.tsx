import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import toast from 'react-hot-toast/headless';
import { CreateProviderKey } from '@rapidaai/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';

import {
  CreateProviderCredentialDialog,
  CredentialKeyValueField,
  parseCredentialEntries,
  ProviderCredentialForm,
  serializeCredentialEntries,
} from '../create-provider-credential-modal';

const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockReloadProviderCredentials = jest.fn();

const mockOpenaiProvider = {
  code: 'openai',
  name: 'OpenAI',
  featureList: ['llm'],
  configurations: [
    { name: 'api_key', type: 'password', label: 'API key' },
    {
      name: 'base_url',
      type: 'text',
      label: 'Base URL (optional)',
      required: false,
    },
    {
      name: 'mode',
      type: 'select',
      label: 'Mode',
      choices: [
        { label: 'Chat', value: 'chat' },
        { label: 'Responses', value: 'responses' },
      ],
    },
    { name: 'headers', type: 'key_value', label: 'Headers', required: false },
  ],
};

jest.mock('@rapidaai/react', () => {
  class CreateProviderCredentialRequest {
    credential: unknown;
    name = '';
    provider = '';

    setCredential(value: unknown) {
      this.credential = value;
    }

    setName(value: string) {
      this.name = value;
    }

    setProvider(value: string) {
      this.provider = value;
    }
  }

  return {
    ConnectionConfig: {
      WithDebugger: jest.fn((headers: unknown) => ({ headers })),
    },
    CreateProviderCredentialRequest,
    CreateProviderKey: jest.fn(),
  };
});

jest.mock('google-protobuf/google/protobuf/struct_pb', () => ({
  Struct: {
    fromJavaScript: jest.fn((value: unknown) => ({ value })),
  },
}));

jest.mock('@/providers', () => ({
  INTEGRATION_PROVIDER: [
    {
      code: 'openai',
      name: 'OpenAI',
      featureList: ['llm'],
      configurations: [
        { name: 'api_key', type: 'password', label: 'API key' },
        {
          name: 'base_url',
          type: 'text',
          label: 'Base URL (optional)',
          required: false,
        },
        {
          name: 'mode',
          type: 'select',
          label: 'Mode',
          choices: [
            { label: 'Chat', value: 'chat' },
            { label: 'Responses', value: 'responses' },
          ],
        },
        {
          name: 'headers',
          type: 'key_value',
          label: 'Headers',
          required: false,
        },
      ],
    },
  ],
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    projectId: 'project-1',
    token: 'token-1',
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    hideLoader: mockHideLoader,
    loading: false,
    showLoader: mockShowLoader,
  }),
}));

jest.mock('@/context/provider-context', () => ({
  useProviderContext: () => ({
    reloadProviderCredentials: mockReloadProviderCredentials,
  }),
}));

jest.mock('@/configs', () => ({
  connectionConfig: { api: 'test' },
}));

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
  success: jest.fn(),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  ErrorMessage: ({ message }: any) =>
    message ? <p role="alert">{message}</p> : null,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  Modal: ({ children, open }: any) =>
    open ? <section role="dialog">{children}</section> : null,
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalFooter: ({ children }: any) => <footer>{children}</footer>,
  ModalHeader: ({ label, onClose, title }: any) => (
    <header>
      <span>{label}</span>
      <h2>{title}</h2>
      <button type="button" onClick={onClose}>
        Close
      </button>
    </header>
  ),
  PrimaryButton: ({ children, isLoading, ...props }: any) => (
    <button data-loading={String(Boolean(isLoading))} {...props}>
      {children}
    </button>
  ),
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  TertiaryButton: ({ children, renderIcon, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  Stack: ({ children }: any) => <div>{children}</div>,
  TextArea: ({ id, labelText, onChange, value }: any) => (
    <label htmlFor={id}>
      {labelText}
      <textarea id={id} onChange={onChange} value={value} />
    </label>
  ),
  TextInput: ({
    hideLabel,
    id,
    labelText,
    onChange,
    placeholder,
    value,
  }: any) => (
    <label htmlFor={id}>
      {hideLabel ? null : labelText || placeholder}
      <input
        aria-label={hideLabel ? placeholder : undefined}
        id={id}
        onChange={onChange}
        placeholder={placeholder}
        value={value}
      />
    </label>
  ),
}));

jest.mock('@carbon/react', () => ({
  Button: ({ children, iconDescription, onClick }: any) => (
    <button type="button" aria-label={iconDescription} onClick={onClick}>
      {children}
    </button>
  ),
  Dropdown: ({
    itemToString,
    items,
    onChange,
    selectedItem,
    titleText,
  }: any) => (
    <label>
      {titleText}
      <select
        aria-label={titleText}
        value={selectedItem?.code || ''}
        onChange={event =>
          onChange({
            selectedItem:
              items.find((item: any) => item.code === event.target.value) ||
              null,
          })
        }
      >
        <option value="">Select</option>
        {items.map((item: any) => (
          <option key={item.code} value={item.code}>
            {itemToString ? itemToString(item) : item.name}
          </option>
        ))}
      </select>
    </label>
  ),
  Select: ({ children, labelText, onChange, value }: any) => (
    <label>
      {labelText}
      <select aria-label={labelText} onChange={onChange} value={value}>
        {children}
      </select>
    </label>
  ),
  SelectItem: ({ text, value }: any) => <option value={value}>{text}</option>,
}));

jest.mock('@carbon/icons-react', () => ({
  Add: () => <svg data-testid="add-icon" />,
  TrashCan: () => <svg data-testid="trash-icon" />,
}));

const apiResponse = (success: boolean, message = 'Create failed') => ({
  getError: () => ({ getHumanmessage: () => message }),
  getSuccess: () => success,
});

describe('CreateProviderCredentialDialog', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('validates provider, key name, and required configuration before submit', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <CreateProviderCredentialDialog
        modalOpen
        setModalOpen={jest.fn()}
        currentProvider={null}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Configure' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Please select the provider which you want to create the key.',
    );

    await user.selectOptions(
      screen.getByLabelText('Select your provider'),
      String('openai'),
    );
    await user.click(screen.getByRole('button', { name: 'Configure' }));
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Please provide a valid key name for the credential.',
    );

    await user.click(screen.getByLabelText('Key Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Production key'));
    await user.click(screen.getByRole('button', { name: 'Configure' }));

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Please fill out the following fields: API key, Mode',
      ),
    );
  });

  it('creates a provider credential and clears the modal on success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    (CreateProviderKey as jest.Mock).mockResolvedValue(apiResponse(true));

    render(
      <CreateProviderCredentialDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentProvider="openai"
      />,
    );

    await screen.findByLabelText('API key');
    await user.click(screen.getByLabelText('Key Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Production key'));
    await user.click(screen.getByLabelText('API key'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.selectOptions(screen.getByLabelText('Mode'), String('chat'));
    await user.click(screen.getByRole('button', { name: 'Configure' }));

    await waitFor(() => expect(CreateProviderKey).toHaveBeenCalledTimes(1));

    const request = (CreateProviderKey as jest.Mock).mock.calls[0][1];
    expect(request.provider).toBe('openai');
    expect(request.name).toBe('Production key');
    expect(Struct.fromJavaScript).toHaveBeenCalledWith({
      api_key: 'secret',
      mode: 'chat',
    });
    expect(mockShowLoader).toHaveBeenCalledTimes(1);
    expect(mockHideLoader).toHaveBeenCalledTimes(1);
    expect(mockReloadProviderCredentials).toHaveBeenCalledTimes(1);
    expect(setModalOpen).toHaveBeenCalledWith(false);
    expect(toast.success).toHaveBeenCalledWith(
      'Provider credential have been successfully added to the vault.',
    );
  });

  it('shows API and rejected request errors', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateProviderKey as jest.Mock).mockResolvedValue(
      apiResponse(false, 'Credential already exists'),
    );

    render(
      <CreateProviderCredentialDialog
        modalOpen
        setModalOpen={jest.fn()}
        currentProvider="openai"
      />,
    );

    await screen.findByLabelText('API key');
    await user.click(screen.getByLabelText('Key Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Production key'));
    await user.click(screen.getByLabelText('API key'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.selectOptions(screen.getByLabelText('Mode'), String('chat'));
    await user.click(screen.getByRole('button', { name: 'Configure' }));

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Credential already exists',
      ),
    );

    (CreateProviderKey as jest.Mock).mockRejectedValue(new Error('network'));
    await user.click(screen.getByRole('button', { name: 'Configure' }));

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(
        'Unable to create provider credential, please try again later.',
      ),
    );
  });

  it('closes from the header and cancel actions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();

    render(
      <CreateProviderCredentialDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentProvider="openai"
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Close' }));
    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(setModalOpen).toHaveBeenCalledWith(false);
    expect(setModalOpen).toHaveBeenCalledTimes(2);
  });
});

describe('ProviderCredentialForm', () => {
  it('notifies callers when form controls change', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onProviderChange = jest.fn();
    const onKeyNameChange = jest.fn();
    const onConfigChange = jest.fn();
    const onCancel = jest.fn();
    const onSubmit = jest.fn();

    render(
      <ProviderCredentialForm
        config={{ api_key: 'secret' }}
        error="Invalid credential"
        isConfigFieldRequired={() => true}
        keyName="Production key"
        keyNameInputId="provider-key-name"
        onCancel={onCancel}
        onConfigChange={onConfigChange}
        onKeyNameChange={onKeyNameChange}
        onProviderChange={onProviderChange}
        onSubmit={onSubmit}
        provider={mockOpenaiProvider}
        providers={[mockOpenaiProvider]}
        loading
      />,
    );

    await user.click(screen.getByLabelText('Key Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('New key'));
    await user.click(screen.getByLabelText('Base URL (optional)'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('https://api.example.test'));
    await user.selectOptions(
      screen.getByLabelText('Mode'),
      String('responses'),
    );
    await user.click(screen.getByRole('button', { name: 'Add headers' }));
    await user.click(screen.getByLabelText('Key'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('X-Trace'));
    await user.click(screen.getByLabelText('Value'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('trace-1'));
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    await user.click(screen.getByRole('button', { name: 'Configure' }));

    expect(onKeyNameChange).toHaveBeenCalledWith('New key');
    expect(onConfigChange).toHaveBeenCalledWith(
      'base_url',
      'https://api.example.test',
    );
    expect(onConfigChange).toHaveBeenCalledWith('mode', 'responses');
    expect(onConfigChange).toHaveBeenCalledWith(
      'headers',
      '{"X-Trace":"trace-1"}',
    );
    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('alert')).toHaveTextContent('Invalid credential');
  });
});

describe('CredentialKeyValueField', () => {
  it('parses, edits, adds, and removes entries as JSON', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <CredentialKeyValueField
        name="headers"
        label="Headers"
        value='{"Authorization":"Bearer token"}'
        onChange={onChange}
      />,
    );

    await user.click(screen.getByDisplayValue('Authorization'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('X-Trace'));
    expect(onChange).toHaveBeenLastCalledWith('{"X-Trace":"Bearer token"}');

    await user.click(screen.getByDisplayValue('Bearer token'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('trace-1'));
    expect(onChange).toHaveBeenLastCalledWith('{"X-Trace":"trace-1"}');

    await user.click(screen.getByRole('button', { name: 'Remove' }));
    expect(onChange).toHaveBeenLastCalledWith('');
  });

  it('handles invalid JSON and helper serialization', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <CredentialKeyValueField
        name="headers"
        label="Headers"
        value="not-json"
        onChange={onChange}
      />,
    );

    expect(screen.getByText(/No entries yet/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Add headers' }));
    await user.click(screen.getByLabelText('Key'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('X-Request-ID'));
    await user.click(screen.getByLabelText('Value'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('req-1'));

    expect(onChange).toHaveBeenLastCalledWith('{"X-Request-ID":"req-1"}');
    expect(parseCredentialEntries('[]')).toEqual([]);
    expect(parseCredentialEntries('{"count":1}')).toEqual([
      { key: 'count', value: '1' },
    ]);
    expect(
      serializeCredentialEntries([
        { key: '', value: 'ignored' },
        { key: 'X-Test', value: 'yes' },
      ]),
    ).toBe('{"X-Test":"yes"}');
  });

  it('renders credential entries as a table form', () => {
    render(
      <CredentialKeyValueField
        name="headers"
        label="Headers"
        value='{"X-Trace":"trace-1"}'
        onChange={jest.fn()}
      />,
    );

    expect(screen.getByRole('table')).toHaveClass('border-border-subtle');
    expect(screen.getByLabelText('Key').closest('td')).toHaveClass(
      'border-r',
      'p-0',
    );
    expect(screen.getByLabelText('Value').closest('td')).toHaveClass(
      'border-r',
      'p-0',
    );
    expect(
      screen.getByRole('button', { name: 'Remove' }).closest('td'),
    ).toHaveClass('w-10', 'p-0', 'text-center');
  });
});

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import {
  ConfigureTelephonyComponent,
  GetDefaultTelephonyConfigIfInvalid,
  TelephonyProvider,
  ValidateTelephonyOptions,
} from '@/app/components/domain/providers/telephony';
import { TELEPHONY_PROVIDER } from '@/providers';

jest.mock('@/app/components/ui/primitives', () => ({
  FormLabel: ({ children, htmlFor }: any) => (
    <label htmlFor={htmlFor}>{children}</label>
  ),
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: ({ id, value, onChange, placeholder }: any) => (
    <input
      id={id}
      value={value ?? ''}
      onChange={onChange}
      placeholder={placeholder}
    />
  ),
}));

jest.mock('@/app/components/domain/providers/help-label', () => ({
  HelpToggletip: ({ helpText, label }: any) =>
    helpText ? <span aria-label={`${label} help`}>{helpText}</span> : null,
}));

jest.mock('@carbon/react', () => {
  const React = require('react');

  return {
    Dropdown: ({
      id,
      itemToString,
      label,
      items = [],
      selectedItem,
      onChange,
      titleText,
    }: any) => (
      <div>
        {titleText ? <span>{titleText}</span> : null}
        <span data-testid={`${id}-empty-name`}>{itemToString?.(null)}</span>
        <select
          id={id}
          aria-label={label || 'dropdown'}
          value={selectedItem?.code || ''}
          onChange={e => {
            const selected = items.find(
              (item: any) => item.code === e.target.value,
            );
            onChange?.({ selectedItem: selected || null });
          }}
        >
          <option value="">Select</option>
          {items.map((item: any) => (
            <option key={item.code} value={item.code}>
              {itemToString(item)}
            </option>
          ))}
        </select>
      </div>
    ),
    Select: ({ id, labelText, value, onChange, children }: any) => (
      <div>
        {labelText ? <label htmlFor={id}>{labelText}</label> : null}
        <select id={id} value={value ?? ''} onChange={onChange}>
          {children}
        </select>
      </div>
    ),
    SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
  };
});

jest.mock('@/app/components/domain/dropdowns/credential-dropdown', () => ({
  CredentialDropdown: ({
    currentCredential,
    onChangeCredential,
    provider,
  }: any) => (
    <button
      data-current={currentCredential}
      data-provider={provider}
      type="button"
      onClick={() => onChangeCredential({ getId: () => 'cred-1' })}
    >
      Pick credential
    </button>
  ),
}));

const meta = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

const getMetadataValue = (parameters: Metadata[], key: string) =>
  parameters.find(p => p.getKey() === key)?.getValue();

describe('Telephony provider runtime parity', () => {
  it('all active telephony providers are selectable', () => {
    expect(TELEPHONY_PROVIDER.length).toBeGreaterThan(0);
    for (const provider of TELEPHONY_PROVIDER) {
      expect(typeof provider.code).toBe('string');
      expect(provider.code.length).toBeGreaterThan(0);
    }
  });

  it.each([
    [
      'twilio',
      [meta('rapida.credential_id', 'cred-1'), meta('phone', '+15551234567')],
    ],
    [
      'exotel',
      [meta('rapida.credential_id', 'cred-1'), meta('phone', '+15551234567')],
    ],
    [
      'vonage',
      [meta('rapida.credential_id', 'cred-1'), meta('phone', '+15551234567')],
    ],
    [
      'sip',
      [meta('rapida.credential_id', 'cred-1'), meta('phone', '+15551234567')],
    ],
    [
      'asterisk',
      [
        meta('rapida.credential_id', 'cred-1'),
        meta('context', 'internal'),
        meta('extension', '1002'),
        meta('phone', '+15551234567'),
      ],
    ],
  ])('%s validates required telephony options', (provider, options) => {
    expect(ValidateTelephonyOptions(provider, options)).toBe(true);
  });

  it('returns false for unknown telephony provider', () => {
    expect(ValidateTelephonyOptions('unknown-telephony', [])).toBe(false);
  });

  it('returns false for telnyx until config-backed telephony schema is added', () => {
    expect(
      ValidateTelephonyOptions('telnyx', [
        meta('rapida.credential_id', 'cred-1'),
        meta('phone', '+15551234567'),
      ]),
    ).toBe(false);
  });

  it('returns false for invalid vonage phone number format', () => {
    expect(
      ValidateTelephonyOptions('vonage', [
        meta('rapida.credential_id', 'cred-1'),
        meta('phone', 'abc'),
      ]),
    ).toBe(false);
  });

  it('returns false when config validation fails before provider-specific checks', () => {
    expect(
      ValidateTelephonyOptions('twilio', [meta('phone', '+15551234567')]),
    ).toBe(false);
  });

  it('renders only the provider selector before a telephony provider is selected', () => {
    render(
      <TelephonyProvider
        provider=""
        parameters={[]}
        onChangeProvider={() => undefined}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.getByLabelText('Select telephony provider')).toHaveValue('');
    expect(screen.queryByText('Pick credential')).not.toBeInTheDocument();
    expect(
      screen.queryByPlaceholderText('Enter your Twilio phone number'),
    ).not.toBeInTheDocument();
  });

  it('updates provider and ignores empty provider selections', async () => {
    const user = userEvent.setup();
    const onChangeProvider = jest.fn();
    const onChangeParameter = jest.fn();
    render(
      <TelephonyProvider
        provider="twilio"
        parameters={[meta('phone', '+15551234567')]}
        onChangeProvider={onChangeProvider}
        onChangeParameter={onChangeParameter}
      />,
    );

    expect(screen.getByText('Telephony provider')).toBeInTheDocument();
    expect(screen.getByLabelText('Telephony provider help')).toHaveTextContent(
      'Select a telephony provider for inbound and outbound phone calls.',
    );
    expect(
      screen.getByTestId('telephony-provider-empty-name'),
    ).toBeEmptyDOMElement();

    await user.selectOptions(
      screen.getByLabelText('Select telephony provider'),
      String('vonage'),
    );
    expect(onChangeProvider).toHaveBeenCalledWith('vonage');
    expect(onChangeParameter).toHaveBeenCalledTimes(1);
    const providerDefaults = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(providerDefaults.map(p => p.getKey())).toEqual([
      'rapida.credential_id',
      'phone',
    ]);
    expect(getMetadataValue(providerDefaults, 'phone')).toBe('+15551234567');

    await user.selectOptions(
      screen.getByLabelText('Select telephony provider'),
      String(''),
    );
    expect(onChangeProvider).toHaveBeenCalledTimes(1);
    expect(onChangeParameter).toHaveBeenCalledTimes(1);
  });

  it('adds and replaces credential metadata from TelephonyProvider UI interactions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeParameter = jest.fn();
    const { rerender } = render(
      <TelephonyProvider
        provider="twilio"
        parameters={[meta('phone', '+15551234567')]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Pick credential' }));
    expect(onChangeParameter).toHaveBeenCalled();
    const params = onChangeParameter.mock.calls[
      onChangeParameter.mock.calls.length - 1
    ][0] as Metadata[];
    expect(
      params.find(p => p.getKey() === 'rapida.credential_id')?.getValue(),
    ).toBe('cred-1');

    rerender(
      <TelephonyProvider
        provider="twilio"
        parameters={[
          meta('rapida.credential_id', 'cred-old'),
          meta('phone', '+15551234567'),
        ]}
        onChangeProvider={() => undefined}
        onChangeParameter={onChangeParameter}
      />,
    );

    const credentialButton = screen.getByRole('button', {
      name: 'Pick credential',
    });
    expect(credentialButton).toHaveAttribute('data-current', 'cred-old');
    expect(credentialButton).toHaveAttribute('data-provider', 'twilio');

    await user.click(credentialButton);
    const replacedParams = onChangeParameter.mock.calls[
      onChangeParameter.mock.calls.length - 1
    ][0] as Metadata[];
    expect(getMetadataValue(replacedParams, 'rapida.credential_id')).toBe(
      'cred-1',
    );
    expect(getMetadataValue(replacedParams, 'phone')).toBe('+15551234567');
  });

  it('drops runtime-only sip keys while keeping supported keys', () => {
    const normalized = GetDefaultTelephonyConfigIfInvalid('sip', [
      meta('rapida.credential_id', 'cred-1'),
      meta('phone', '+15551234567'),
      meta('rapida.sip_inbound', 'true'),
      meta('rapida.sip_error', 'timeout'),
      meta('rapida.sip_retry_count', '10'),
    ]);
    expect(normalized.map(x => x.getKey())).toEqual([
      'rapida.credential_id',
      'phone',
      'rapida.sip_inbound',
    ]);
    expect(
      normalized.find(x => x.getKey() === 'rapida.sip_error'),
    ).toBeUndefined();
    expect(
      normalized.find(x => x.getKey() === 'rapida.sip_retry_count'),
    ).toBeUndefined();
  });

  it('returns provider defaults on switch baseline', () => {
    const asteriskDefaults = GetDefaultTelephonyConfigIfInvalid('asterisk', []);
    expect(asteriskDefaults.map(x => x.getKey())).toEqual([
      'rapida.credential_id',
      'context',
      'extension',
      'phone',
    ]);
    expect(
      asteriskDefaults.find(x => x.getKey() === 'context')?.getValue(),
    ).toBe('');
  });

  it('keeps credential and fills config defaults for exotel', () => {
    const defaults = GetDefaultTelephonyConfigIfInvalid('exotel', [
      meta('rapida.credential_id', 'cred-7'),
      meta('phone', '+15550001111'),
    ]);
    expect(defaults.map(x => x.getKey())).toEqual([
      'rapida.credential_id',
      'phone',
      'app_id',
    ]);
    expect(
      defaults.find(x => x.getKey() === 'rapida.credential_id')?.getValue(),
    ).toBe('cred-7');
    expect(defaults.find(x => x.getKey() === 'app_id')?.getValue()).toBe('');
  });

  it('returns empty defaults for unknown provider', () => {
    expect(GetDefaultTelephonyConfigIfInvalid('unknown-telephony', [])).toEqual(
      [],
    );
  });

  it('renders telephony config only when provider config exists', () => {
    const props = {
      provider: 'unknown-telephony',
      parameters: [],
      onChangeProvider: () => undefined,
      onChangeParameter: () => undefined,
    };
    const { rerender } = render(<ConfigureTelephonyComponent {...props} />);

    expect(
      screen.queryByLabelText('Select telephony provider'),
    ).not.toBeInTheDocument();

    rerender(<ConfigureTelephonyComponent {...props} provider="twilio" />);

    expect(
      screen.getByPlaceholderText('Enter your Twilio phone number'),
    ).toBeInTheDocument();
  });

  it('renders provider-specific json fields when switching providers', async () => {
    const user = userEvent.setup();
    const Harness = () => {
      const [provider, setProvider] = React.useState('exotel');
      const [parameters, setParameters] = React.useState<Metadata[]>(
        GetDefaultTelephonyConfigIfInvalid('exotel', []),
      );

      return (
        <TelephonyProvider
          provider={provider}
          parameters={parameters}
          onChangeProvider={setProvider}
          onChangeParameter={setParameters}
        />
      );
    };

    render(<Harness />);

    expect(
      screen.getByPlaceholderText('Enter exotel phone number'),
    ).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('Enter exotel applet app_id'),
    ).toBeInTheDocument();

    await user.selectOptions(
      screen.getByLabelText('Select telephony provider'),
      String('sip'),
    );

    expect(
      screen.queryByPlaceholderText('Enter exotel applet app_id'),
    ).not.toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('e.g., +15551234567'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Accept inbound calls')).toBeInTheDocument();

    await user.selectOptions(
      screen.getByLabelText('Select telephony provider'),
      String('asterisk'),
    );

    expect(screen.getByPlaceholderText('e.g., internal')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('e.g., 1002')).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('e.g., +15559876543'),
    ).toBeInTheDocument();
  });
});

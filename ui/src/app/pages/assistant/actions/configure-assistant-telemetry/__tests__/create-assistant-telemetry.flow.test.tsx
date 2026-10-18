import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { CreateAssistantTelemetry } from '@/app/pages/assistant/actions/configure-assistant-telemetry/create-assistant-telemetry';
import { CreateAssistantConfiguration, Metadata } from '@rapidaai/react';
import {
  GetDefaultTelemetryIfInvalid,
  ValidateTelemetry,
} from '@/app/components/domain/providers/telemetry/provider';

const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockNavigator = {
  goBack: jest.fn(),
  goToAssistantTelemetry: jest.fn(),
};

const meta = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

jest.mock('@rapidaai/react', () => {
  class ConnectionConfig {
    static WithDebugger(config: unknown) {
      return config;
    }
  }
  class Metadata {
    private key = '';
    private value = '';
    setKey(v: string) {
      this.key = v;
    }
    getKey() {
      return this.key;
    }
    setValue(v: string) {
      this.value = v;
    }
    getValue() {
      return this.value;
    }
  }
  class CreateAssistantConfigurationRequest {
    assistantid = '';
    configurationtype = '';
    provider = '';
    enabled = false;
    optionsList: Metadata[] = [];
    setAssistantid(v: string) {
      this.assistantid = v;
    }
    setConfigurationtype(v: string) {
      this.configurationtype = v;
    }
    setProvider(v: string) {
      this.provider = v;
    }
    setEnabled(v: boolean) {
      this.enabled = v;
    }
    setOptionsList(v: Metadata[]) {
      this.optionsList = v;
    }
  }
  return {
    ConnectionConfig,
    Metadata,
    CreateAssistantConfigurationRequest,
    CreateAssistantConfiguration: jest.fn(),
  };
});

jest.mock('@/providers', () => ({
  TELEMETRY_PROVIDER: [
    { code: 'otlp_http', name: 'OTLP HTTP' },
    { code: 'otlp_grpc', name: 'OTLP gRPC' },
  ],
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({ authId: 'u1', token: 't1', projectId: 'p1' }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => mockNavigator,
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    showDialog: (cb: () => void) => cb(),
    ConfirmDialogComponent: () => null,
  }),
}));

jest.mock('@/app/components/domain/providers/telemetry/provider', () => ({
  GetDefaultTelemetryIfInvalid: jest.fn(),
  ValidateTelemetry: jest.fn(),
}));

jest.mock('@/app/components/domain/providers/telemetry', () => ({
  TelemetryProvider: ({ onChangeProvider }: any) => (
    <button type="button" onClick={() => onChangeProvider('otlp_grpc')}>
      Switch provider
    </button>
  ),
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({ children, isLoading: _, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SecondaryButton: ({ children, isLoading: _, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@/app/components/ui/primitives/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/ui/feedback/notification', () => ({
  Notification: ({ subtitle }: any) => <div>{subtitle}</div>,
}));

jest.mock('@/app/components/ui/primitives/input-group', () => ({
  InputGroup: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@carbon/react', () => ({
  ButtonSet: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('react-hot-toast/headless', () => ({
  success: jest.fn(),
  error: jest.fn(),
}));

describe('Create assistant telemetry flow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (GetDefaultTelemetryIfInvalid as jest.Mock).mockImplementation(
      (provider: string, parameters: Metadata[]) => [
        ...(parameters || []),
        meta('telemetry.provider', provider),
      ],
    );
    (ValidateTelemetry as jest.Mock).mockReturnValue(undefined);
    (CreateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
    });
  });

  it('shows validation error when telemetry config is invalid', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (ValidateTelemetry as jest.Mock).mockReturnValue(
      'Please provide a valid telemetry endpoint.',
    );
    render(<CreateAssistantTelemetry assistantId="assistant-1" />);

    await user.click(
      screen.getByRole('button', { name: 'Configure telemetry' }),
    );

    expect(
      screen.getByText('Please provide a valid telemetry endpoint.'),
    ).toBeInTheDocument();
    expect(CreateAssistantConfiguration).not.toHaveBeenCalled();
  });

  it('switching provider rehydrates defaults from credential-only parameters', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (GetDefaultTelemetryIfInvalid as jest.Mock)
      .mockReturnValueOnce([
        meta('rapida.credential_id', 'cred-1'),
        meta('telemetry.endpoint', 'https://old-endpoint'),
      ])
      .mockImplementation((_provider: string, params: Metadata[]) => params);

    render(<CreateAssistantTelemetry assistantId="assistant-1" />);
    await user.click(screen.getByRole('button', { name: 'Switch provider' }));

    expect(GetDefaultTelemetryIfInvalid).toHaveBeenNthCalledWith(
      2,
      'otlp_grpc',
      [expect.objectContaining({})],
    );
    const params = (GetDefaultTelemetryIfInvalid as jest.Mock).mock
      .calls[1][1] as Metadata[];
    expect(params).toHaveLength(1);
    expect(params[0].getKey()).toBe('rapida.credential_id');
    expect(params[0].getValue()).toBe('cred-1');
  });

  it('creates telemetry provider successfully and navigates back to telemetry listing', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<CreateAssistantTelemetry assistantId="assistant-1" />);

    await user.click(
      screen.getByRole('button', { name: 'Configure telemetry' }),
    );

    await waitFor(() => {
      expect(CreateAssistantConfiguration).toHaveBeenCalledTimes(1);
    });

    expect(CreateAssistantConfiguration).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        assistantid: 'assistant-1',
        configurationtype: 'telemetry',
        provider: 'otlp_http',
        enabled: true,
        optionsList: expect.any(Array),
      }),
      expect.objectContaining({
        'x-auth-id': 'u1',
        authorization: 't1',
        'x-project-id': 'p1',
      }),
    );
    expect(mockShowLoader).toHaveBeenCalled();
    expect(mockHideLoader).toHaveBeenCalled();
    expect(mockNavigator.goToAssistantTelemetry).toHaveBeenCalledWith(
      'assistant-1',
    );
  });

  it('applies page background tokens for light/dark mode parity', () => {
    const { container } = render(
      <CreateAssistantTelemetry assistantId="assistant-1" />,
    );

    const pageRoot = container.querySelector('section > div') as HTMLElement;
    expect(pageRoot).toBeInTheDocument();
    expect(pageRoot).toHaveClass('bg-white');
    expect(pageRoot).toHaveClass('dark:bg-gray-900');
  });
});

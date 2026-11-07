import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { CreateAgentKit } from '../create-agentkit';
import { CreateWebsocket } from '../create-websocket';
import { CreateAgentKitVersion } from '../../create-assistant-version/create-agent-kit-version';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { GetAssistant } from '@rapidaai/react';

let mockParams: Record<string, string | undefined> = {};
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockGoBack = jest.fn();
const mockGoToAssistantVersions = jest.fn();

jest.mock('@rapidaai/react', () => {
  class ConnectionConfig {
    constructor(_: unknown) {}
    static WithDebugger(config: unknown) {
      return config;
    }
  }

  class CreateAssistantProviderRequest {
    static CreateAssistantProviderAgentkit = class {
      private metadata = new Map<string, string>();
      setAgentkiturl(_: string) {}
      setCertificate(_: string) {}
      setTransportsecurity(_: string) {}
      setTlsverification(_: string) {}
      setTlsservername(_: string) {}
      setConnecttimeoutms(_: number) {}
      setKeepalivetimems(_: number) {}
      setKeepalivetimeoutms(_: number) {}
      setMaxrecvmessagebytes(_: number) {}
      setMaxsendmessagebytes(_: number) {}
      getMetadataMap() {
        return this.metadata;
      }
    };
    static CreateAssistantProviderWebsocket = class {
      private headers = new Map<string, string>();
      private connectionParameters = new Map<string, string>();
      setWebsocketurl(_: string) {}
      getHeadersMap() {
        return this.headers;
      }
      getConnectionparametersMap() {
        return this.connectionParameters;
      }
    };
    setAgentkit(_: unknown) {}
    setWebsocket(_: unknown) {}
    setAssistantid(_: string) {}
    setDescription(_: string) {}
  }

  class CreateAssistantRequest {
    setAssistantprovider(_: unknown) {}
    setName(_: string) {}
    setTagsList(_: string[]) {}
    setDescription(_: string) {}
  }

  class GetAssistantRequest {
    setAssistantdefinition(_: unknown) {}
  }

  class AssistantDefinition {
    setAssistantid(_: string) {}
  }

  class Assistant {
    getId() {
      return 'assistant-1';
    }
  }

  return {
    Assistant,
    AssistantDefinition,
    ConnectionConfig,
    CreateAssistantProviderRequest,
    CreateAssistantRequest,
    GetAssistantRequest,
    CreateAssistant: jest.fn(() =>
      Promise.resolve({ getSuccess: () => false }),
    ),
    CreateAssistantProvider: jest.fn(() =>
      Promise.resolve({ getSuccess: () => false }),
    ),
    GetAssistant: jest.fn(),
  };
});

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => mockParams,
}));

jest.mock('@/configs', () => ({ connectionConfig: {} }));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    token: 'token',
    projectId: 'project-1',
  }),
  useCredential: () => ['user-1', 'token', 'project-1'],
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goBack: mockGoBack,
    goToAssistant: jest.fn(),
    goToConfigureDebugger: jest.fn(),
    goToConfigureWeb: jest.fn(),
    goToConfigureCall: jest.fn(),
    goToConfigureApi: jest.fn(),
    goToCreateAssistantAnalysis: jest.fn(),
    goToCreateAssistantWebhook: jest.fn(),
    goToAssistantListing: jest.fn(),
    goToAssistantVersions: mockGoToAssistantVersions,
  }),
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    showDialog: (callback: () => void) => callback(),
    ConfirmDialogComponent: () => null,
  }),
}));

jest.mock('@/app/components/ui/composites/create-full-page', () => ({
  ...jest.requireActual('@/app/components/ui/composites/create-full-page'),
  CreateFullPageWithHeader: ({
    activeStep,
    description,
    errorMessage,
    steps,
  }: any) => {
    const active =
      steps.find((item: any) => item.code === activeStep) ?? steps[0];
    return (
      <div>
        <h1>{description}</h1>
        {errorMessage ? <div>{errorMessage}</div> : null}
        <p>{active.description}</p>
        <div>{active.body}</div>
        <div>
          {Array.isArray(active.actions)
            ? active.actions.map(
                (action: React.ReactElement, index: number) => (
                  <div key={index}>{action}</div>
                ),
              )
            : active.actions}
        </div>
      </div>
    );
  },
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  GhostButton: ({ children, isLoading: _isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  PrimaryButton: ({ children, isLoading: _isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SecondaryButton: ({ children, isLoading: _isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@carbon/react', () => ({
  Button: ({ children, kind: _kind, size: _size, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  ButtonSet: ({ children }: any) => <div>{children}</div>,
  Form: ({ children, ...props }: any) => <form {...props}>{children}</form>,
  InlineNotification: ({ subtitle, title }: any) => (
    <div role="alert">
      {title} {subtitle}
    </div>
  ),
  ProgressIndicator: ({ children }: any) => <ol>{children}</ol>,
  ProgressStep: ({ label, secondaryLabel }: any) => (
    <li>
      <span>{label}</span>
      <span>{secondaryLabel}</span>
    </li>
  ),
  Slider: ({ id, labelText }: any) => (
    <input aria-label={labelText} id={id} readOnly />
  ),
  Toggletip: ({ children }: any) => <span>{children}</span>,
  ToggletipButton: ({ children, label }: any) => (
    <button type="button" aria-label={label}>
      {children ?? label}
    </button>
  ),
  ToggletipContent: ({ children }: any) => <span>{children}</span>,
}));

jest.mock('@carbon/icons-react', () => {
  const Icon =
    (testId: string) =>
    ({ className, size, strokeWidth }: any) => (
      <svg
        className={className}
        data-size={size}
        data-stroke-width={strokeWidth}
        data-testid={testId}
      />
    );

  return {
    ChevronDown: Icon('chevron-down-icon'),
    ChevronRight: Icon('chevron-right-icon'),
    Code: Icon('code-icon'),
    ConnectionSignal: Icon('connection-signal-icon'),
    Debug: Icon('debug-icon'),
    Globe: Icon('globe-icon'),
    Information: Icon('information-icon'),
    Launch: Icon('launch-icon'),
    Phone: Icon('phone-icon'),
  };
});

jest.mock('@/app/components/app-shell/helmet', () => ({ Helmet: () => null }));
jest.mock('@/app/components/ui/primitives/fieldset', () => ({
  FieldSet: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/ui/primitives/form-label', () => ({
  FormLabel: ({ children, ...props }: any) => (
    <label {...props}>{children}</label>
  ),
}));
jest.mock('@/app/components/ui/primitives/input', () => ({
  Input: (props: any) => <input {...props} />,
}));
jest.mock('@/app/components/ui/primitives/select', () => ({
  Select: ({ options = [], ...props }: any) => (
    <select {...props}>
      {options.map((option: { name: string; value: string }) => (
        <option key={option.value} value={option.value}>
          {option.name}
        </option>
      ))}
    </select>
  ),
}));
jest.mock('@/app/components/ui/primitives/textarea', () => ({
  Textarea: (props: any) => <textarea {...props} />,
}));
jest.mock('@/app/components/ui/composites/tag-input', () => ({
  TagInput: () => null,
}));
jest.mock('@/app/components/domain/tags/assistant-tags', () => ({
  AssistantTag: [],
}));
jest.mock(
  '@/app/components/layout/container/message/notice-block/doc-notice-block',
  () => ({
    DocNoticeBlock: ({ children }: any) => <div>{children}</div>,
  }),
);
jest.mock('@/app/components/layout/container/message/notice-block', () => ({
  YellowNoticeBlock: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/domain/external-api/api-parameter', () => ({
  APiParameter: () => null,
}));
jest.mock('@/app/components/ui/primitives/input-helper', () => ({
  InputHelper: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/ui/editor/code-editor', () => ({
  CodeEditor: (props: any) => <textarea {...props} />,
}));
jest.mock('@/app/components/layout/blocks/section-divider', () => ({
  SectionDivider: ({ label }: any) => <div>{label}</div>,
}));
jest.mock('@/app/components/ui/feedback/error-container', () => ({
  ErrorContainer: ({ code, title }: any) => (
    <div>
      <span>{code}</span>
      <span>{title}</span>
    </div>
  ),
}));
jest.mock('@/utils', () => ({
  ...jest.requireActual('@/utils'),
  randomMeaningfullName: () => 'assistant-default',
}));
jest.mock('react-hot-toast/headless', () => ({
  __esModule: true,
  default: { success: jest.fn() },
}));

const theme: ThemeManifest = {
  schemaVersion: 1,
  id: 'custom',
  brand: {
    name: 'Acme Voice',
  },
  links: {
    documentation: '#',
    source: '#',
    support: '#',
    terms: '#',
    privacy: '#',
  },
  defaultMode: 'light',
  allowModeSelection: false,
  colors: {
    light: {
      carbonTheme: 'white',
      primary: '#111111',
      primaryHover: '#222222',
      primaryActive: '#333333',
      onPrimary: '#ffffff',
    },
    dark: {
      carbonTheme: 'g100',
      primary: '#eeeeee',
      primaryHover: '#dddddd',
      primaryActive: '#cccccc',
      onPrimary: '#000000',
    },
  },
};

const renderWithTheme = (component: React.ReactElement) =>
  render(<ThemeProvider theme={theme}>{component}</ThemeProvider>);

const getAgentKitProvider = () => ({
  getUrl: () => 'agent.example.com:5051',
  getCertificate: () => '',
  getTransportsecurity: () => '',
  getTlsverification: () => '',
  getTlsservername: () => '',
  getConnecttimeoutms: () => 0,
  getKeepalivetimems: () => 0,
  getKeepalivetimeoutms: () => 0,
  getMaxrecvmessagebytes: () => 0,
  getMaxsendmessagebytes: () => 0,
  getMetadataMap: () => new Map<string, string>(),
});

describe('AgentKit brand copy', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
    mockParams = { assistantId: 'assistant-1' };
    jest.clearAllMocks();
    (GetAssistant as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({
        getAssistantprovideragentkit: () => getAgentKitProvider(),
      }),
    });
  });

  it('renders the create AgentKit page with the configured brand', () => {
    renderWithTheme(<CreateAgentKit />);

    expect(document.body).toHaveTextContent(
      'Connect an AgentKit endpoint, define its assistant profile, then choose how to deploy it.',
    );
    expect(document.body).toHaveTextContent(
      'where your Acme Voice AgentKit is running',
    );
    expect(document.body).toHaveTextContent('Show advanced settings');
    expect(screen.getByTestId('chevron-down-icon')).toHaveAttribute(
      'data-size',
      '16',
    );
    expect(screen.getByTestId('chevron-down-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );
    expect(document.body).not.toHaveTextContent('Rapida AgentKit');
  });

  it('keeps the connection step active until the endpoint is valid', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithTheme(<CreateAgentKit />);

    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Enter a valid AgentKit endpoint in host:port format.',
    );
    expect(
      screen.getByPlaceholderText('agent.your-domain.com:5051'),
    ).toBeVisible();

    await user.type(
      screen.getByPlaceholderText('agent.your-domain.com:5051'),
      'agent.example.com:5051',
    );
    await user.click(screen.getByRole('button', { name: 'Continue' }));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('moves to the profile step after a valid endpoint is entered', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithTheme(<CreateAgentKit />);

    await user.type(
      screen.getByPlaceholderText('agent.your-domain.com:5051'),
      'agent.example.com:5051',
    );
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(
      screen.getByPlaceholderText('e.g. customer-support-assistant'),
    ).toBeVisible();
    expect(
      screen.getByRole('button', { name: 'Create and continue' }),
    ).toBeVisible();
  });

  it('renders the create WebSocket page with the configured brand', () => {
    renderWithTheme(<CreateWebsocket />);

    expect(document.body).toHaveTextContent(
      'Connect your external AI agent to Acme Voice using a WebSocket endpoint.',
    );
    expect(screen.getByTestId('information-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );
    expect(screen.getByTestId('launch-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );
    expect(document.body).not.toHaveTextContent('Rapida');
  });

  it('renders the create AgentKit version page with the configured brand', async () => {
    renderWithTheme(<CreateAgentKitVersion />);

    expect(document.body).toHaveTextContent(
      'Provide the connection configuration for your Acme Voice AgentKit setup.',
    );
    expect(document.body).toHaveTextContent(
      'where your Acme Voice AgentKit is running',
    );
    expect(document.body).not.toHaveTextContent('Rapida AgentKit');
    await waitFor(() => expect(GetAssistant).toHaveBeenCalled());
  });
});

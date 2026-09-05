import React from 'react';
import { render, waitFor } from '@testing-library/react';
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

jest.mock('@/hooks', () => ({
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

jest.mock('@/app/components/form/tab-form', () => ({
  TabForm: ({ activeTab, errorMessage, form, formHeading }: any) => {
    const active = form.find((item: any) => item.code === activeTab) ?? form[0];
    return (
      <div>
        <h1>{formHeading}</h1>
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

jest.mock('@/app/components/carbon/button', () => ({
  PrimaryButton: ({ children, isLoading: _isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SecondaryButton: ({ children, isLoading: _isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@carbon/react', () => ({
  ButtonSet: ({ children }: any) => <div>{children}</div>,
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

jest.mock('@carbon/icons-react', () => ({
  ChevronDown: () => <span />,
  Information: () => <span />,
}));

jest.mock('lucide-react', () => ({
  Bug: () => <span />,
  ChevronRight: () => <span />,
  Code: () => <span />,
  ExternalLink: () => <span />,
  Globe: () => <span />,
  Info: () => <span />,
  PhoneCall: () => <span />,
}));

jest.mock('@/app/components/helmet', () => ({ Helmet: () => null }));
jest.mock('@/app/components/form/fieldset', () => ({
  FieldSet: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/form-label', () => ({
  FormLabel: ({ children }: any) => <label>{children}</label>,
}));
jest.mock('@/app/components/form/input', () => ({
  Input: (props: any) => <input {...props} />,
}));
jest.mock('@/app/components/form/select', () => ({
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
jest.mock('@/app/components/form/textarea', () => ({
  Textarea: (props: any) => <textarea {...props} />,
}));
jest.mock('@/app/components/form/tag-input', () => ({ TagInput: () => null }));
jest.mock('@/app/components/form/tag-input/assistant-tags', () => ({
  AssistantTag: [],
}));
jest.mock(
  '@/app/components/container/message/notice-block/doc-notice-block',
  () => ({
    DocNoticeBlock: ({ children }: any) => <div>{children}</div>,
  }),
);
jest.mock('@/app/components/container/message/notice-block', () => ({
  YellowNoticeBlock: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/external-api/api-parameter', () => ({
  APiParameter: () => null,
}));
jest.mock('@/app/components/input-helper', () => ({
  InputHelper: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/form/editor/code-editor', () => ({
  CodeEditor: (props: any) => <textarea {...props} />,
}));
jest.mock('@/app/components/blocks/section-divider', () => ({
  SectionDivider: ({ label }: any) => <div>{label}</div>,
}));
jest.mock('@/app/components/error-container', () => ({
  ErrorContainer: ({ code, title }: any) => (
    <div>
      <span>{code}</span>
      <span>{title}</span>
    </div>
  ),
}));
jest.mock('@/utils', () => ({
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
      primary: '#111111',
      primaryHover: '#222222',
      primaryActive: '#333333',
      onPrimary: '#ffffff',
    },
    dark: {
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
      'with the Acme Voice orchestration engine',
    );
    expect(document.body).toHaveTextContent(
      'where your Acme Voice AgentKit is running',
    );
    expect(document.body).not.toHaveTextContent('Rapida AgentKit');
  });

  it('renders the create WebSocket page with the configured brand', () => {
    renderWithTheme(<CreateWebsocket />);

    expect(document.body).toHaveTextContent(
      'Connect your external AI agent to Acme Voice using a WebSocket endpoint.',
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

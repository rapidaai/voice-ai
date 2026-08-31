import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { ForgotPasswordPage } from '@/app/pages/authentication/forgot-password';
import { CreateEndpointPage } from '@/app/pages/endpoint/actions/create-endpoint';
import { CreateNewVersionEndpointPage } from '@/app/pages/endpoint/actions/create-endpoint-version';
import { CreateAssistantPage } from '@/app/pages/assistant/actions/create-assistant';
import { CreateVersionAssistantPage } from '@/app/pages/assistant/actions/create-assistant-version';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import {
  CreateEndpointProviderModel,
  ForgotPassword,
  GetAssistant,
} from '@rapidaai/react';

let mockParams: Record<string, string | undefined> = {};
const mockNavigate = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockConfigPrompt = jest.fn();
const mockTextProvider = jest.fn();
const mockValidateTextProviderDefaultOptions = jest.fn();
const mockGetDefaultTextProviderConfigIfInvalid = jest.fn();
const mockGetDefaultTextProviderConfigOnProviderSwitch = jest.fn();

jest.mock('@rapidaai/react', () => {
  class ConnectionConfig {
    constructor(_: unknown) {}
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
  class EndpointProviderModelAttribute {
    setModelprovidername(_: string) {}
    setEndpointmodeloptionsList(_: unknown[]) {}
    setChatcompleteprompt(_: unknown) {}
    setDescription(_: string) {}
  }
  class EndpointAttribute {
    setName(_: string) {}
    setDescription(_: string) {}
  }
  class CreateAssistantProviderRequest {
    static CreateAssistantProviderModel = class {
      setTemplate(_: unknown) {}
      setModelprovidername(_: string) {}
      setAssistantmodeloptionsList(_: unknown[]) {}
    };
    setModel(_: unknown) {}
    setAssistantid(_: string) {}
    setDescription(_: string) {}
  }
  class CreateAssistantRequest {
    setAssistantprovider(_: unknown) {}
    setAssistanttoolsList(_: unknown[]) {}
    setName(_: string) {}
    setTagsList(_: string[]) {}
    setDescription(_: string) {}
  }
  class CreateAssistantToolRequest {
    setName(_: string) {}
    setDescription(_: string) {}
    setFields(_: unknown) {}
    setExecutionmethod(_: string) {}
    setExecutionoptionsList(_: unknown[]) {}
  }
  class GetAssistantRequest {
    setAssistantdefinition(_: unknown) {}
  }
  class AssistantDefinition {
    setAssistantid(_: string) {}
  }

  return {
    ConnectionConfig,
    Metadata,
    EndpointProviderModelAttribute,
    EndpointAttribute,
    CreateAssistantProviderRequest,
    CreateAssistantRequest,
    CreateAssistantToolRequest,
    GetAssistantRequest,
    AssistantDefinition,
    ForgotPassword: jest.fn(),
    CreateEndpoint: jest.fn(),
    GetEndpoint: jest.fn(),
    CreateEndpointProviderModel: jest.fn(),
    CreateAssistant: jest.fn(() => Promise.resolve({ getSuccess: () => false })),
    GetAssistant: jest.fn(() => Promise.resolve({ getSuccess: () => false })),
    CreateAssistantProvider: jest.fn(() => Promise.resolve({ getSuccess: () => false })),
  };
});

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => mockParams,
  useNavigate: () => mockNavigate,
}));

jest.mock('@carbon/react', () => {
  const actual = jest.requireActual('@carbon/react');
  return {
    ...actual,
    Toggletip: ({ children }: any) => <span>{children}</span>,
    ToggletipButton: ({ children, label }: any) => (
      <button type="button" aria-label={label}>
        {children || label}
      </button>
    ),
    ToggletipContent: ({ children }: any) => <span>{children}</span>,
  };
});

jest.mock('@/hooks', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({ authId: 'u1', token: 't1', projectId: 'p1' }),
}));

jest.mock('@/hooks/use-model', () => ({
  useAllProviderCredentials: () => ({ providerCredentials: [] }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goBack: jest.fn(),
    goToAssistant: jest.fn(),
    goToAssistantListing: jest.fn(),
    goToAssistantVersions: jest.fn(),
  }),
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    showDialog: (cb: () => void) => cb(),
    ConfirmDialogComponent: () => null,
  }),
}));

jest.mock('@/app/components/form/tab-form', () => ({
  TabForm: ({ activeTab, errorMessage, form, formHeading }: any) => {
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

jest.mock('@/app/components/providers/text', () => ({
  GetDefaultTextProviderConfigIfInvalid: (...args: unknown[]) =>
    mockGetDefaultTextProviderConfigIfInvalid(...args),
  GetDefaultTextProviderConfigOnProviderSwitch: (...args: unknown[]) =>
    mockGetDefaultTextProviderConfigOnProviderSwitch(...args),
  ValidateTextProviderDefaultOptions: (...args: unknown[]) =>
    mockValidateTextProviderDefaultOptions(...args),
  TextProvider: (props: any) => {
    mockTextProvider(props);
    return null;
  },
}));

jest.mock('@/app/components/configuration/config-prompt', () => ({
  ConfigPrompt: (props: any) => {
    mockConfigPrompt(props);
    return null;
  },
}));

jest.mock('@/app/components/tools', () => ({
  BuildinToolConfig: {},
}));

jest.mock('@/utils/prompt', () => ({
  ChatCompletePrompt: () => ({}),
  Prompt: () => ({ prompt: [], variables: [] }),
}));

jest.mock('@/utils', () => {
  const actual = jest.requireActual('@/utils');
  return {
    ...actual,
    randomMeaningfullName: () => 'assistant-default',
    randomString: () => 'seed',
  };
});

jest.mock('@/app/components/error-container', () => ({
  ErrorContainer: ({ title, code }: any) => (
    <div>
      <span>{code}</span>
      <span>{title}</span>
    </div>
  ),
}));

jest.mock('@/app/components/helmet', () => ({ Helmet: () => null }));
jest.mock('@/app/components/carbon/button', () => ({
  PrimaryButton: ({ children, isLoading: _, renderIcon: _r, hasIconOnly: _h, iconDescription: _d, ...props }: any) => <button {...props}>{children}</button>,
  SecondaryButton: ({ children, isLoading: _, renderIcon: _r, hasIconOnly: _h, iconDescription: _d, ...props }: any) => <button {...props}>{children}</button>,
  GhostButton: ({ children, isLoading: _, renderIcon: _r, hasIconOnly: _h, iconDescription: _d, ...props }: any) => <button {...props}>{children}</button>,
}));

jest.mock('@/app/components/base/modal/confirm-ui', () => () => null);
jest.mock('@/app/components/base/modal/configure-endpoint-prompt-modal/index', () => ({
  ConfigureEndpointPromptDialog: () => null,
}));
jest.mock('@/app/components/base/modal/assistant-configure-next-modal', () => ({
  ConfigureAssistantNextDialog: () => null,
}));
jest.mock('@/app/components/base/modal/assistant-configure-tool-modal', () => ({
  ConfigureAssistantToolDialog: () => null,
}));
jest.mock('@/app/components/base/modal/configure-assistant-template-modal', () => ({
  ConfigureAssistantTemplateDialog: () => null,
}));

jest.mock('@/app/components/container/message/notice-block', () => ({
  YellowNoticeBlock: () => null,
}));
jest.mock('@/app/components/container/message/notice-block/doc-notice-block', () => ({
  DocNoticeBlock: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/carbon/empty-state', () => ({
  EmptyState: () => null,
}));

jest.mock('@/app/components/blocks/section-divider', () => ({
  SectionDivider: () => null,
}));
jest.mock('@/app/components/base/corner-border', () => ({
  CornerBorderOverlay: () => null,
}));

jest.mock('@/app/components/form/input', () => ({
  Input: require('react').forwardRef((props: any, ref: any) => <input ref={ref} {...props} />),
}));
jest.mock('@/app/components/form/error-message', () => ({
  ErrorMessage: ({ message }: any) => (message ? <div>{message}</div> : null),
}));
jest.mock('@/app/components/form/fieldset', () => ({ FieldSet: ({ children }: any) => <div>{children}</div> }));
jest.mock('@/app/components/form-label', () => ({ FormLabel: ({ children }: any) => <label>{children}</label> }));

const getLatestConfigPromptProps = () =>
  mockConfigPrompt.mock.calls[mockConfigPrompt.mock.calls.length - 1]?.[0];

const getLatestTextProviderProps = () =>
  mockTextProvider.mock.calls[mockTextProvider.mock.calls.length - 1]?.[0];

const theme = developmentConfig.theme as unknown as ThemeManifest;

const renderWithTheme = (component: React.ReactElement) =>
  render(<ThemeProvider theme={theme}>{component}</ThemeProvider>);

describe('Requested create/update flow pages', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
    jest.clearAllMocks();
    mockParams = {};
    mockConfigPrompt.mockClear();
    mockTextProvider.mockClear();
    mockValidateTextProviderDefaultOptions.mockReset();
    mockGetDefaultTextProviderConfigIfInvalid.mockReset();
    mockGetDefaultTextProviderConfigOnProviderSwitch.mockReset();
    mockValidateTextProviderDefaultOptions.mockReturnValue(undefined);
    mockGetDefaultTextProviderConfigIfInvalid.mockImplementation(
      (_provider: string, current: unknown[] = []) => current,
    );
    mockGetDefaultTextProviderConfigOnProviderSwitch.mockImplementation(
      (_provider: string, current: unknown[] = []) => current,
    );

    (GetAssistant as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({
        getAssistantprovidermodel: () => ({
          getTemplate: () => ({}),
          getModelprovidername: () => 'azure-foundry',
          getAssistantmodeloptionsList: () => [],
        }),
      }),
    });
  });

  it('forgot password shows success message on success callback', async () => {
    (ForgotPassword as jest.Mock).mockImplementation((_cfg, _email, cb) => {
      cb(null, { getSuccess: () => true });
    });

    renderWithTheme(<ForgotPasswordPage />);
    fireEvent.change(screen.getByPlaceholderText('eg: john@example.com'), {
      target: { value: 'user@x.com' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Send Email' }));

    expect(await screen.findByText(/Thanks! An email was sent/i)).toBeInTheDocument();
  });

  it('create endpoint blocks continue when prompt template variables are missing', () => {
    render(<CreateEndpointPage />);
    fireEvent.click(screen.getByRole('button', { name: 'Configure instruction' }));
    expect(
      screen.getByText(
        'Please provide a valid prompt template, it should at least have one variable.',
      ),
    ).toBeInTheDocument();
  });

  it('create endpoint moves to define step after prompt variable edit', () => {
    render(<CreateEndpointPage />);

    const endpointPromptProps = getLatestConfigPromptProps();
    act(() => {
      endpointPromptProps.onChange({
        prompt: [{ role: 'system', content: 'Hi {{name}}' }],
        variables: [{ name: 'name', type: 'string', defaultvalue: '' }],
      });
    });

    fireEvent.click(
      screen.getByRole('button', { name: 'Configure instruction' }),
    );

    expect(screen.queryByText('Please define at least one variable.')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create endpoint' })).toBeInTheDocument();
    expect(screen.getByText('Endpoint name')).toBeInTheDocument();
    expect(screen.queryByText('Endpoint name *')).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Endpoint name information' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Endpoint description information' }),
    ).toBeInTheDocument();
  });

  it('create endpoint validates using changed provider', () => {
    render(<CreateEndpointPage />);

    const endpointTextProviderProps = getLatestTextProviderProps();
    act(() => {
      endpointTextProviderProps.onChangeProvider('openai');
    });

    const endpointPromptProps = getLatestConfigPromptProps();
    act(() => {
      endpointPromptProps.onChange({
        prompt: [{ role: 'system', content: 'Hello {{question}}' }],
        variables: [{ name: 'question', type: 'string', defaultvalue: '' }],
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Configure instruction' }));

    expect(mockValidateTextProviderDefaultOptions).toHaveBeenCalled();
    expect(mockValidateTextProviderDefaultOptions.mock.calls.at(-1)?.[0]).toBe('openai');
  });

  it('create endpoint version blocks continue when variables are missing', () => {
    render(<CreateNewVersionEndpointPage />);
    fireEvent.click(screen.getByRole('button', { name: 'Configure instruction' }));
    expect(screen.getByText('Please define at least one variable.')).toBeInTheDocument();
  });

  it('create endpoint version moves to commit step after prompt variable edit', () => {
    render(<CreateNewVersionEndpointPage />);

    const endpointVersionPromptProps = getLatestConfigPromptProps();
    act(() => {
      endpointVersionPromptProps.onChange({
        prompt: [{ role: 'system', content: 'Hi {{name}}' }],
        variables: [{ name: 'name', type: 'string', defaultvalue: '' }],
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Configure instruction' }));

    expect(screen.getByText('Version note')).toBeInTheDocument();
  });

  it('create endpoint version validates using changed provider', () => {
    render(<CreateNewVersionEndpointPage />);

    const endpointVersionTextProviderProps = getLatestTextProviderProps();
    act(() => {
      endpointVersionTextProviderProps.onChangeProvider('anthropic');
    });

    const endpointVersionPromptProps = getLatestConfigPromptProps();
    act(() => {
      endpointVersionPromptProps.onChange({
        prompt: [{ role: 'system', content: 'Hello {{topic}}' }],
        variables: [{ name: 'topic', type: 'string', defaultvalue: '' }],
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Configure instruction' }));

    expect(mockValidateTextProviderDefaultOptions).toHaveBeenCalled();
    expect(mockValidateTextProviderDefaultOptions.mock.calls.at(-1)?.[0]).toBe('anthropic');
  });

  it('redirects to endpoint versions after creating a new endpoint version', () => {
    mockParams = { endpointId: 'endpoint-1' };
    (CreateEndpointProviderModel as jest.Mock).mockImplementation(
      (_cfg, _endpointId, _attr, _auth, cb) => {
        cb(null, {
          getSuccess: () => true,
          getData: () => ({
            getEndpointid: () => 'endpoint-1',
          }),
        });
      },
    );

    render(<CreateNewVersionEndpointPage />);

    const endpointVersionPromptProps = getLatestConfigPromptProps();
    act(() => {
      endpointVersionPromptProps.onChange({
        prompt: [{ role: 'system', content: 'Hello {{topic}}' }],
        variables: [{ name: 'topic', type: 'string', defaultvalue: '' }],
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Configure instruction' }));
    fireEvent.click(screen.getByRole('button', { name: 'Create new version' }));

    expect(CreateEndpointProviderModel).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith(
      '/deployment/endpoint/endpoint-1/versions',
    );
  });

  it('create assistant blocks continue when prompt content is empty', () => {
    renderWithTheme(<CreateAssistantPage />);
    const assistantConfigPrompt = getLatestConfigPromptProps();
    expect(assistantConfigPrompt.showRuntimeReplacementHint).toBe(true);
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    expect(
      screen.getByText('Each prompt message must have a valid role and non-empty content.'),
    ).toBeInTheDocument();
  });

  it('create assistant moves forward after prompt content edit', () => {
    renderWithTheme(<CreateAssistantPage />);

    const assistantConfigPrompt = getLatestConfigPromptProps();
    act(() => {
      assistantConfigPrompt.onChange({
        prompt: [{ role: 'system', content: 'You are helper {{name}}' }],
        variables: [{ name: 'name', type: 'string', defaultvalue: '' }],
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

    expect(screen.getByRole('button', { name: 'Skip for now' })).toBeInTheDocument();
  });

  it('create assistant validates using changed provider', () => {
    renderWithTheme(<CreateAssistantPage />);

    const assistantTextProviderProps = getLatestTextProviderProps();
    act(() => {
      assistantTextProviderProps.onChangeProvider('openai');
    });

    const assistantConfigPrompt = getLatestConfigPromptProps();
    act(() => {
      assistantConfigPrompt.onChange({
        prompt: [{ role: 'system', content: 'Ready {{name}}' }],
        variables: [{ name: 'name', type: 'string', defaultvalue: '' }],
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

    expect(mockValidateTextProviderDefaultOptions).toHaveBeenCalled();
    expect(mockValidateTextProviderDefaultOptions.mock.calls.at(-1)?.[0]).toBe('openai');
  });

  it('create endpoint does not attach assistant runtime argument hints', () => {
    render(<CreateEndpointPage />);
    const endpointConfigPrompt = getLatestConfigPromptProps();
    expect(endpointConfigPrompt.showRuntimeReplacementHint).toBeUndefined();
  });

  it('create assistant version shows unavailable state when assistantId is missing', async () => {
    mockParams = {};
    render(<CreateVersionAssistantPage />);
    expect(screen.getByText('403')).toBeInTheDocument();
    expect(screen.getByText('Assistant not available')).toBeInTheDocument();
    await act(async () => {});
  });

  it('create assistant version moves to commit step after prompt edit', async () => {
    mockParams = { assistantId: 'a-1' };
    render(<CreateVersionAssistantPage />);

    const assistantVersionConfigPrompt = getLatestConfigPromptProps();
    act(() => {
      assistantVersionConfigPrompt.onChange({
        prompt: [{ role: 'system', content: 'Edited {{context}}' }],
        variables: [{ name: 'context', type: 'string', defaultvalue: '' }],
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

    expect(screen.getByText('Version note')).toBeInTheDocument();
    await act(async () => {});
  });

  it('create assistant version shows default system prompt when latest version is not model', async () => {
    mockParams = { assistantId: 'a-1' };
    (GetAssistant as jest.Mock).mockResolvedValueOnce({
      getSuccess: () => true,
      getData: () => ({
        getAssistantprovidermodel: () => null,
        getAssistantprovideragentflow: () => ({}),
      }),
    });

    render(<CreateVersionAssistantPage />);

    expect(getLatestConfigPromptProps().existingPrompt.prompt).toEqual([
      { role: 'system', content: '' },
    ]);
    await act(async () => {});
  });

  it('create assistant version validates using changed provider', async () => {
    mockParams = { assistantId: 'a-1' };
    render(<CreateVersionAssistantPage />);

    const assistantVersionTextProviderProps = getLatestTextProviderProps();
    act(() => {
      assistantVersionTextProviderProps.onChangeProvider('meta');
    });

    const assistantVersionConfigPrompt = getLatestConfigPromptProps();
    act(() => {
      assistantVersionConfigPrompt.onChange({
        prompt: [{ role: 'system', content: 'Edited {{context}}' }],
        variables: [{ name: 'context', type: 'string', defaultvalue: '' }],
      });
    });

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

    expect(mockValidateTextProviderDefaultOptions).toHaveBeenCalled();
    expect(mockValidateTextProviderDefaultOptions.mock.calls.at(-1)?.[0]).toBe('meta');
    await act(async () => {});
  });
});

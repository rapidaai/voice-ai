import React from 'react';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import '@testing-library/jest-dom';

import { ConfigureAssistantDeploymentPage } from '@/app/pages/assistant/actions/create-deployment';
import { CreateTool } from '@/app/pages/assistant/actions/configure-assistant-tool/create-assistant-tool';
import { UpdateTool } from '@/app/pages/assistant/actions/configure-assistant-tool/update-assistant-tool';
import {
  CreateAssistantDebuggerDeployment,
  CreateAssistantTool,
  GetAssistant,
  GetAssistantDebuggerDeployment,
  GetAssistantApiDeployment,
  GetAssistantTool,
  UpdateAssistantTool,
} from '@rapidaai/react';

let mockParams: Record<string, string | undefined> = {
  assistantId: 'assistant-1',
  assistantToolId: 'tool-1',
};
const mockNavigate = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

const mockGlobalNavigation = {
  goBack: jest.fn(),
  goToConfigureWeb: jest.fn(),
  goToEditWeb: jest.fn(),
  goToConfigureApi: jest.fn(),
  goToEditApi: jest.fn(),
  goToConfigureCall: jest.fn(),
  goToEditCall: jest.fn(),
  goToConfigureDebugger: jest.fn(),
  goToEditDebugger: jest.fn(),
  goToConfigureDebuggerExperience: jest.fn(),
  goToConfigureDebuggerSTT: jest.fn(),
  goToConfigureDebuggerTTS: jest.fn(),
  goToConfigureAssistantTool: jest.fn(),
};

jest.mock('@rapidaai/react', () => {
  class ConnectionConfig {
    static WithDebugger(config: unknown) {
      return config;
    }
  }
  class GetAssistantRequest {
    setAssistantdefinition(_: unknown) {}
  }
  class AssistantDefinition {
    setAssistantid(_: string) {}
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
  class GetAssistantDeploymentRequest {
    setAssistantid(_: string) {}
  }
  class CreateAssistantDeploymentRequest {
    setDebugger(_: unknown) {}
    setApi(_: unknown) {}
    setPlugin(_: unknown) {}
    setPhone(_: unknown) {}
  }
  class AssistantDebuggerDeployment {
    setAssistantid(_: string) {}
    setGreeting(_: string) {}
    setGreetinginterruptible(_: boolean) {}
    setMistake(_: string) {}
    setUnclearinputtimeout(_: number) {}
    setUnclearinputmessage(_: string) {}
    setIdealtimeout(_: string) {}
    setIdealtimeoutbackoff(_: string) {}
    setIdealtimeoutmessage(_: string) {}
    setMaxsessionduration(_: string) {}
    setInputaudio(_: unknown) {}
    setOutputaudio(_: unknown) {}
  }
  class AssistantApiDeployment {
    setAssistantid(_: string) {}
    setGreeting(_: string) {}
    setGreetinginterruptible(_: boolean) {}
    setMistake(_: string) {}
    setUnclearinputtimeout(_: number) {}
    setUnclearinputmessage(_: string) {}
    setIdealtimeout(_: string) {}
    setIdealtimeoutbackoff(_: string) {}
    setIdealtimeoutmessage(_: string) {}
    setMaxsessionduration(_: string) {}
    setInputaudio(_: unknown) {}
    setOutputaudio(_: unknown) {}
  }
  class AssistantWebpluginDeployment {
    setAssistantid(_: string) {}
    setGreeting(_: string) {}
    setGreetinginterruptible(_: boolean) {}
    setMistake(_: string) {}
    setUnclearinputtimeout(_: number) {}
    setUnclearinputmessage(_: string) {}
    setIdealtimeout(_: string) {}
    setIdealtimeoutbackoff(_: string) {}
    setIdealtimeoutmessage(_: string) {}
    setMaxsessionduration(_: string) {}
    setInputaudio(_: unknown) {}
    setOutputaudio(_: unknown) {}
    setSuggestionList(_: unknown) {}
    setHelpcenterenabled(_: boolean) {}
    setProductcatalogenabled(_: boolean) {}
    setArticlecatalogenabled(_: boolean) {}
    setUploadfileenabled(_: boolean) {}
    getSuggestionList() {
      return [];
    }
  }
  class AssistantPhoneDeployment {
    setAssistantid(_: string) {}
    setGreeting(_: string) {}
    setGreetinginterruptible(_: boolean) {}
    setMistake(_: string) {}
    setUnclearinputtimeout(_: number) {}
    setUnclearinputmessage(_: string) {}
    setIdealtimeout(_: string) {}
    setIdealtimeoutbackoff(_: string) {}
    setIdealtimeoutmessage(_: string) {}
    setMaxsessionduration(_: string) {}
    setInputaudio(_: unknown) {}
    setOutputaudio(_: unknown) {}
    setPhoneprovidername(_: string) {}
    setPhoneoptionsList(_: unknown[]) {}
  }
  class DeploymentAudioProvider {
    setAudioprovider(_: string) {}
    setAudiooptionsList(_: unknown[]) {}
  }
  return {
    ConnectionConfig,
    Metadata,
    GetAssistantRequest,
    AssistantDefinition,
    AssistantDebuggerDeployment,
    AssistantApiDeployment,
    AssistantWebpluginDeployment,
    AssistantPhoneDeployment,
    DeploymentAudioProvider,
    GetAssistantDeploymentRequest,
    CreateAssistantDeploymentRequest,
    GetAssistant: jest.fn(),
    GetAssistantDebuggerDeployment: jest.fn(),
    GetAssistantApiDeployment: jest.fn(),
    GetAssistantWebpluginDeployment: jest.fn(),
    GetAssistantPhoneDeployment: jest.fn(),
    CreateAssistantDebuggerDeployment: jest.fn(),
    CreateAssistantApiDeployment: jest.fn(),
    CreateAssistantWebpluginDeployment: jest.fn(),
    CreateAssistantPhoneDeployment: jest.fn(),
    GetAssistantTool: jest.fn(),
    CreateAssistantTool: jest.fn(),
    UpdateAssistantTool: jest.fn(),
  };
});

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => mockParams,
  useNavigate: () => mockNavigate,
}));

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

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => mockGlobalNavigation,
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

jest.mock('@/app/components/tools', () => ({
  BuildinTool: ({ onChangeBuildinTool, onChangeConfig, config }: any) => (
    <div>
      <button onClick={() => onChangeBuildinTool('knowledge_retrieval')}>
        Use Knowledge Tool
      </button>
      <button onClick={() => onChangeBuildinTool('endpoint')}>
        Use Endpoint Tool
      </button>
      <button onClick={() => onChangeBuildinTool('end_of_conversation')}>
        Use End Of Conversation Tool
      </button>
      <button onClick={() => onChangeBuildinTool('api_request')}>
        Use API Tool
      </button>
      <button onClick={() => onChangeBuildinTool('mcp')}>Use MCP Tool</button>
      <button
        onClick={() => {
          const { Metadata } = require('@rapidaai/react');
          const serverUrl = new Metadata();
          serverUrl.setKey('mcp.server_url');
          serverUrl.setValue('https://mcp.example.com/sse');
          const toolName = new Metadata();
          toolName.setKey('mcp.tool_name');
          toolName.setValue('calendar_lookup');
          const protocol = new Metadata();
          protocol.setKey('mcp.protocol');
          protocol.setValue('sse');
          const timeout = new Metadata();
          timeout.setKey('mcp.timeout');
          timeout.setValue('45');
          const headers = new Metadata();
          headers.setKey('mcp.headers');
          headers.setValue('{"Authorization":"Bearer token"}');
          onChangeConfig?.({
            ...(config || { code: 'mcp', parameters: [] }),
            code: 'mcp',
            parameters: [serverUrl, toolName, protocol, timeout, headers],
          });
        }}
      >
        Set MCP Config
      </button>
      <button
        onClick={() => {
          const { Metadata } = require('@rapidaai/react');
          const m = new Metadata();
          m.setKey('tool.condition');
          m.setValue(
            JSON.stringify([{ key: 'source', condition: '=', value: 'phone' }]),
          );
          onChangeConfig?.({
            ...(config || { code: 'knowledge_retrieval', parameters: [] }),
            parameters: [m],
          });
        }}
      >
        Set Condition Phone
      </button>
      <button
        onClick={() => {
          const { Metadata } = require('@rapidaai/react');
          const method = new Metadata();
          method.setKey('tool.method');
          method.setValue('POST');
          const endpoint = new Metadata();
          endpoint.setKey('tool.endpoint');
          endpoint.setValue('https://api.example.com/orders');
          const headers = new Metadata();
          headers.setKey('tool.headers');
          headers.setValue('{"Authorization":"Bearer token"}');
          const params = new Metadata();
          params.setKey('tool.parameters');
          params.setValue('{"tool.argument":"order_id"}');
          onChangeConfig?.({
            ...(config || { code: 'api_request', parameters: [] }),
            code: 'api_request',
            parameters: [method, endpoint, headers, params],
          });
        }}
      >
        Set API Request Config
      </button>
      <button
        onClick={() => {
          const { Metadata } = require('@rapidaai/react');
          const endpointId = new Metadata();
          endpointId.setKey('tool.endpoint_id');
          endpointId.setValue('endpoint-123');
          const params = new Metadata();
          params.setKey('tool.parameters');
          params.setValue('{"tool.argument":"customer_id"}');
          onChangeConfig?.({
            ...(config || { code: 'endpoint', parameters: [] }),
            code: 'endpoint',
            parameters: [endpointId, params],
          });
        }}
      >
        Set Endpoint Config
      </button>
      <div data-testid="loaded-condition">
        {(config?.parameters || [])
          .find((p: any) => p.getKey?.() === 'tool.condition')
          ?.getValue?.() || ''}
      </div>
      <div data-testid="loaded-method">
        {(config?.parameters || [])
          .find((p: any) => p.getKey?.() === 'tool.method')
          ?.getValue?.() || ''}
      </div>
      <div data-testid="selected-tool-code">{config?.code || ''}</div>
      <div data-testid="loaded-endpoint-id">
        {(config?.parameters || [])
          .find((p: any) => p.getKey?.() === 'tool.endpoint_id')
          ?.getValue?.() || ''}
      </div>
      <div data-testid="loaded-mcp-server-url">
        {(config?.parameters || [])
          .find((p: any) => p.getKey?.() === 'mcp.server_url')
          ?.getValue?.() || ''}
      </div>
    </div>
  ),
  BuildinToolConfig: {},
  GetDefaultToolConfigIfInvalid: (_code: string, parameters: any[]) =>
    parameters || [],
  GetDefaultToolDefintion: (code: string, defaults: any) => {
    if (code === 'mcp') {
      return {
        name: 'mcp_tool',
        description: 'MCP tool',
        parameters: '{}',
      };
    }
    return defaults;
  },
  ValidateToolDefaultOptions: () => undefined,
}));

jest.mock('@/app/components/tools/common', () => ({
  ToolDefinitionForm: ({ toolDefinition, onChangeToolDefinition }: any) => (
    <div>
      <input
        aria-label="Tool Name"
        value={toolDefinition.name}
        onChange={e =>
          onChangeToolDefinition({ ...toolDefinition, name: e.target.value })
        }
      />
      <textarea
        aria-label="Tool Description"
        value={toolDefinition.description}
        onChange={e =>
          onChangeToolDefinition({
            ...toolDefinition,
            description: e.target.value,
          })
        }
      />
      <textarea
        aria-label="Tool Parameters"
        value={toolDefinition.parameters}
        onChange={e =>
          onChangeToolDefinition({
            ...toolDefinition,
            parameters: e.target.value,
          })
        }
      />
    </div>
  ),
}));

jest.mock('@/app/components/carbon/overflow-menu', () => ({
  OverflowMenu: ({ children }: any) => <div>{children}</div>,
  OverflowMenuItem: ({ itemText, onClick }: any) => (
    <button onClick={onClick}>{itemText}</button>
  ),
}));

jest.mock('@/app/components/helmet', () => ({ Helmet: () => null }));
jest.mock('@/app/components/blocks/page-header-block', () => ({
  PageHeaderBlock: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/blocks/page-title-block', () => ({
  PageTitleBlock: ({ children }: any) => <h2>{children}</h2>,
}));

jest.mock('@/app/components/carbon/button', () => ({
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
  GhostButton: ({
    children,
    isLoading: _,
    renderIcon: _r,
    hasIconOnly: _h,
    iconDescription: _d,
    ...props
  }: any) => <button {...props}>{children}</button>,
  IconOnlyButton: ({
    iconDescription,
    renderIcon: _r,
    tooltipPosition: _tp,
    ...props
  }: any) => <button aria-label={iconDescription} {...props} />,
}));
jest.mock('@/app/components/carbon/modal', () => ({
  Modal: ({ children, open }: any) => (open ? <div>{children}</div> : null),
  ModalHeader: ({ title }: any) => <div>{title}</div>,
  ModalBody: ({ children }: any) => <div>{children}</div>,
  ModalFooter: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/hooks/use-model', () => ({
  useAllProviderCredentials: () => ({ providerCredentials: [] }),
}));

jest.mock('@/app/components/providers/telephony', () => ({
  TelephonyProvider: () => <div>telephony</div>,
  ValidateTelephonyOptions: () => true,
}));

jest.mock('@/app/components/base/corner-border', () => ({
  CornerBorderOverlay: () => null,
}));

jest.mock('@/app/components/providers/speech-to-text/provider', () => ({
  GetDefaultMicrophoneConfig: () => [],
  GetDefaultSpeechToTextIfInvalid: () => [],
  ValidateSpeechToTextIfInvalid: () => undefined,
}));
jest.mock('@/app/components/providers/text-to-speech/provider', () => ({
  GetDefaultSpeakerConfig: () => [],
  GetDefaultTextToSpeechIfInvalid: () => [],
  ValidateTextToSpeechIfInvalid: () => undefined,
}));

jest.mock('@/app/components/base/cards', () => ({
  BaseCard: ({ children }: any) => <div>{children}</div>,
}));

jest.mock(
  '@/app/components/base/modal/assistant-phone-call-deployment-modal',
  () => ({
    AssistantPhoneCallDeploymentDialog: () => null,
  }),
);
jest.mock(
  '@/app/components/base/modal/assistant-debug-deployment-modal',
  () => ({
    AssistantDebugDeploymentDialog: () => null,
  }),
);
jest.mock(
  '@/app/components/base/modal/assistant-web-widget-deployment-modal',
  () => ({
    AssistantWebWidgetlDeploymentDialog: () => null,
  }),
);
jest.mock('@/app/components/base/modal/assistant-api-deployment-modal', () => ({
  AssistantApiDeploymentDialog: () => null,
}));

jest.mock('@/app/components/carbon/empty-state', () => ({
  EmptyState: ({ title, subtitle, action, onAction, actionComponent }: any) => (
    <div>
      <div>{title}</div>
      <div>{subtitle}</div>
      {actionComponent}
      {action ? <button onClick={onAction}>{action}</button> : null}
    </div>
  ),
}));

jest.mock('@/app/components/input-helper', () => ({ InputHelper: () => null }));
jest.mock('@/app/components/form-label', () => ({
  FormLabel: ({ children }: any) => <label>{children}</label>,
}));
jest.mock('@/app/components/form/fieldset', () => ({
  FieldSet: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/carbon/button/copy-button', () => ({
  CopyButton: () => null,
}));

jest.mock('@/utils/date', () => ({
  toHumanReadableDateTime: () => 'date-time',
}));

jest.mock('@/utils', () => ({
  cn: (...args: string[]) => args.filter(Boolean).join(' '),
}));

describe('Deployment and tool flows', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockParams = { assistantId: 'assistant-1', assistantToolId: 'tool-1' };

    (GetAssistant as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({
        getApideployment: () => ({
          getId: () => 'api-dep-1',
          getInputaudio: () => null,
          getOutputaudio: () => null,
          getCreateddate: () => '2026-01-01',
        }),
        hasApideployment: () => true,
        getWebplugindeployment: () => null,
        hasWebplugindeployment: () => false,
        getDebuggerdeployment: () => null,
        hasDebuggerdeployment: () => false,
        getPhonedeployment: () => null,
        hasPhonedeployment: () => false,
      }),
    });

    (GetAssistantTool as jest.Mock).mockImplementation(
      (_cfg, _assistantId, _toolId, cb) => cb(null, { getData: () => null }),
    );

    (GetAssistantDebuggerDeployment as jest.Mock).mockResolvedValue({
      getData: () => ({
        getGreeting: () => 'hello',
        getMistake: () => '',
        getIdealtimeout: () => '30',
        getIdealtimeoutmessage: () => 'Are you there?',
        getMaxsessionduration: () => '300',
        getIdealtimeoutbackoff: () => '2',
        getInputaudio: () => ({
          getAudioprovider: () => 'deepgram',
          getAudiooptionsList: () => [],
        }),
        getOutputaudio: () => ({
          getAudioprovider: () => 'cartesia',
          getAudiooptionsList: () => [],
        }),
      }),
    });
    (CreateAssistantDebuggerDeployment as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({ id: 'dep-1' }),
    });

    (CreateAssistantTool as jest.Mock).mockImplementation(
      (_cfg, _assistantId, _name, _desc, _fields, _method, _opts, cb) =>
        cb(null, {
          getSuccess: () => true,
          getData: () => ({ getName: () => _name }),
        }),
    );

    (UpdateAssistantTool as jest.Mock).mockImplementation(
      (
        _cfg,
        _assistantId,
        _toolId,
        _name,
        _desc,
        _fields,
        _method,
        _opts,
        cb,
      ) =>
        cb(null, {
          getSuccess: () => true,
        }),
    );
  });

  it('create deployment allows channel selection and routes to web deployment', async () => {
    render(<ConfigureAssistantDeploymentPage />);

    fireEvent.click(
      screen.getAllByRole('button', { name: /Add deployment/i })[0],
    );
    fireEvent.click(screen.getByRole('menuitem', { name: /Web Widget/i }));

    expect(mockGlobalNavigation.goToConfigureWeb).toHaveBeenCalledWith(
      'assistant-1',
    );
    await act(async () => {});
  });

  it('create deployment empty state shows add deployment action', async () => {
    (GetAssistant as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({
        getApideployment: () => null,
        hasApideployment: () => false,
        getWebplugindeployment: () => null,
        hasWebplugindeployment: () => false,
        getDebuggerdeployment: () => null,
        hasDebuggerdeployment: () => false,
        getPhonedeployment: () => null,
        hasPhonedeployment: () => false,
      }),
    });

    render(<ConfigureAssistantDeploymentPage />);

    await waitFor(() => {
      expect(screen.getByText('No deployments found')).toBeInTheDocument();
    });

    fireEvent.click(
      screen.getAllByRole('button', { name: /Add deployment/i })[0],
    );
    fireEvent.click(screen.getByRole('menuitem', { name: /Debugger/i }));
    expect(mockGlobalNavigation.goToConfigureDebugger).toHaveBeenCalledWith(
      'assistant-1',
    );
    await act(async () => {});
  });

  it('create deployment routes to API, phone and debugger channels from add deployment menu', async () => {
    render(<ConfigureAssistantDeploymentPage />);

    fireEvent.click(
      screen.getAllByRole('button', { name: /Add deployment/i })[0],
    );
    fireEvent.click(screen.getByRole('menuitem', { name: /SDK \/ API/i }));
    expect(mockGlobalNavigation.goToConfigureApi).toHaveBeenCalledWith(
      'assistant-1',
    );

    fireEvent.click(
      screen.getAllByRole('button', { name: /Add deployment/i })[0],
    );
    fireEvent.click(screen.getByRole('menuitem', { name: /Phone Call/i }));
    expect(mockGlobalNavigation.goToConfigureCall).toHaveBeenCalledWith(
      'assistant-1',
    );

    fireEvent.click(
      screen.getAllByRole('button', { name: /Add deployment/i })[0],
    );
    fireEvent.click(screen.getByRole('menuitem', { name: /Debugger/i }));
    expect(mockGlobalNavigation.goToConfigureDebugger).toHaveBeenCalledWith(
      'assistant-1',
    );
    await act(async () => {});
  });

  it('create deployment shows edit action for existing API deployment', async () => {
    (GetAssistantApiDeployment as jest.Mock).mockResolvedValue({
      getData: () => ({
        getGreeting: () => '',
        getMistake: () => '',
        getIdealtimeout: () => '30',
        getIdealtimeoutmessage: () => '',
        getMaxsessionduration: () => '300',
        getIdealtimeoutbackoff: () => '2',
        getInputaudio: () => null,
        getOutputaudio: () => null,
      }),
    });

    render(<ConfigureAssistantDeploymentPage />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: 'Edit deployment' }),
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Edit deployment' }));
    expect(mockGlobalNavigation.goToEditApi).toHaveBeenCalledWith(
      'assistant-1',
      'api-dep-1',
    );
    await act(async () => {});
  });

  it('debugger edit routes to debugger edit page', async () => {
    (GetAssistant as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({
        getApideployment: () => null,
        hasApideployment: () => false,
        getWebplugindeployment: () => null,
        hasWebplugindeployment: () => false,
        getDebuggerdeployment: () => ({
          getId: () => 'debugger-dep-1',
          getInputaudio: () => ({
            getAudioprovider: () => 'deepgram',
          }),
          getOutputaudio: () => ({
            getAudioprovider: () => 'cartesia',
          }),
          getCreateddate: () => '2026-01-01',
        }),
        hasDebuggerdeployment: () => true,
        getPhonedeployment: () => null,
        hasPhonedeployment: () => false,
      }),
    });

    render(<ConfigureAssistantDeploymentPage />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: 'Edit deployment' }),
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Edit deployment' }));
    expect(mockGlobalNavigation.goToEditDebugger).toHaveBeenCalledWith(
      'assistant-1',
      'debugger-dep-1',
    );
  });

  it('create tool validates missing name before submit', () => {
    render(<CreateTool assistantId="assistant-1" />);

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Configure Tool' }));

    expect(
      screen.getByText('Please provide a valid name for tool.'),
    ).toBeInTheDocument();
  });

  it('update tool validates missing name before submit', () => {
    render(<UpdateTool assistantId="assistant-1" />);

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Update Tool' }));

    expect(
      screen.getByText('Please provide a valid name for tool.'),
    ).toBeInTheDocument();
  });

  it('create tool validates invalid JSON parameters on definition step', () => {
    render(<CreateTool assistantId="assistant-1" />);

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.change(screen.getByLabelText('Tool Name'), {
      target: { value: 'valid_tool' },
    });
    fireEvent.change(screen.getByLabelText('Tool Parameters'), {
      target: { value: '{not-json' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Configure Tool' }));

    expect(
      screen.getByText(
        'Please provide valid parameters, must be a valid JSON.',
      ),
    ).toBeInTheDocument();
  });

  it('update tool validates missing description on definition step', () => {
    (GetAssistantTool as jest.Mock).mockImplementation(
      (_cfg, _assistantId, _toolId, cb) =>
        cb(null, {
          getData: () => ({
            getName: () => 'existing_tool',
            getDescription: () => 'existing description',
            getFields: () => ({ toJavaScript: () => ({ context: 'ok' }) }),
            getExecutionmethod: () => 'knowledge_retrieval',
            getExecutionoptionsList: () => [],
          }),
        }),
    );

    render(<UpdateTool assistantId="assistant-1" />);

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.change(screen.getByLabelText('Tool Description'), {
      target: { value: '' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Update Tool' }));

    expect(
      screen.getByText('Please provide a description for the tool.'),
    ).toBeInTheDocument();
  });

  it('create tool submits directly for MCP from action step', () => {
    render(<CreateTool assistantId="assistant-1" />);

    fireEvent.click(screen.getByRole('button', { name: 'Use MCP Tool' }));
    fireEvent.click(screen.getByRole('button', { name: 'Configure Tool' }));

    expect(CreateAssistantTool).toHaveBeenCalledTimes(1);
    expect((CreateAssistantTool as jest.Mock).mock.calls[0][5]).toBe('mcp');
  });

  it('create tool submits mcp metadata options from action step', () => {
    render(<CreateTool assistantId="assistant-1" />);

    fireEvent.click(screen.getByRole('button', { name: 'Use MCP Tool' }));
    fireEvent.click(screen.getByRole('button', { name: 'Set MCP Config' }));
    fireEvent.click(screen.getByRole('button', { name: 'Configure Tool' }));

    expect(CreateAssistantTool).toHaveBeenCalledTimes(1);
    expect((CreateAssistantTool as jest.Mock).mock.calls[0][2]).toBe(
      'mcp_tool',
    );
    expect((CreateAssistantTool as jest.Mock).mock.calls[0][5]).toBe('mcp');
    const executionOptions = (CreateAssistantTool as jest.Mock).mock
      .calls[0][6];
    const byKey = Object.fromEntries(
      executionOptions.map((m: any) => [m.getKey(), m.getValue()]),
    );
    expect(byKey['mcp.server_url']).toBe('https://mcp.example.com/sse');
    expect(byKey['mcp.tool_name']).toBe('calendar_lookup');
    expect(byKey['mcp.protocol']).toBe('sse');
    expect(byKey['mcp.timeout']).toBe('45');
    expect(byKey['mcp.headers']).toContain('Authorization');
  });

  it('create tool submits end_of_conversation execution method', () => {
    render(<CreateTool assistantId="assistant-1" />);

    fireEvent.click(
      screen.getByRole('button', { name: 'Use End Of Conversation Tool' }),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.change(screen.getByLabelText('Tool Name'), {
      target: { value: 'end_tool' },
    });
    fireEvent.change(screen.getByLabelText('Tool Parameters'), {
      target: { value: '{"reason":"user said goodbye"}' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Configure Tool' }));

    expect(CreateAssistantTool).toHaveBeenCalledTimes(1);
    expect((CreateAssistantTool as jest.Mock).mock.calls[0][5]).toBe(
      'end_of_conversation',
    );
    const executionOptions = (CreateAssistantTool as jest.Mock).mock
      .calls[0][6];
    expect(executionOptions).toEqual([]);
  });

  it('create tool includes tool.condition metadata in execution options', () => {
    render(<CreateTool assistantId="assistant-1" />);

    fireEvent.click(
      screen.getByRole('button', { name: 'Set Condition Phone' }),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.change(screen.getByLabelText('Tool Name'), {
      target: { value: 'valid_tool' },
    });
    fireEvent.change(screen.getByLabelText('Tool Parameters'), {
      target: { value: '{"context":"ok"}' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Configure Tool' }));

    expect(CreateAssistantTool).toHaveBeenCalledTimes(1);
    const executionOptions = (CreateAssistantTool as jest.Mock).mock
      .calls[0][6];
    const condition = executionOptions.find(
      (m: any) => m.getKey() === 'tool.condition',
    );
    expect(condition).toBeTruthy();
    expect(condition.getValue()).toContain('"source"');
    expect(condition.getValue()).toContain('"phone"');
  });

  it('update tool loads existing tool.condition metadata and submits it', async () => {
    (GetAssistantTool as jest.Mock).mockImplementation(
      (_cfg, _assistantId, _toolId, cb) => {
        const { Metadata } = require('@rapidaai/react');
        const m = new Metadata();
        m.setKey('tool.condition');
        m.setValue(
          JSON.stringify([{ key: 'source', condition: '=', value: 'phone' }]),
        );
        cb(null, {
          getData: () => ({
            getName: () => 'existing_tool',
            getDescription: () => 'existing description',
            getFields: () => ({ toJavaScript: () => ({ context: 'ok' }) }),
            getExecutionmethod: () => 'knowledge_retrieval',
            getExecutionoptionsList: () => [m],
          }),
        });
      },
    );

    render(<UpdateTool assistantId="assistant-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('loaded-condition').textContent).toContain(
        '"phone"',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Update Tool' }));

    expect(UpdateAssistantTool).toHaveBeenCalledTimes(1);
    const executionOptions = (UpdateAssistantTool as jest.Mock).mock
      .calls[0][7];
    const condition = executionOptions.find(
      (m: any) => m.getKey() === 'tool.condition',
    );
    expect(condition).toBeTruthy();
    expect(condition.getValue()).toContain('"phone"');
  });

  it('update tool loads mcp options and submits directly from action step', async () => {
    (GetAssistantTool as jest.Mock).mockImplementation(
      (_cfg, _assistantId, _toolId, cb) => {
        const { Metadata } = require('@rapidaai/react');
        const serverUrl = new Metadata();
        serverUrl.setKey('mcp.server_url');
        serverUrl.setValue('https://mcp.example.com/ws');
        const toolName = new Metadata();
        toolName.setKey('mcp.tool_name');
        toolName.setValue('crm_search');
        const protocol = new Metadata();
        protocol.setKey('mcp.protocol');
        protocol.setValue('websocket');
        const timeout = new Metadata();
        timeout.setKey('mcp.timeout');
        timeout.setValue('60');
        const headers = new Metadata();
        headers.setKey('mcp.headers');
        headers.setValue('{"x-api-key":"k1"}');
        cb(null, {
          getData: () => ({
            getName: () => 'existing_mcp_tool',
            getDescription: () => 'existing mcp description',
            getFields: () => ({ toJavaScript: () => ({ type: 'object' }) }),
            getExecutionmethod: () => 'mcp',
            getExecutionoptionsList: () => [
              serverUrl,
              toolName,
              protocol,
              timeout,
              headers,
            ],
          }),
        });
      },
    );

    render(<UpdateTool assistantId="assistant-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('selected-tool-code').textContent).toBe('mcp');
      expect(screen.getByTestId('loaded-mcp-server-url').textContent).toBe(
        'https://mcp.example.com/ws',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Update Tool' }));

    expect(UpdateAssistantTool).toHaveBeenCalledTimes(1);
    expect((UpdateAssistantTool as jest.Mock).mock.calls[0][6]).toBe('mcp');
    const executionOptions = (UpdateAssistantTool as jest.Mock).mock
      .calls[0][7];
    const byKey = Object.fromEntries(
      executionOptions.map((m: any) => [m.getKey(), m.getValue()]),
    );
    expect(byKey['mcp.server_url']).toBe('https://mcp.example.com/ws');
    expect(byKey['mcp.tool_name']).toBe('crm_search');
    expect(byKey['mcp.protocol']).toBe('websocket');
    expect(byKey['mcp.timeout']).toBe('60');
    expect(byKey['mcp.headers']).toContain('x-api-key');
  });

  it('update tool loads end_of_conversation and submits update', async () => {
    (GetAssistantTool as jest.Mock).mockImplementation(
      (_cfg, _assistantId, _toolId, cb) =>
        cb(null, {
          getData: () => ({
            getName: () => 'existing_end_tool',
            getDescription: () => 'existing end description',
            getFields: () => ({
              toJavaScript: () => ({ reason: 'conversation completed' }),
            }),
            getExecutionmethod: () => 'end_of_conversation',
            getExecutionoptionsList: () => [],
          }),
        }),
    );

    render(<UpdateTool assistantId="assistant-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('selected-tool-code').textContent).toBe(
        'end_of_conversation',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Update Tool' }));

    expect(UpdateAssistantTool).toHaveBeenCalledTimes(1);
    expect((UpdateAssistantTool as jest.Mock).mock.calls[0][5]).toEqual({
      reason: 'conversation completed',
    });
    expect((UpdateAssistantTool as jest.Mock).mock.calls[0][6]).toBe(
      'end_of_conversation',
    );
    const executionOptions = (UpdateAssistantTool as jest.Mock).mock
      .calls[0][7];
    expect(executionOptions).toEqual([]);
  });

  it('create tool submits api_request execution method with api metadata options', () => {
    render(<CreateTool assistantId="assistant-1" />);

    fireEvent.click(screen.getByRole('button', { name: 'Use API Tool' }));
    fireEvent.click(
      screen.getByRole('button', { name: 'Set API Request Config' }),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.change(screen.getByLabelText('Tool Name'), {
      target: { value: 'api_tool' },
    });
    fireEvent.change(screen.getByLabelText('Tool Parameters'), {
      target: { value: '{"context":"ok"}' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Configure Tool' }));

    expect(CreateAssistantTool).toHaveBeenCalledTimes(1);
    expect((CreateAssistantTool as jest.Mock).mock.calls[0][5]).toBe(
      'api_request',
    );
    const executionOptions = (CreateAssistantTool as jest.Mock).mock
      .calls[0][6];
    const byKey = Object.fromEntries(
      executionOptions.map((m: any) => [m.getKey(), m.getValue()]),
    );
    expect(byKey['tool.method']).toBe('POST');
    expect(byKey['tool.endpoint']).toBe('https://api.example.com/orders');
    expect(byKey['tool.headers']).toContain('Authorization');
    expect(byKey['tool.parameters']).toContain('tool.argument');
  });

  it('update tool loads api_request options and preserves them on submit', async () => {
    (GetAssistantTool as jest.Mock).mockImplementation(
      (_cfg, _assistantId, _toolId, cb) => {
        const { Metadata } = require('@rapidaai/react');
        const method = new Metadata();
        method.setKey('tool.method');
        method.setValue('PATCH');
        const endpoint = new Metadata();
        endpoint.setKey('tool.endpoint');
        endpoint.setValue('https://api.example.com/orders/1');
        const headers = new Metadata();
        headers.setKey('tool.headers');
        headers.setValue('{"Authorization":"Bearer token"}');
        const params = new Metadata();
        params.setKey('tool.parameters');
        params.setValue('{"tool.argument":"order_id"}');
        cb(null, {
          getData: () => ({
            getName: () => 'existing_api_tool',
            getDescription: () => 'existing api description',
            getFields: () => ({ toJavaScript: () => ({ context: 'ok' }) }),
            getExecutionmethod: () => 'api_request',
            getExecutionoptionsList: () => [method, endpoint, headers, params],
          }),
        });
      },
    );

    render(<UpdateTool assistantId="assistant-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('selected-tool-code').textContent).toBe(
        'api_request',
      );
      expect(screen.getByTestId('loaded-method').textContent).toBe('PATCH');
    });

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Update Tool' }));

    expect(UpdateAssistantTool).toHaveBeenCalledTimes(1);
    expect((UpdateAssistantTool as jest.Mock).mock.calls[0][5]).toEqual({
      context: 'ok',
    });
    expect((UpdateAssistantTool as jest.Mock).mock.calls[0][6]).toBe(
      'api_request',
    );
    const executionOptions = (UpdateAssistantTool as jest.Mock).mock
      .calls[0][7];
    const byKey = Object.fromEntries(
      executionOptions.map((m: any) => [m.getKey(), m.getValue()]),
    );
    expect(byKey['tool.method']).toBe('PATCH');
    expect(byKey['tool.endpoint']).toBe('https://api.example.com/orders/1');
    expect(byKey['tool.headers']).toContain('Authorization');
    expect(byKey['tool.parameters']).toContain('tool.argument');
  });

  it('create tool submits endpoint execution method with endpoint metadata options', () => {
    render(<CreateTool assistantId="assistant-1" />);

    fireEvent.click(screen.getByRole('button', { name: 'Use Endpoint Tool' }));
    fireEvent.click(
      screen.getByRole('button', { name: 'Set Endpoint Config' }),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.change(screen.getByLabelText('Tool Name'), {
      target: { value: 'endpoint_tool' },
    });
    fireEvent.change(screen.getByLabelText('Tool Parameters'), {
      target: { value: '{"context":"ok"}' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Configure Tool' }));

    expect(CreateAssistantTool).toHaveBeenCalledTimes(1);
    expect((CreateAssistantTool as jest.Mock).mock.calls[0][5]).toBe(
      'endpoint',
    );
    const executionOptions = (CreateAssistantTool as jest.Mock).mock
      .calls[0][6];
    const byKey = Object.fromEntries(
      executionOptions.map((m: any) => [m.getKey(), m.getValue()]),
    );
    expect(byKey['tool.endpoint_id']).toBe('endpoint-123');
    expect(byKey['tool.parameters']).toContain('customer_id');
  });

  it('update tool loads endpoint options and preserves them on submit', async () => {
    (GetAssistantTool as jest.Mock).mockImplementation(
      (_cfg, _assistantId, _toolId, cb) => {
        const { Metadata } = require('@rapidaai/react');
        const endpointId = new Metadata();
        endpointId.setKey('tool.endpoint_id');
        endpointId.setValue('endpoint-777');
        const params = new Metadata();
        params.setKey('tool.parameters');
        params.setValue('{"tool.argument":"account_id"}');
        cb(null, {
          getData: () => ({
            getName: () => 'existing_endpoint_tool',
            getDescription: () => 'existing endpoint description',
            getFields: () => ({ toJavaScript: () => ({ context: 'ok' }) }),
            getExecutionmethod: () => 'endpoint',
            getExecutionoptionsList: () => [endpointId, params],
          }),
        });
      },
    );

    render(<UpdateTool assistantId="assistant-1" />);

    await waitFor(() => {
      expect(screen.getByTestId('selected-tool-code').textContent).toBe(
        'endpoint',
      );
      expect(screen.getByTestId('loaded-endpoint-id').textContent).toBe(
        'endpoint-777',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Update Tool' }));

    expect(UpdateAssistantTool).toHaveBeenCalledTimes(1);
    expect((UpdateAssistantTool as jest.Mock).mock.calls[0][5]).toEqual({
      context: 'ok',
    });
    expect((UpdateAssistantTool as jest.Mock).mock.calls[0][6]).toBe(
      'endpoint',
    );
    const executionOptions = (UpdateAssistantTool as jest.Mock).mock
      .calls[0][7];
    const byKey = Object.fromEntries(
      executionOptions.map((m: any) => [m.getKey(), m.getValue()]),
    );
    expect(byKey['tool.endpoint_id']).toBe('endpoint-777');
    expect(byKey['tool.parameters']).toContain('account_id');
  });
});

import React from 'react';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import '@testing-library/jest-dom';
import { ConfigureAssistantWebDeploymentPage } from '@/app/pages/assistant/actions/create-deployment/web-plugin';
import {
  CreateAssistantWebpluginDeployment,
  GetAssistantWebpluginDeployment,
} from '@rapidaai/react';

let mockParams: Record<string, string | undefined> = {
  assistantId: 'assistant-1',
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

  class DeploymentAudioProvider {
    private audioProvider = '';
    private audioOptions: Metadata[] = [];
    setAudioprovider(v: string) {
      this.audioProvider = v;
    }
    getAudioprovider() {
      return this.audioProvider;
    }
    setAudiooptionsList(v: Metadata[]) {
      this.audioOptions = v;
    }
    getAudiooptionsList() {
      return this.audioOptions;
    }
  }

  class AssistantWebpluginDeployment {
    private inputAudio?: DeploymentAudioProvider;
    private outputAudio?: DeploymentAudioProvider;
    setAssistantid(_: string) {}
    setGreeting(_: string) {}
    setGreetinginterruptible(_: boolean) {}
    setMistake(_: string) {}
    setUnclearinputtimeout(_: number) {}
    hasUnclearinputtimeout() {
      return false;
    }
    getUnclearinputtimeout() {
      return 0;
    }
    setUnclearinputmessage(_: string) {}
    hasUnclearinputmessage() {
      return false;
    }
    getUnclearinputmessage() {
      return '';
    }
    setIdealtimeout(_: string) {}
    setIdealtimeoutbackoff(_: string) {}
    setIdealtimeoutmessage(_: string) {}
    setMaxsessionduration(_: string) {}
    setSuggestionList(_: string[]) {}
    setHelpcenterenabled(_: boolean) {}
    setProductcatalogenabled(_: boolean) {}
    setArticlecatalogenabled(_: boolean) {}
    setUploadfileenabled(_: boolean) {}
    setInputaudio(v: DeploymentAudioProvider) {
      this.inputAudio = v;
    }
    getInputaudio() {
      return this.inputAudio;
    }
    setOutputaudio(v: DeploymentAudioProvider) {
      this.outputAudio = v;
    }
    getOutputaudio() {
      return this.outputAudio;
    }
  }

  class CreateAssistantDeploymentRequest {
    private plugin?: AssistantWebpluginDeployment;
    setPlugin(v: AssistantWebpluginDeployment) {
      this.plugin = v;
    }
    getPlugin() {
      return this.plugin;
    }
  }

  class GetAssistantDeploymentRequest {
    setAssistantid(_: string) {}
  }

  return {
    ConnectionConfig,
    Metadata,
    DeploymentAudioProvider,
    AssistantWebpluginDeployment,
    CreateAssistantDeploymentRequest,
    GetAssistantDeploymentRequest,
    GetAssistantWebpluginDeployment: jest.fn(),
    CreateAssistantWebpluginDeployment: jest.fn(),
  };
});

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => mockParams,
}));

jest.mock('@/hooks', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: jest.fn(),
    hideLoader: jest.fn(),
  }),
}));

jest.mock('@/hooks/use-model', () => ({
  useAllProviderCredentials: () => ({ providerCredentials: [] }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'u-1',
    projectId: 'p-1',
    token: 't-1',
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToDeploymentAssistant: jest.fn(),
  }),
}));

jest.mock('@/app/components/helmet', () => ({ Helmet: () => null }));
jest.mock('@/app/components/base/modal/assistant-instruction-modal', () => ({
  AssistantWebwidgetDeploymentDialog: () => null,
}));
jest.mock('@/app/components/base/cards', () => ({
  BaseCard: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/carbon/form/input-checkbox', () => ({
  InputCheckbox: ({ children, ...props }: any) => (
    <label>
      <input type="checkbox" {...props} />
      {children}
    </label>
  ),
}));
jest.mock('@/app/components/input-helper', () => ({
  InputHelper: ({ children }: any) => <div>{children}</div>,
}));
jest.mock('@/app/components/form/switch', () => ({
  SwitchWithLabel: ({ enable, setEnable, label }: any) => (
    <button type="button" onClick={() => setEnable(!enable)}>
      {label}
    </button>
  ),
}));

jest.mock('@/app/components/form/tab-form', () => ({
  TabForm: ({ activeTab, form, errorMessage }: any) => {
    const active = form.find((x: any) => x.code === activeTab);
    return (
      <div>
        {errorMessage ? <div>{errorMessage}</div> : null}
        <div>{active?.body}</div>
        <div>
          {active?.actions?.map((action: React.ReactElement, idx: number) => (
            <div key={idx}>{action}</div>
          ))}
        </div>
      </div>
    );
  },
}));

jest.mock(
  '@/app/pages/assistant/actions/create-deployment/web-plugin/configure-experience',
  () => {
    const React = require('react');
    return {
      ConfigureExperience: ({ setExperienceConfig }: any) => {
        React.useEffect(() => {
          setExperienceConfig((prev: any) => ({
            ...prev,
            greeting: prev.greeting || 'hello',
          }));
        }, [setExperienceConfig]);
        return <div>experience</div>;
      },
    };
  },
);

jest.mock(
  '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-input',
  () => ({
    ConfigureAudioInputProvider: () => <div>audio-input</div>,
  }),
);

jest.mock(
  '@/app/pages/assistant/actions/create-deployment/commons/configure-audio-output',
  () => ({
    ConfigureAudioOutputProvider: () => <div>audio-output</div>,
  }),
);

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

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => {
  const React = require('react');
  return {
    useConfirmDialog: ({
      title = 'Are you sure?',
    }: { title?: string } = {}) => {
      const [isOpen, setIsOpen] = React.useState(false);
      const [onConfirm, setOnConfirm] = React.useState<() => void>(
        () => () => {},
      );

      return {
        showDialog: (cb: () => void) => {
          setOnConfirm(() => cb);
          setIsOpen(true);
        },
        ConfirmDialogComponent: () =>
          isOpen ? (
            <button
              onClick={() => {
                onConfirm();
                setIsOpen(false);
              }}
            >
              {title}
            </button>
          ) : null,
      };
    },
  };
});

jest.mock('@/app/components/carbon/button', () => ({
  PrimaryButton: ({ children, isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  GhostButton: ({ children, isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SecondaryButton: ({ children, isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

describe('Web plugin deployment voice input intent actions', () => {
  const mockEditDeployment = () => ({
    getData: () => ({
      getId: () => 'dep-1',
      getGreeting: () => 'hello',
      hasGreetinginterruptible: () => true,
      getGreetinginterruptible: () => true,
      getSuggestionList: () => [],
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

  beforeEach(() => {
    jest.clearAllMocks();

    (GetAssistantWebpluginDeployment as jest.Mock).mockResolvedValue(
      mockEditDeployment(),
    );

    (CreateAssistantWebpluginDeployment as jest.Mock).mockResolvedValue({
      getData: () => ({ id: 'dep-1' }),
      getSuccess: () => true,
    });
  });

  it('Continue keeps existing voice input enabled', async () => {
    render(<ConfigureAssistantWebDeploymentPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    await waitFor(() =>
      expect(
        screen.getByText(/receive user input via audio and text/i),
      ).toBeInTheDocument(),
    );

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Deploy Web Widget' }));

    await waitFor(() =>
      expect(CreateAssistantWebpluginDeployment).toHaveBeenCalledTimes(1),
    );

    const req = (CreateAssistantWebpluginDeployment as jest.Mock).mock
      .calls[0][1];
    const deployment = req.getPlugin();
    expect(deployment.getInputaudio()).toBeDefined();
    await act(async () => {});
  });

  it('unchecking voice input removes input audio on save', async () => {
    render(<ConfigureAssistantWebDeploymentPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    await waitFor(() =>
      expect(
        screen.getByText(/receive user input via audio and text/i),
      ).toBeInTheDocument(),
    );
    fireEvent.click(
      screen.getByLabelText(/Enable Voice Input \(Speech-to-Text\)/i),
    );
    expect(
      screen.getByText(/receive user input via text only/i),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Deploy Web Widget' }));

    await waitFor(() =>
      expect(CreateAssistantWebpluginDeployment).toHaveBeenCalledTimes(1),
    );

    const req = (CreateAssistantWebpluginDeployment as jest.Mock).mock
      .calls[0][1];
    const deployment = req.getPlugin();
    expect(deployment.getInputaudio()).toBeUndefined();
    await act(async () => {});
  });

  it('unchecking voice output removes output audio on save', async () => {
    render(<ConfigureAssistantWebDeploymentPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    await waitFor(() =>
      expect(
        screen.getByText(/delivered via audio and text/i),
      ).toBeInTheDocument(),
    );
    fireEvent.click(
      screen.getByLabelText(/Enable Voice Output \(Text-to-Speech\)/i),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Deploy Web Widget' }));

    await waitFor(() =>
      expect(CreateAssistantWebpluginDeployment).toHaveBeenCalledTimes(1),
    );

    const req = (CreateAssistantWebpluginDeployment as jest.Mock).mock
      .calls[0][1];
    const deployment = req.getPlugin();
    expect(deployment.getOutputaudio()).toBeUndefined();
    await act(async () => {});
  });

  it('create mode deploys without existing deployment data', async () => {
    (GetAssistantWebpluginDeployment as jest.Mock).mockResolvedValue({
      getData: () => null,
    });

    render(<ConfigureAssistantWebDeploymentPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Deploy Web Widget' }));

    await waitFor(() =>
      expect(CreateAssistantWebpluginDeployment).toHaveBeenCalledTimes(1),
    );

    const req = (CreateAssistantWebpluginDeployment as jest.Mock).mock
      .calls[0][1];
    const deployment = req.getPlugin();
    expect(deployment.getInputaudio()).toBeUndefined();
    expect(deployment.getOutputaudio()).toBeDefined();
    await act(async () => {});
  });
});

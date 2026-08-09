import React from 'react';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import '@testing-library/jest-dom';
import { ConfigureAssistantCallDeploymentPage } from '@/app/pages/assistant/actions/create-deployment/phone';
import { EditAssistantCallDeploymentPage } from '@/app/pages/assistant/actions/create-deployment/phone/edit';
import {
  CreateAssistantPhoneDeployment,
  GetAssistantPhoneDeployment,
} from '@rapidaai/react';

let mockParams: Record<string, string | undefined> = {
  assistantId: 'assistant-1',
};

const mockValidateTelephonyOptions = jest.fn();
const mockValidateSpeechToTextIfInvalid = jest.fn();
const mockValidateTextToSpeechIfInvalid = jest.fn();

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

  class AssistantPhoneDeployment {
    private inputAudio?: DeploymentAudioProvider;
    private outputAudio?: DeploymentAudioProvider;
    private phoneProvider = '';
    private phoneOptions: Metadata[] = [];
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
    setPhoneprovidername(v: string) {
      this.phoneProvider = v;
    }
    getPhoneprovidername() {
      return this.phoneProvider;
    }
    setPhoneoptionsList(v: Metadata[]) {
      this.phoneOptions = v;
    }
    getPhoneoptionsList() {
      return this.phoneOptions;
    }
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
    private phone?: AssistantPhoneDeployment;
    setPhone(v: AssistantPhoneDeployment) {
      this.phone = v;
    }
    getPhone() {
      return this.phone;
    }
  }

  class GetAssistantDeploymentRequest {
    setAssistantid(_: string) {}
  }

  return {
    ConnectionConfig,
    Metadata,
    DeploymentAudioProvider,
    AssistantPhoneDeployment,
    CreateAssistantDeploymentRequest,
    GetAssistantDeploymentRequest,
    GetAssistantPhoneDeployment: jest.fn(),
    CreateAssistantPhoneDeployment: jest.fn(),
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

jest.mock('@/app/components/form/tab-form', () => ({
  TabForm: ({ activeTab, form, errorMessage, onChangeActiveTab }: any) => {
    const active = form.find((x: any) => x.code === activeTab);
    return (
      <div>
        <div>
          {form.map((item: any) => (
            <button
              key={item.code}
              type="button"
              onClick={() => onChangeActiveTab(item.code)}
            >
              {item.name}
            </button>
          ))}
        </div>
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

jest.mock('@/app/components/carbon/tabs', () => ({
  Tabs: ({ tabs = [], children, selectedIndex = 0, onChange }: any) => {
    const panels = Array.isArray(children) ? children : [children];
    return (
      <div>
        <div>
          {tabs.map((tab: string, index: number) => (
            <button key={tab} type="button" onClick={() => onChange?.(index)}>
              {tab}
            </button>
          ))}
        </div>
        <div>{panels[selectedIndex]}</div>
      </div>
    );
  },
}));

jest.mock(
  '@/app/pages/assistant/actions/create-deployment/commons/configure-experience',
  () => ({
    ConfigureExperience: () => <div>experience</div>,
  }),
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

jest.mock('@/app/components/providers/telephony', () => ({
  TelephonyProvider: () => <div>telephony</div>,
  GetDefaultTelephonyConfigIfInvalid: (_provider: string, parameters: any[]) =>
    parameters || [],
  ValidateTelephonyOptions: (...args: any[]) =>
    mockValidateTelephonyOptions(...args),
}));

jest.mock('@/app/components/providers/speech-to-text/provider', () => ({
  GetDefaultMicrophoneConfig: () => [],
  GetDefaultSpeechToTextIfInvalid: () => [],
  ValidateSpeechToTextIfInvalid: (...args: any[]) =>
    mockValidateSpeechToTextIfInvalid(...args),
}));
jest.mock('@/app/components/providers/text-to-speech/provider', () => ({
  GetDefaultSpeakerConfig: () => [],
  GetDefaultTextToSpeechIfInvalid: () => [],
  ValidateTextToSpeechIfInvalid: (...args: any[]) =>
    mockValidateTextToSpeechIfInvalid(...args),
}));

jest.mock('@/app/components/carbon/button', () => ({
  PrimaryButton: ({ children, isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SecondaryButton: ({ children, isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  GhostButton: ({ children, isLoading, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

describe('Phone deployment create and edit flows', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    window.history.replaceState(null, '', '/');
    mockValidateTelephonyOptions.mockReturnValue(true);
    mockValidateSpeechToTextIfInvalid.mockReturnValue(undefined);
    mockValidateTextToSpeechIfInvalid.mockReturnValue(undefined);

    (CreateAssistantPhoneDeployment as jest.Mock).mockResolvedValue({
      getData: () => ({ id: 'dep-1' }),
      getSuccess: () => true,
    });
  });

  it('edit mode preserves telephony + audio payload when deploying', async () => {
    (GetAssistantPhoneDeployment as jest.Mock).mockResolvedValue({
      getData: () => ({
        getGreeting: () => 'hello',
        hasGreetinginterruptible: () => true,
        getGreetinginterruptible: () => true,
        getMistake: () => '',
        getIdealtimeout: () => '30',
        getIdealtimeoutmessage: () => 'Are you there?',
        getMaxsessionduration: () => '300',
        getIdealtimeoutbackoff: () => '2',
        getPhoneprovidername: () => 'twilio',
        getPhoneoptionsList: () => [],
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

    render(<ConfigureAssistantCallDeploymentPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Deploy Phone' }));

    await waitFor(() =>
      expect(CreateAssistantPhoneDeployment).toHaveBeenCalledTimes(1),
    );

    const req = (CreateAssistantPhoneDeployment as jest.Mock).mock.calls[0][1];
    const deployment = req.getPhone();
    expect(deployment.getPhoneprovidername()).toBe('twilio');
    expect(deployment.getInputaudio()).toBeDefined();
    expect(deployment.getOutputaudio()).toBeDefined();
    await act(async () => {});
  });

  it('create mode deploys with default telephony + audio payload', async () => {
    (GetAssistantPhoneDeployment as jest.Mock).mockResolvedValue({
      getData: () => null,
    });

    render(<ConfigureAssistantCallDeploymentPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Deploy Phone' }));

    await waitFor(() =>
      expect(CreateAssistantPhoneDeployment).toHaveBeenCalledTimes(1),
    );

    const req = (CreateAssistantPhoneDeployment as jest.Mock).mock.calls[0][1];
    const deployment = req.getPhone();
    expect(deployment.getPhoneprovidername()).toBe('twilio');
    expect(deployment.getInputaudio()).toBeDefined();
    expect(deployment.getOutputaudio()).toBeDefined();
    await act(async () => {});
  });

  it('blocks moving forward when telephony configuration is invalid', async () => {
    mockValidateTelephonyOptions.mockReturnValue(false);
    (GetAssistantPhoneDeployment as jest.Mock).mockResolvedValue({
      getData: () => null,
    });

    render(<ConfigureAssistantCallDeploymentPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    expect(
      screen.getByText('Please provide a valid telephony configuration.'),
    ).toBeInTheDocument();
    expect(mockValidateTelephonyOptions).toHaveBeenCalledTimes(1);
  });

  it('blocks voice-input step when STT provider options are invalid', async () => {
    mockValidateSpeechToTextIfInvalid.mockReturnValue(
      'Please configure STT provider credentials.',
    );
    (GetAssistantPhoneDeployment as jest.Mock).mockResolvedValue({
      getData: () => null,
    });

    render(<ConfigureAssistantCallDeploymentPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    expect(
      screen.getByText('Please configure STT provider credentials.'),
    ).toBeInTheDocument();
    expect(mockValidateSpeechToTextIfInvalid).toHaveBeenCalled();
  });

  it('blocks deploy when TTS configuration is invalid', async () => {
    mockValidateTextToSpeechIfInvalid.mockReturnValue(
      'Please configure TTS provider credentials.',
    );
    (GetAssistantPhoneDeployment as jest.Mock).mockResolvedValue({
      getData: () => null,
    });

    render(<ConfigureAssistantCallDeploymentPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.click(screen.getByRole('button', { name: 'Deploy Phone' }));

    expect(
      screen.getByText('Please configure TTS provider credentials.'),
    ).toBeInTheDocument();
    expect(CreateAssistantPhoneDeployment).not.toHaveBeenCalled();
    expect(mockValidateTextToSpeechIfInvalid).toHaveBeenCalled();
  });

  it('allows switching tabs without step-by-step progression', async () => {
    (GetAssistantPhoneDeployment as jest.Mock).mockResolvedValue({
      getData: () => null,
    });

    render(<EditAssistantCallDeploymentPage />);

    expect(screen.getByText('telephony')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Voice Output' }));
    expect(screen.getByText('audio-output')).toBeInTheDocument();
  });

  it('opens edit page with telephony tab selected by default', async () => {
    (GetAssistantPhoneDeployment as jest.Mock).mockResolvedValue({
      getData: () => null,
    });

    render(<EditAssistantCallDeploymentPage />);
    expect(screen.getByText('telephony')).toBeInTheDocument();
  });
});

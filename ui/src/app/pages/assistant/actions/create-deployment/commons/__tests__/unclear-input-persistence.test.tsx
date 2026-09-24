import React from 'react';
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
} from '@testing-library/react';
import '@testing-library/jest-dom';
import {
  AssistantApiDeployment,
  AssistantDebuggerDeployment,
  AssistantPhoneDeployment,
  AssistantWebpluginDeployment,
  CreateAssistantApiDeployment,
  CreateAssistantDebuggerDeployment,
  CreateAssistantPhoneDeployment,
  CreateAssistantWebpluginDeployment,
  GetAssistantApiDeployment,
  GetAssistantDebuggerDeployment,
  GetAssistantPhoneDeployment,
  GetAssistantWebpluginDeployment,
} from '@rapidaai/react';
import {
  ConfigureExperience,
  DEFAULT_UNCLEAR_INPUT_MESSAGE,
  DEFAULT_UNCLEAR_INPUT_TIMEOUT,
} from '../configure-experience';
import { ConfigureExperience as ConfigureWebExperience } from '../../web-plugin/configure-experience';
import { ConfigureAssistantCallDeploymentPage } from '../../phone';
import { EditAssistantCallDeploymentPage } from '../../phone/edit';
import { ConfigureAssistantWebDeploymentPage } from '../../web-plugin';
import { EditAssistantWebDeploymentPage } from '../../web-plugin/edit';
import { ConfigureAssistantDebuggerDeploymentPage } from '../../debugger';
import { EditAssistantDebuggerDeploymentPage } from '../../debugger/edit';
import { ConfigureAssistantApiDeploymentPage } from '../../api';
import { EditAssistantApiDeploymentPage } from '../../api/edit';
import { useDeploymentSectionEdit } from '../../hooks/use-deployment-section-edit';

let mockSearchParams = new URLSearchParams();

jest.mock('../configure-experience', () => {
  const actual = jest.requireActual('../configure-experience');
  return {
    ...actual,
    ConfigureExperience: jest.fn(props => (
      <actual.ConfigureExperience {...props} />
    )),
  };
});
jest.mock('@rapidaai/react', () => ({
  ...jest.requireActual('@rapidaai/react'),
  GetAssistantApiDeployment: jest.fn(),
  CreateAssistantApiDeployment: jest.fn(),
  GetAssistantPhoneDeployment: jest.fn(),
  GetAssistantWebpluginDeployment: jest.fn(),
  GetAssistantDebuggerDeployment: jest.fn(),
  CreateAssistantPhoneDeployment: jest.fn(),
  CreateAssistantWebpluginDeployment: jest.fn(),
  CreateAssistantDebuggerDeployment: jest.fn(),
}));
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ assistantId: '123' }),
  useSearchParams: () => [mockSearchParams],
}));
jest.mock('@/configs', () => ({ connectionConfig: {} }));
jest.mock('@/hooks', () => ({
  useRapidaStore: () => ({ showLoader: jest.fn(), hideLoader: jest.fn() }),
}));
jest.mock('@/hooks/use-model', () => ({
  useAllProviderCredentials: () => ({ providerCredentials: [] }),
}));
jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({ authId: 'u', projectId: 'p', token: 't' }),
}));
jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({ goToDeploymentAssistant: jest.fn() }),
}));
jest.mock('@/theme/documentation-url', () => ({
  useDocumentationUrl: () => '/experience',
}));
jest.mock('@/app/components/helmet', () => ({ Helmet: () => null }));
jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    showDialog: jest.fn(),
    ConfirmDialogComponent: () => null,
  }),
}));
jest.mock('@/app/components/base/modal/assistant-instruction-modal', () => ({
  AssistantWebwidgetDeploymentDialog: () => null,
}));
jest.mock(
  '@/app/components/base/modal/debugger-deployment-success-modal',
  () => ({
    DebuggerDeploymentSuccessDialog: () => null,
  }),
);
jest.mock('../configure-audio-input', () => ({
  ConfigureAudioInputProvider: () => null,
}));
jest.mock('../configure-audio-output', () => ({
  ConfigureAudioOutputProvider: () => null,
}));
jest.mock('@/app/components/providers/telephony', () => ({
  TelephonyProvider: () => null,
  GetDefaultTelephonyConfigIfInvalid: (_: string, parameters: any[]) =>
    parameters || [],
  ValidateTelephonyOptions: () => true,
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
jest.mock('@/app/components/form/tab-form', () => ({
  TabForm: ({ activeTab, form, errorMessage }: any) => {
    const active = form.find((item: any) => item.code === activeTab);
    return (
      <>
        {errorMessage && <div role="alert">{errorMessage}</div>}
        {active.body}
        {active.actions.map((action: React.ReactNode, index: number) => (
          <div key={index}>{action}</div>
        ))}
      </>
    );
  },
}));
jest.mock('@/app/components/carbon/tabs', () => ({
  Tabs: ({ tabs, onChange }: any) => (
    <>
      {tabs.map((label: string, index: number) => (
        <button key={label} onClick={() => onChange(index)}>
          {label}
        </button>
      ))}
    </>
  ),
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
jest.mock('@/app/components/carbon/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextArea: ({ labelText, value, onChange }: any) => (
    <textarea aria-label={labelText} value={value} onChange={onChange} />
  ),
  TextInput: ({ labelText, value, onChange }: any) => (
    <input aria-label={labelText} value={value} onChange={onChange} />
  ),
}));
jest.mock(
  '@/app/components/configuration/config-var/config-select',
  () => () => null,
);
jest.mock('@carbon/react', () => ({
  Button: ({ children }: any) => <button>{children}</button>,
  ButtonSet: ({ children }: any) => <div>{children}</div>,
  CheckboxGroup: ({ children }: any) => <div>{children}</div>,
  ComboBox: ({ 'aria-label': label, selectedItem, onChange }: any) => (
    <input
      aria-label={label}
      value={selectedItem || ''}
      onChange={e =>
        onChange({ selectedItem: null, inputValue: e.target.value })
      }
    />
  ),
  TextInput: ({ labelText, value, onChange }: any) => (
    <input aria-label={labelText} value={value} onChange={onChange} />
  ),
  FormLabel: ({ children }: any) => <label>{children}</label>,
  Slider: ({ labelText, value, onChange }: any) => (
    <input
      aria-label={labelText}
      type="number"
      value={Number.isNaN(value) ? '' : value}
      onChange={e => onChange({ value: Number(e.target.value) })}
    />
  ),
  Toggletip: ({ children }: any) => <div>{children}</div>,
  ToggletipActions: ({ children }: any) => <div>{children}</div>,
  ToggletipButton: () => null,
  ToggletipContent: () => null,
  Toggle: () => null,
}));
jest.mock('@/app/components/carbon/form/input-checkbox', () => ({
  InputCheckbox: () => null,
}));
jest.mock('@/app/components/carbon/notification', () => ({
  Notification: ({ subtitle }: any) => <div role="alert">{subtitle}</div>,
}));

const deployments = [
  {
    type: 'phone' as const,
    Deployment: AssistantPhoneDeployment,
    fetch: GetAssistantPhoneDeployment as jest.Mock,
    save: CreateAssistantPhoneDeployment as jest.Mock,
    field: 'getPhone' as const,
    Create: ConfigureAssistantCallDeploymentPage,
    Edit: EditAssistantCallDeploymentPage,
    Experience: ConfigureExperience,
    deployLabel: 'Deploy Phone',
  },
  {
    type: 'web' as const,
    Deployment: AssistantWebpluginDeployment,
    fetch: GetAssistantWebpluginDeployment as jest.Mock,
    save: CreateAssistantWebpluginDeployment as jest.Mock,
    field: 'getPlugin' as const,
    Create: ConfigureAssistantWebDeploymentPage,
    Edit: EditAssistantWebDeploymentPage,
    Experience: ConfigureWebExperience,
    deployLabel: 'Deploy Web Widget',
  },
  {
    type: 'debugger' as const,
    Deployment: AssistantDebuggerDeployment,
    fetch: GetAssistantDebuggerDeployment as jest.Mock,
    save: CreateAssistantDebuggerDeployment as jest.Mock,
    field: 'getDebugger' as const,
    Create: ConfigureAssistantDebuggerDeploymentPage,
    Edit: EditAssistantDebuggerDeploymentPage,
    Experience: ConfigureExperience,
    deployLabel: 'Deploy Debugger',
  },
];

beforeEach(() => {
  jest.clearAllMocks();
  (ConfigureExperience as jest.Mock).mockImplementation(props => {
    const ActualExperience = jest.requireActual(
      '../configure-experience',
    ).ConfigureExperience;
    return <ActualExperience {...props} />;
  });
  mockSearchParams = new URLSearchParams();
  (CreateAssistantApiDeployment as jest.Mock).mockResolvedValue({
    getData: () => ({}),
    getSuccess: () => true,
  });
  deployments.forEach(({ save }) =>
    save.mockResolvedValue({
      getData: () => ({}),
      getSuccess: () => true,
    }),
  );
});

describe.each(deployments)('$type unclear speech persistence', config => {
  describe.each(['create', 'edit'] as const)('%s form', mode => {
    it.each(['new', 'absent', 'empty', 'explicit', 'cleared'])(
      'roundtrips the displayed values for %s settings',
      async state => {
        const saved = new config.Deployment();
        saved.setGreeting('Hello');
        if (state === 'explicit' || state === 'cleared') {
          saved.setUnclearinputtimeout(4.5);
          saved.setUnclearinputmessage('Please say that again.');
        } else if (state === 'empty') {
          saved.setUnclearinputmessage('');
        }
        config.fetch.mockResolvedValue({
          getData: () => (state === 'new' ? null : saved),
        });
        const Page = mode === 'create' ? config.Create : config.Edit;
        await act(async () => {
          render(<Page />);
        });
        if (config.type === 'phone') {
          fireEvent.click(
            screen.getByRole('button', {
              name: mode === 'create' ? 'Next' : 'General Experience',
            }),
          );
        }
        if (config.type === 'web') {
          fireEvent.click(
            screen.getByRole('button', { name: 'Show advanced settings' }),
          );
          if (state === 'new') {
            fireEvent.change(screen.getByLabelText('Greeting'), {
              target: { value: 'Hello' },
            });
          }
        }
        const message = screen.getByLabelText('Unclear Speech Message');
        const timeout = screen.getByLabelText('Unclear Speech Wait (Seconds)');
        if (state === 'cleared') {
          fireEvent.change(message, { target: { value: '' } });
        }
        const expectedMessage =
          state === 'explicit'
            ? 'Please say that again.'
            : DEFAULT_UNCLEAR_INPUT_MESSAGE;
        const expectedTimeout =
          state === 'explicit' || state === 'cleared'
            ? 4.5
            : Number(DEFAULT_UNCLEAR_INPUT_TIMEOUT);
        expect(message).toHaveValue(expectedMessage);
        expect(timeout).toHaveValue(expectedTimeout);
        if (mode === 'create') {
          fireEvent.click(screen.getByRole('button', { name: 'Next' }));
          fireEvent.click(screen.getByRole('button', { name: 'Next' }));
        }
        await act(async () => {
          fireEvent.click(
            screen.getByRole('button', {
              name: mode === 'create' ? config.deployLabel : 'Save Changes',
            }),
          );
        });
        expect(config.save).toHaveBeenCalledTimes(1);
        const sent = config.save.mock.calls[0][1][config.field]();
        const persisted = config.Deployment.deserializeBinary(
          sent.serializeBinary(),
        );
        expect(persisted.hasUnclearinputtimeout()).toBe(true);
        expect(persisted.getUnclearinputtimeout()).toBe(expectedTimeout);
        expect(persisted.hasUnclearinputmessage()).toBe(true);
        expect(persisted.getUnclearinputmessage()).toBe(expectedMessage);
      },
    );
  });

  it.each(['absent', 'empty', 'explicit', 'cleared'])(
    'saves the displayed values in shared experience edit for %s settings',
    async state => {
      const saved = new config.Deployment();
      if (state === 'explicit' || state === 'cleared') {
        saved.setUnclearinputtimeout(3.5);
        saved.setUnclearinputmessage('Repeat please.');
      } else if (state === 'empty') {
        saved.setUnclearinputmessage('');
      }
      config.fetch.mockResolvedValue({ getData: () => saved });
      const { result } = renderHook(() =>
        useDeploymentSectionEdit('assistant-1', jest.fn()),
      );
      await act(async () => {
        result.current.openEditModal(config.type, 'experience');
      });
      const Experience = config.Experience;
      const view = render(
        <Experience
          experienceConfig={{
            ...result.current.experienceConfig,
            suggestions: [],
          }}
          setExperienceConfig={result.current.setExperienceConfig}
        />,
      );
      if (config.type === 'web') {
        fireEvent.click(
          screen.getByRole('button', { name: 'Show advanced settings' }),
        );
      }
      if (state === 'cleared') {
        fireEvent.change(screen.getByLabelText('Unclear Speech Message'), {
          target: { value: '' },
        });
        view.rerender(
          <Experience
            experienceConfig={{
              ...result.current.experienceConfig,
              suggestions: [],
            }}
            setExperienceConfig={result.current.setExperienceConfig}
          />,
        );
      }
      const expectedMessage =
        state === 'explicit' ? 'Repeat please.' : DEFAULT_UNCLEAR_INPUT_MESSAGE;
      const expectedTimeout =
        state === 'explicit' || state === 'cleared'
          ? 3.5
          : Number(DEFAULT_UNCLEAR_INPUT_TIMEOUT);
      expect(screen.getByLabelText('Unclear Speech Message')).toHaveValue(
        expectedMessage,
      );
      expect(
        screen.getByLabelText('Unclear Speech Wait (Seconds)'),
      ).toHaveValue(expectedTimeout);
      await act(async () => {
        result.current.saveSection();
      });
      const sent = config.save.mock.calls[0][1][config.field]();
      expect(sent.hasUnclearinputtimeout()).toBe(true);
      expect(sent.getUnclearinputtimeout()).toBe(expectedTimeout);
      expect(sent.hasUnclearinputmessage()).toBe(true);
      expect(sent.getUnclearinputmessage()).toBe(expectedMessage);
    },
  );

  it.each(['absent', 'explicit'])(
    'preserves %s experience overrides when saving another shared section',
    async state => {
      const saved = new config.Deployment();
      if (state === 'explicit') {
        saved.setUnclearinputtimeout(6);
        saved.setUnclearinputmessage('Repeat please.');
      }
      config.fetch.mockResolvedValue({ getData: () => saved });
      const { result } = renderHook(() =>
        useDeploymentSectionEdit('assistant-1', jest.fn()),
      );
      await act(async () => {
        result.current.openEditModal(config.type, 'voice-output');
      });
      await act(async () => {
        result.current.saveSection();
      });
      const sent = config.save.mock.calls[0][1][config.field]();
      expect(sent.hasUnclearinputtimeout()).toBe(
        saved.hasUnclearinputtimeout(),
      );
      expect(sent.getUnclearinputtimeout()).toBe(
        saved.getUnclearinputtimeout(),
      );
      expect(sent.hasUnclearinputmessage()).toBe(
        saved.hasUnclearinputmessage(),
      );
      expect(sent.getUnclearinputmessage()).toBe(
        saved.getUnclearinputmessage(),
      );
    },
  );
});

it.each(['experience', 'stt', 'tts'])(
  'only applies displayed defaults in debugger section mode when editing experience (%s)',
  async section => {
    mockSearchParams = new URLSearchParams({ editMode: 'section', section });
    (GetAssistantDebuggerDeployment as jest.Mock).mockResolvedValue({
      getData: () => new AssistantDebuggerDeployment(),
    });
    await act(async () => {
      render(<ConfigureAssistantDebuggerDeploymentPage />);
    });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    });
    const sent = (
      CreateAssistantDebuggerDeployment as jest.Mock
    ).mock.calls[0][1].getDebugger();
    expect(sent.hasUnclearinputtimeout()).toBe(section === 'experience');
    expect(sent.hasUnclearinputmessage()).toBe(section === 'experience');
    if (section === 'experience') {
      expect(sent.getUnclearinputtimeout()).toBe(
        Number(DEFAULT_UNCLEAR_INPUT_TIMEOUT),
      );
      expect(sent.getUnclearinputmessage()).toBe(DEFAULT_UNCLEAR_INPUT_MESSAGE);
    }
  },
);

describe.each(deployments)('$type idle timeout section edit', config => {
  it.each([
    { name: 'legacy zero timeout', timeout: '0', count: '3', expected: '10' },
    { name: 'stored settings', timeout: '37', count: '4', expected: '37' },
    { name: 'unlimited count', timeout: '5', count: '0', expected: '5' },
    { name: 'upper bounds', timeout: '120', count: '5', expected: '120' },
  ])('loads and saves $name', async ({ timeout, count, expected }) => {
    const saved = new config.Deployment();
    saved.setIdealtimeout(timeout);
    saved.setIdealtimeoutbackoff(count);
    config.fetch.mockResolvedValue({ getData: () => saved });
    const { result } = renderHook(() =>
      useDeploymentSectionEdit('assistant-1', jest.fn()),
    );
    await act(async () => {
      result.current.openEditModal(config.type, 'experience');
    });
    expect(result.current.experienceConfig.idealTimeout).toBe(expected);
    expect(result.current.experienceConfig.idleTimeoutBackoffTimes).toBe(count);
    await act(async () => {
      result.current.saveSection();
    });
    expect(config.save).toHaveBeenCalledTimes(1);
    const sent = config.save.mock.calls[0][1][config.field]();
    expect(sent.getIdealtimeout()).toBe(expected);
    expect(sent.getIdealtimeoutbackoff()).toBe(count);
  });

  it.each([
    { name: 'omitted', timeout: undefined, count: undefined },
    { name: 'blank', timeout: '', count: '' },
    { name: 'whitespace', timeout: '   ', count: '   ' },
    { name: 'explicit unlimited count', timeout: '', count: '0' },
  ])('saves defaults for $name values', async ({ timeout, count }) => {
    config.fetch.mockResolvedValue({ getData: () => new config.Deployment() });
    const { result } = renderHook(() =>
      useDeploymentSectionEdit('assistant-1', jest.fn()),
    );
    await act(async () => {
      result.current.openEditModal(config.type, 'experience');
    });
    act(() => {
      result.current.setExperienceConfig({
        ...result.current.experienceConfig,
        idealTimeout: timeout,
        idleTimeoutBackoffTimes: count,
      });
    });
    await act(async () => {
      result.current.saveSection();
    });
    expect(config.save).toHaveBeenCalledTimes(1);
    const sent = config.save.mock.calls[0][1][config.field]();
    expect(sent.getIdealtimeout()).toBe('10');
    expect(sent.getIdealtimeoutbackoff()).toBe(count === '0' ? '0' : '2');
  });

  it.each([
    {
      name: 'timeout below minimum',
      timeout: '4',
      count: '2',
      range: /5.*120/,
    },
    {
      name: 'timeout above maximum',
      timeout: '121',
      count: '2',
      range: /5.*120/,
    },
    {
      name: 'fractional timeout',
      timeout: '10.5',
      count: '2',
      range: /5.*120/,
    },
    { name: 'NaN timeout', timeout: 'NaN', count: '2', range: /5.*120/ },
    { name: 'negative count', timeout: '10', count: '-1', range: /0.*5/ },
    { name: 'count above maximum', timeout: '10', count: '6', range: /0.*5/ },
    { name: 'fractional count', timeout: '10', count: '2.5', range: /0.*5/ },
  ])(
    'rejects $name before requesting a save',
    async ({ timeout, count, range }) => {
      config.fetch.mockResolvedValue({
        getData: () => new config.Deployment(),
      });
      const { result } = renderHook(() =>
        useDeploymentSectionEdit('assistant-1', jest.fn()),
      );
      await act(async () => {
        result.current.openEditModal(config.type, 'experience');
      });
      act(() => {
        result.current.setExperienceConfig({
          ...result.current.experienceConfig,
          idealTimeout: timeout,
          idleTimeoutBackoffTimes: count,
        });
      });
      await act(async () => {
        result.current.saveSection();
      });
      expect(config.save).not.toHaveBeenCalled();
      expect(result.current.editError).toMatch(/idle/i);
      expect(result.current.editError).toMatch(range);
      expect(result.current.isSaving).toBe(false);
    },
  );
});

describe.each([
  {
    name: 'debugger shared creation',
    Deployment: AssistantDebuggerDeployment,
    fetch: GetAssistantDebuggerDeployment as jest.Mock,
    save: CreateAssistantDebuggerDeployment as jest.Mock,
    field: 'getDebugger' as const,
    Page: ConfigureAssistantDebuggerDeploymentPage,
    mode: 'create',
    saveLabel: 'Deploy Debugger',
  },
  {
    name: 'standalone API creation',
    Deployment: AssistantApiDeployment,
    fetch: GetAssistantApiDeployment as jest.Mock,
    save: CreateAssistantApiDeployment as jest.Mock,
    field: 'getApi' as const,
    Page: ConfigureAssistantApiDeploymentPage,
    mode: 'create',
    saveLabel: 'Deploy API',
  },
  {
    name: 'standalone API edit',
    Deployment: AssistantApiDeployment,
    fetch: GetAssistantApiDeployment as jest.Mock,
    save: CreateAssistantApiDeployment as jest.Mock,
    field: 'getApi' as const,
    Page: EditAssistantApiDeploymentPage,
    mode: 'edit',
    saveLabel: 'Save Changes',
  },
])('$name idle timeout', config => {
  it.each([
    { name: 'legacy zero timeout', timeout: '0', count: '3', expected: '10' },
    { name: 'stored settings', timeout: '37', count: '4', expected: '37' },
    { name: 'unlimited count', timeout: '5', count: '0', expected: '5' },
  ])('loads and saves $name', async ({ timeout, count, expected }) => {
    const saved = new config.Deployment();
    saved.setIdealtimeout(timeout);
    saved.setIdealtimeoutbackoff(count);
    config.fetch.mockResolvedValue({
      getData: () => saved,
    });
    await act(async () => {
      render(<config.Page />);
    });
    const experience = (ConfigureExperience as jest.Mock).mock.calls.slice(
      -1,
    )[0][0];
    expect(experience.experienceConfig.idealTimeout).toBe(expected);
    expect(experience.experienceConfig.idleTimeoutBackoffTimes).toBe(count);
    if (config.mode === 'create') {
      fireEvent.click(screen.getByRole('button', { name: 'Next' }));
      fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    }
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: config.saveLabel }));
    });
    expect(config.save).toHaveBeenCalledTimes(1);
    const sent = config.save.mock.calls[0][1][config.field]();
    expect(sent.getIdealtimeout()).toBe(expected);
    expect(sent.getIdealtimeoutbackoff()).toBe(count);
  });

  it.each([
    { name: 'omitted', timeout: undefined, count: undefined },
    { name: 'blank', timeout: '', count: '' },
    { name: 'whitespace', timeout: '   ', count: '   ' },
    { name: 'explicit unlimited count', timeout: '', count: '0' },
  ])('saves defaults for $name values', async ({ timeout, count }) => {
    config.fetch.mockResolvedValue({
      getData: () => new config.Deployment(),
    });
    await act(async () => {
      render(<config.Page />);
    });
    const experience = (ConfigureExperience as jest.Mock).mock.calls.slice(
      -1,
    )[0][0];
    act(() => {
      experience.setExperienceConfig({
        ...experience.experienceConfig,
        idealTimeout: timeout,
        idleTimeoutBackoffTimes: count,
      });
    });
    if (config.mode === 'create') {
      fireEvent.click(screen.getByRole('button', { name: 'Next' }));
      fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    }
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: config.saveLabel }));
    });
    expect(config.save).toHaveBeenCalledTimes(1);
    const sent = config.save.mock.calls[0][1][config.field]();
    expect(sent.getIdealtimeout()).toBe('10');
    expect(sent.getIdealtimeoutbackoff()).toBe(count === '0' ? '0' : '2');
  });

  it.each([
    {
      name: 'timeout below minimum',
      timeout: '4',
      count: '2',
      range: /5.*120/,
    },
    {
      name: 'timeout above maximum',
      timeout: '121',
      count: '2',
      range: /5.*120/,
    },
    {
      name: 'fractional timeout',
      timeout: '10.5',
      count: '2',
      range: /5.*120/,
    },
    { name: 'NaN timeout', timeout: 'NaN', count: '2', range: /5.*120/ },
    { name: 'negative count', timeout: '10', count: '-1', range: /0.*5/ },
    { name: 'count above maximum', timeout: '10', count: '6', range: /0.*5/ },
    { name: 'fractional count', timeout: '10', count: '2.5', range: /0.*5/ },
  ])(
    'rejects $name before requesting a save',
    async ({ timeout, count, range }) => {
      config.fetch.mockResolvedValue({
        getData: () => new config.Deployment(),
      });
      await act(async () => {
        render(<config.Page />);
      });
      const experience = (ConfigureExperience as jest.Mock).mock.calls.slice(
        -1,
      )[0][0];
      act(() => {
        experience.setExperienceConfig({
          ...experience.experienceConfig,
          idealTimeout: timeout,
          idleTimeoutBackoffTimes: count,
        });
      });
      if (config.mode === 'create') {
        fireEvent.click(screen.getByRole('button', { name: 'Next' }));
        fireEvent.click(screen.getByRole('button', { name: 'Next' }));
      }
      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: config.saveLabel }));
      });
      expect(config.save).not.toHaveBeenCalled();
      expect(screen.getByRole('alert')).toHaveTextContent(/idle/i);
      expect(screen.getByRole('alert')).toHaveTextContent(range);
      expect(
        screen.getByRole('button', { name: config.saveLabel }),
      ).toBeEnabled();
    },
  );
});

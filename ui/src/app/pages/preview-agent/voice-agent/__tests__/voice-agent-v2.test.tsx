import React from 'react';
import '@testing-library/jest-dom';
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent, {
  PointerEventsCheckLevel,
} from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { PreviewVoiceAgentV2 } from '../v2';

const mockAddMessage = jest.fn();
const mockAddMessageChunk = jest.fn();
const mockSend = jest.fn();
const mockOnSendText = jest.fn();
const mockSetInputChannel = jest.fn();
const mockDisconnect = jest.fn();
const mockToggleMute = jest.fn();
const mockSetActiveMediaDevice = jest.fn();
const mockAddArgument = jest.fn();
const mockWriteText = jest.fn();
const mockGetAssistant = jest.fn();
const mockRegisterCallback = jest.fn();
const mockOnAgentEvent = jest.fn();
const mockOffAgentEvent = jest.fn();
let mockCarbonChatProps: Record<string, any>;
let mockResolvedMode = 'light';
let mockIsOverlayPanel = false;
let mockIsSmallViewport = false;
let mockAgentCallback: Record<string, (...args: any[]) => void>;
let mockConnectionStateHandler: ((state: string) => void) | undefined;
let mockAgentErrorHandler: ((type: string, error: string) => void) | undefined;
let resolveVoiceConnection: (() => void) | undefined;

const mockModelOptions = [
  ['model.name', 'gpt-4.1-mini'],
  ['model.temperature', '0.7'],
  ['model.top_p', '1'],
  ['model.max_completion_tokens', '1048'],
  ['rapida.credential_id', 'credential-secret-123'],
].map(([key, value]) => ({
  getKey: () => key,
  getValue: () => value,
}));
const mockSpeechToTextOptions = [['listen.model', 'nova-3']].map(
  ([key, value]) => ({
    getKey: () => key,
    getValue: () => value,
  }),
);
const mockTextToSpeechOptions = [
  ['speak.model', 'sonic-2'],
  ['speak.voice.id', '6ccbfb76-1fc6-48f7-b71d-91ac6298247b'],
  ['speak.language', 'en'],
].map(([key, value]) => ({
  getKey: () => key,
  getValue: () => value,
}));
const mockVariables = [
  { name: 'customer_name', value: 'Prashant', type: 'string' },
  {
    name: 'delivery_instructions',
    value: 'Leave at reception',
    type: 'paragraph',
  },
].map(variable => ({
  getName: () => variable.name,
  getDefaultvalue: () => variable.value,
  getType: () => variable.type,
}));
const mockAssistant = {
  getName: () => 'Delivery concierge',
  getAssistantprovidermodel: () => ({
    getModelprovidername: () => 'OpenAI',
    getAssistantmodeloptionsList: () => mockModelOptions,
    getTemplate: () => ({
      getPromptvariablesList: () => mockVariables,
    }),
  }),
  getDebuggerdeployment: () => ({
    getInputaudio: () => ({
      getAudioprovider: () => 'Deepgram',
      getAudiooptionsList: () => mockSpeechToTextOptions,
    }),
    getOutputaudio: () => ({
      getAudioprovider: () => 'Cartesia',
      getAudiooptionsList: () => mockTextToSpeechOptions,
    }),
  }),
};
const mockAgentConfiguration = {
  inputOptions: { channel: 'text' },
  outputOptions: { channel: 'text' },
  addArgument: mockAddArgument,
};
const mockVoiceAgent = {
  agentConfiguration: mockAgentConfiguration,
  isConnected: false,
  messages: [] as Array<{
    id: string;
    role: 'system' | 'user';
    messages: string[];
  }>,
  getAssistant: mockGetAssistant,
  registerCallback: mockRegisterCallback,
  on: mockOnAgentEvent,
  off: mockOffAgentEvent,
  onSendText: mockOnSendText,
  setInputChannel: mockSetInputChannel,
  disconnect: mockDisconnect,
  toggleMute: mockToggleMute,
};

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    user: { name: 'Prashant' },
    authId: 'user-123',
    token: 'token-123',
    projectId: 'project-123',
  }),
}));

jest.mock('@/clients/runtime.client', () => ({
  createVoiceAgentDebuggerConnection: jest.fn(() => ({
    kind: 'debugger-connection',
  })),
}));

jest.mock('@rapidaai/react', () => {
  class InputOptions {
    channels: string[];
    channel: string;

    constructor(channels: string[], channel: string) {
      this.channels = channels;
      this.channel = channel;
    }
  }

  class AgentConfig {
    inputOptions: InputOptions;
    outputOptions: { channel: string };
    addArgument = mockAddArgument;
    addKeywords = jest.fn(() => this);
    addMetadata = jest.fn(() => this);
    setUserIdentifier = jest.fn(() => this);

    constructor(_id: string, inputOptions: InputOptions) {
      this.inputOptions = inputOptions;
      this.outputOptions = { channel: inputOptions.channel };
    }
  }

  return {
    AgentConfig,
    Channel: { Audio: 'audio', Text: 'text' },
    InputOptions,
    StringToAny: (value: string) => value,
    VoiceAgent: jest.fn((_connection, agentConfiguration) => {
      mockVoiceAgent.agentConfiguration = agentConfiguration;
      return mockVoiceAgent;
    }),
    useSelectInputDeviceAgent: () => ({
      devices: [
        { deviceId: 'built-in', label: 'Built-in Microphone' },
        { deviceId: 'usb', label: 'USB Microphone' },
      ],
      activeDeviceId: 'built-in',
      setActiveMediaDevice: mockSetActiveMediaDevice,
    }),
  };
});

jest.mock(
  '@carbon/ai-chat',
  () => ({
    CarbonTheme: { G100: 'g100', WHITE: 'white' },
    ChatCustomElement: (props: Record<string, any>) => {
      mockCarbonChatProps = props;
      return (
        <div aria-label="Carbon AI chat">
          {props.renderWriteableElements?.customHeader}
          <div aria-label="Voice session">
            {props.renderWriteableElements?.beforeInputElement}
          </div>
          <div aria-label="Composer actions">
            {props.renderWriteableElements?.promptLineActionsEnd}
          </div>
          <div aria-label="Composer controls">
            {props.renderWriteableElements?.promptLineSendButtonStart}
          </div>
        </div>
      );
    },
    MessageInputType: { TEXT: 'text' },
    MessageResponseTypes: { TEXT: 'text' },
    WriteableElementName: {
      CUSTOM_HEADER: 'customHeader',
      BEFORE_INPUT_ELEMENT: 'beforeInputElement',
      PROMPT_LINE_ACTIONS_END: 'promptLineActionsEnd',
      PROMPT_LINE_SEND_BUTTON_START: 'promptLineSendButtonStart',
    },
  }),
  { virtual: true },
);

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => ({ resolvedMode: mockResolvedMode }),
}));

jest.mock('../preview-agent-header', () => ({
  PreviewAgentHeader: () => <header>Product header</header>,
}));

describe('PreviewVoiceAgentV2', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: mockWriteText },
    });
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: jest.fn(),
    });
    mockAddMessage.mockReset().mockResolvedValue(undefined);
    mockAddMessageChunk.mockReset().mockResolvedValue(undefined);
    mockWriteText.mockReset().mockResolvedValue(undefined);
    mockSend.mockReset().mockResolvedValue(undefined);
    mockOnSendText.mockReset().mockImplementation(async () => {
      mockVoiceAgent.isConnected = true;
    });
    mockDisconnect.mockReset().mockImplementation(async () => {
      mockVoiceAgent.isConnected = false;
      mockConnectionStateHandler?.('disconnected');
    });
    mockToggleMute.mockReset().mockReturnValue(true);
    mockSetActiveMediaDevice.mockReset().mockResolvedValue(undefined);
    mockAddArgument
      .mockReset()
      .mockImplementation(() => mockAgentConfiguration);
    mockGetAssistant.mockReset().mockResolvedValue({
      getSuccess: () => true,
      getData: () => mockAssistant,
    });
    mockRegisterCallback.mockReset().mockImplementation(callback => {
      mockAgentCallback = callback;
    });
    mockOnAgentEvent.mockReset().mockImplementation((event, handler) => {
      if (event === 'onConnectionStateEvent') {
        mockConnectionStateHandler = handler;
      } else if (event === 'onErrorEvent') {
        mockAgentErrorHandler = handler;
      }
    });
    mockOffAgentEvent.mockReset();
    mockSetInputChannel.mockReset().mockImplementation(channel => {
      if (channel === 'audio') {
        return new Promise<void>(resolve => {
          resolveVoiceConnection = () => {
            mockVoiceAgent.agentConfiguration.inputOptions.channel = channel;
            mockVoiceAgent.agentConfiguration.outputOptions.channel = channel;
            mockVoiceAgent.isConnected = true;
            mockConnectionStateHandler?.('connected');
            resolve();
          };
        });
      }
      mockVoiceAgent.agentConfiguration.inputOptions.channel = channel;
      mockVoiceAgent.agentConfiguration.outputOptions.channel = channel;
      return Promise.resolve();
    });
    mockVoiceAgent.messages = [];
    mockVoiceAgent.isConnected = false;
    mockVoiceAgent.agentConfiguration = mockAgentConfiguration;
    mockAgentConfiguration.inputOptions.channel = 'text';
    mockAgentConfiguration.outputOptions.channel = 'text';
    mockAgentCallback = {};
    mockConnectionStateHandler = undefined;
    mockAgentErrorHandler = undefined;
    resolveVoiceConnection = undefined;
    mockCarbonChatProps = {};
    mockResolvedMode = 'light';
    mockIsOverlayPanel = false;
    mockIsSmallViewport = false;
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockImplementation(query => ({
        get matches() {
          return query === '(max-width: 671px)'
            ? mockIsSmallViewport
            : mockIsOverlayPanel;
        },
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      })),
    });
  });

  it('renders the debugger workspace and opens focused controls', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
      writeToClipboard: false,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole('heading', { name: 'Delivery concierge' }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Carbon AI chat')).toBeInTheDocument();
    expect(mockCarbonChatProps.openChatByDefault).toBe(true);
    expect(mockCarbonChatProps.injectCarbonTheme).toBe('white');
    expect(mockCarbonChatProps.header).toEqual({ isOn: true });
    expect(mockCarbonChatProps.homescreen).toEqual({ isOn: false });
    expect(mockCarbonChatProps.messaging.skipWelcome).toBe(false);
    expect(mockCarbonChatProps.layout).toMatchObject({
      showFrame: false,
      hasContentMaxWidth: true,
      customProperties: {
        'messages-max-width': '45rem',
        '$chat-shell-background': '#f4f4f4',
        '$chat-prompt-background': '#ffffff',
        '$ai-aura-start-sm': '#f4f4f4',
        '$ai-aura-end': '#f4f4f4',
      },
    });
    const toolbar = screen.getByRole('toolbar', {
      name: 'Conversation toolbar',
    });
    expect(toolbar).toBeInTheDocument();
    expect(toolbar).toHaveClass('h-12!');
    expect(within(toolbar).getAllByRole('button')[0]).toHaveAccessibleName(
      'Back to assistant',
    );
    expect(
      screen.queryByRole('button', { name: 'New conversation' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Conversation history' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Session 04' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Restart conversation' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'More conversation actions' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('Voice agent debugger · Development'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('complementary', { name: 'Session inspector' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Test setup' }),
    ).not.toBeInTheDocument();
    const configurationButton = within(toolbar).getByRole('button', {
      name: 'Configuration',
    });
    expect(configurationButton).toHaveAttribute('aria-expanded', 'false');
    expect(
      within(toolbar).queryByRole('button', { name: 'Arguments' }),
    ).not.toBeInTheDocument();
    expect(mockCarbonChatProps.input.expanded).toBe(true);
    const startVoiceButton = screen.getByRole('button', {
      name: 'Start voice conversation',
    });
    expect(startVoiceButton).toBeInTheDocument();
    expect(startVoiceButton).not.toHaveTextContent('Start voice conversation');
    expect(
      within(screen.getByLabelText('Composer controls')).getByRole('button', {
        name: 'Start voice conversation',
      }),
    ).toBe(startVoiceButton);
    expect(
      within(screen.getByLabelText('Composer actions')).queryByRole('button'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Start voice input' }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('Connected')).not.toBeInTheDocument();
    await waitFor(() => expect(mockGetAssistant).toHaveBeenCalledTimes(1));
    const history = await mockCarbonChatProps.messaging.customLoadHistory({
      messaging: {
        addMessage: mockAddMessage,
      },
      send: mockSend,
    });
    expect(history).toEqual([]);
    await user.click(configurationButton);

    expect(configurationButton).toHaveAttribute('aria-expanded', 'true');
    expect(
      screen.getByRole('complementary', { name: 'Session configuration' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Session configuration' }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('heading', { name: 'Model and voice' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('Services used for this voice session.'),
    ).not.toBeInTheDocument();
    expect(screen.getByLabelText('Session settings')).toBeInTheDocument();
    expect(screen.getByText('Speech to text')).toBeInTheDocument();
    expect(screen.getByText('Text to speech')).toBeInTheDocument();
    const configuredServices = screen.getByRole('list', {
      name: 'Session configuration sections',
    });
    expect(configuredServices).toHaveClass('debugger-configuration-accordion');
    const languageModelToggle = within(configuredServices).getByRole('button', {
      name: 'Language model, OpenAI, gpt-4.1-mini',
    });
    const speechToTextToggle = within(configuredServices).getByRole('button', {
      name: 'Speech to text, Deepgram, nova-3',
    });
    const textToSpeechToggle = within(configuredServices).getByRole('button', {
      name: 'Text to speech, Cartesia, sonic-2',
    });
    const argumentsToggle = within(configuredServices).getByRole('button', {
      name: 'Arguments, 2 variables',
    });
    expect(languageModelToggle).toHaveAttribute('aria-expanded', 'true');
    expect(speechToTextToggle).toHaveAttribute('aria-expanded', 'false');
    expect(textToSpeechToggle).toHaveAttribute('aria-expanded', 'false');
    expect(argumentsToggle).toHaveAttribute('aria-expanded', 'false');
    expect(
      within(configuredServices).getByText('OpenAI · gpt-4.1-mini'),
    ).toBeVisible();
    expect(
      within(configuredServices).getByText('Deepgram · nova-3'),
    ).toBeVisible();
    expect(
      within(configuredServices).getByText('Cartesia · sonic-2'),
    ).toBeVisible();
    expect(within(configuredServices).getByText('2 variables')).toBeVisible();

    await user.click(speechToTextToggle);
    await user.click(textToSpeechToggle);

    expect(speechToTextToggle).toHaveAttribute('aria-expanded', 'true');
    expect(textToSpeechToggle).toHaveAttribute('aria-expanded', 'true');
    const languageModelConfiguration = within(configuredServices).getByRole(
      'table',
      { name: 'Language model configuration' },
    );
    const speechToTextConfiguration = within(configuredServices).getByRole(
      'table',
      { name: 'Speech to text configuration' },
    );
    const textToSpeechConfiguration = within(configuredServices).getByRole(
      'table',
      { name: 'Text to speech configuration' },
    );
    const expectedConfigurations = [
      {
        table: languageModelConfiguration,
        rows: [
          ['Provider', 'provider', 'OpenAI'],
          ['Model', 'model.name', 'gpt-4.1-mini'],
          ['Temperature', 'model.temperature', '0.7'],
          ['Top P', 'model.top_p', '1'],
          ['Max completion tokens', 'model.max_completion_tokens', '1048'],
        ],
      },
      {
        table: speechToTextConfiguration,
        rows: [
          ['Provider', 'provider', 'Deepgram'],
          ['Model', 'listen.model', 'nova-3'],
        ],
      },
      {
        table: textToSpeechConfiguration,
        rows: [
          ['Provider', 'provider', 'Cartesia'],
          ['Model', 'speak.model', 'sonic-2'],
          ['Voice', 'speak.voice.id', '6ccbfb76-1fc6-48f7-b71d-91ac6298247b'],
          ['Language', 'speak.language', 'en'],
        ],
      },
    ];
    expectedConfigurations.forEach(configuration => {
      const renderedRows = within(configuration.table).getAllByRole('row');
      expect(renderedRows).toHaveLength(configuration.rows.length);
      configuration.rows.forEach(([label, key, value], index) => {
        const cells = within(renderedRows[index]).getAllByRole('cell');
        expect(cells).toHaveLength(2);
        expect(cells[0]).toHaveClass('!pl-4');
        expect(cells[1]).toHaveClass('!pr-4');
        expect(cells[0].textContent).toBe(`${label}${key}`);
        expect(within(cells[1]).getByText(value)).toBeInTheDocument();
      });
    });
    expect(
      within(configuredServices).queryByText('rapida.credential_id'),
    ).not.toBeInTheDocument();
    expect(
      within(configuredServices).queryByText('credential-secret-123'),
    ).not.toBeInTheDocument();
    const copyModelButton = within(languageModelConfiguration).getByRole(
      'button',
      { name: 'Copy Model' },
    );
    expect(
      within(languageModelConfiguration).queryByRole('button', {
        name: 'Copy Provider',
      }),
    ).not.toBeInTheDocument();
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: mockWriteText },
    });
    act(() => copyModelButton.click());
    expect(mockWriteText).toHaveBeenCalledWith('gpt-4.1-mini');
    expect(
      within(textToSpeechConfiguration).getByRole('button', {
        name: 'Copy Voice',
      }),
    ).toBeInTheDocument();

    await user.click(argumentsToggle);

    expect(argumentsToggle).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByTestId('arguments-accordion-content')).toHaveClass(
      'px-4',
    );
    expect(screen.getByRole('form', { name: 'Arguments' })).toBeInTheDocument();
    expect(
      screen.queryByRole('tree', { name: 'Session configuration sections' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('table', { name: 'Configured services' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('These services process every turn in this preview.'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('Active for this test run'),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('Type')).not.toBeInTheDocument();
    expect(
      screen.queryByText('Applied to your next message'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('Changes apply to the next turn'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Add variable' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Apply changes' }),
    ).not.toBeInTheDocument();

    const customerName = screen.getByRole('textbox', {
      name: '{{customer_name}}',
    });
    const deliveryInstructions = screen.getByRole('textbox', {
      name: '{{delivery_instructions}}',
    });
    expect(customerName.tagName).toBe('INPUT');
    expect(deliveryInstructions.tagName).toBe('TEXTAREA');
    await user.clear(customerName);
    await user.type(customerName, 'Avery');

    expect(customerName).toHaveValue('Avery');
    expect(mockAddArgument).toHaveBeenCalledWith('customer_name', 'Avery');

    await act(async () => {
      await mockCarbonChatProps.messaging.customSendMessage(
        {
          id: 'request-1',
          input: { message_type: 'text', text: 'Where is my delivery?' },
        },
        { signal: new AbortController().signal, silent: false },
        {
          messaging: {
            addMessage: mockAddMessage,
          },
          send: mockSend,
        },
      );
    });

    expect(mockOnSendText).toHaveBeenCalledWith('Where is my delivery?');
    mockVoiceAgent.messages = [
      {
        id: 'assistant-response-1',
        role: 'system',
        messages: ['Your delivery is Friday.'],
      },
    ];
    await act(async () => {
      mockAgentCallback.onAssistantMessage?.({
        id: 'assistant-response-1',
        messageText: '',
        completed: true,
      });
    });
    expect(mockAddMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        request_id: 'request-1',
        output: {
          generic: [
            expect.objectContaining({
              text: 'Your delivery is Friday.',
            }),
          ],
        },
      }),
    );
    await user.click(
      screen.getByRole('button', { name: 'Close session configuration' }),
    );
    expect(
      screen.queryByRole('complementary', { name: 'Session configuration' }),
    ).not.toBeInTheDocument();
    await waitFor(() => expect(configurationButton).toHaveFocus());
  });

  it('uses the assistant shell chrome when embedded', async () => {
    render(
      <MemoryRouter
        initialEntries={['/deployment/assistant/assistant-123/playground/chat']}
      >
        <Routes>
          <Route
            path="/deployment/assistant/:assistantId/playground/chat"
            element={<PreviewVoiceAgentV2 embedded />}
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole('heading', { name: 'Delivery concierge' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Product header')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Back to assistant' }),
    ).not.toBeInTheDocument();
  });

  it('removes the Carbon composer bottom margin on small screens', async () => {
    mockIsSmallViewport = true;

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });

    expect(mockCarbonChatProps.layout.hasContentMaxWidth).toBe(false);
  });

  it('renders the assistant response when runtime messages share an id', async () => {
    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    const chatInstance = {
      messaging: { addMessage: mockAddMessage },
      send: mockSend,
    };
    await mockCarbonChatProps.messaging.customLoadHistory(chatInstance);
    await act(async () => {
      await mockCarbonChatProps.messaging.customSendMessage(
        {
          id: 'request-1',
          input: { message_type: 'text', text: 'Where is my delivery?' },
        },
        { signal: new AbortController().signal, silent: false },
        chatInstance,
      );
    });

    mockVoiceAgent.messages = [
      {
        id: 'shared-message-id',
        role: 'user',
        messages: ['Where is my delivery?'],
      },
      {
        id: 'shared-message-id',
        role: 'system',
        messages: ['Your delivery is Friday.'],
      },
    ];
    await act(async () => {
      mockAgentCallback.onAssistantMessage?.({
        id: 'shared-message-id',
        messageText: 'Your delivery is Friday.',
        completed: true,
      });
    });

    expect(mockAddMessage).toHaveBeenLastCalledWith(
      expect.objectContaining({
        output: {
          generic: [
            expect.objectContaining({ text: 'Your delivery is Friday.' }),
          ],
        },
      }),
    );
  });

  it('streams assistant markdown through Carbon AI Chat', async () => {
    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    const chatInstance = {
      messaging: {
        addMessage: mockAddMessage,
        addMessageChunk: mockAddMessageChunk,
      },
      send: mockSend,
    };
    await mockCarbonChatProps.messaging.customLoadHistory(chatInstance);
    await act(async () => {
      await mockCarbonChatProps.messaging.customSendMessage(
        {
          id: 'request-1',
          input: { message_type: 'text', text: 'When is my delivery?' },
        },
        { signal: new AbortController().signal, silent: false },
        chatInstance,
      );
      mockAgentCallback.onAssistantMessage?.({
        id: 'markdown-response',
        messageText: '**Delivery',
        completed: false,
      });
      mockAgentCallback.onAssistantMessage?.({
        id: 'markdown-response',
        messageText: ' Friday**',
        completed: false,
      });
    });

    mockVoiceAgent.messages = [
      {
        id: 'markdown-response',
        role: 'system',
        messages: ['**Delivery Friday**'],
      },
    ];
    await act(async () => {
      mockAgentCallback.onAssistantMessage?.({
        id: 'markdown-response',
        messageText: '',
        completed: true,
      });
    });

    const streamingResponseId =
      mockAddMessageChunk.mock.calls[0][0].streaming_metadata.response_id;
    expect(streamingResponseId).toMatch(/^assistant-response-/);
    expect(mockAddMessageChunk).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        streaming_metadata: { response_id: streamingResponseId },
        partial_item: expect.objectContaining({
          response_type: 'text',
          text: '**Delivery',
        }),
      }),
    );
    expect(mockAddMessageChunk).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        partial_item: expect.objectContaining({ text: ' Friday**' }),
      }),
    );
    expect(mockAddMessageChunk).toHaveBeenLastCalledWith(
      expect.objectContaining({
        final_response: expect.objectContaining({
          id: streamingResponseId,
          request_id: 'request-1',
          output: {
            generic: [expect.objectContaining({ text: '**Delivery Friday**' })],
          },
        }),
      }),
    );
    expect(mockAddMessage).not.toHaveBeenCalled();
  });

  it('preserves multiple assistant messages emitted for one runtime message id', async () => {
    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    const chatInstance = {
      messaging: { addMessage: mockAddMessage },
      send: mockSend,
    };
    await mockCarbonChatProps.messaging.customLoadHistory(chatInstance);
    await mockCarbonChatProps.messaging.customSendMessage(
      {
        id: 'request-1',
        input: { message_type: 'text', text: 'Give me both updates' },
      },
      { signal: new AbortController().signal, silent: false },
      chatInstance,
    );

    mockVoiceAgent.messages = [
      {
        id: 'shared-response-id',
        role: 'system',
        messages: ['First update.'],
      },
    ];
    await act(async () => {
      mockAgentCallback.onAssistantMessage?.({
        id: 'shared-response-id',
        messageText: 'First update.',
        completed: true,
      });
    });

    mockVoiceAgent.messages.push({
      id: 'shared-response-id',
      role: 'system',
      messages: ['Second update.'],
    });
    await act(async () => {
      mockAgentCallback.onAssistantMessage?.({
        id: 'shared-response-id',
        messageText: 'Second update.',
        completed: true,
      });
    });

    const firstMessage = mockAddMessage.mock.calls[0][0];
    const secondMessage = mockAddMessage.mock.calls[1][0];
    expect(firstMessage.output.generic[0].text).toBe('First update.');
    expect(secondMessage.output.generic[0].text).toBe('Second update.');
    expect(secondMessage.id).not.toBe(firstMessage.id);
    expect(firstMessage.request_id).toBe('request-1');
    expect(secondMessage.request_id).toBe('request-1');
  });

  it('does not show voice connecting for a text transport connection', async () => {
    mockOnSendText.mockImplementationOnce(async () => {
      mockConnectionStateHandler?.('connecting');
      mockVoiceAgent.isConnected = true;
    });
    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    const chatInstance = {
      messaging: { addMessage: mockAddMessage },
      send: mockSend,
    };
    await act(async () => {
      await mockCarbonChatProps.messaging.customSendMessage(
        {
          id: 'request-1',
          input: { message_type: 'text', text: 'Where is my delivery?' },
        },
        { signal: new AbortController().signal, silent: false },
        chatInstance,
      );
    });

    expect(mockOnSendText).toHaveBeenCalledWith('Where is my delivery?');
    expect(screen.queryByText('Connecting voice...')).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    ).toBeInTheDocument();
  });

  it('disconnects an active text chat session beside the send control', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });
    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await mockCarbonChatProps.messaging.customLoadHistory({
      messaging: {
        addMessage: mockAddMessage,
        addMessageChunk: mockAddMessageChunk,
      },
      send: mockSend,
    });
    const conversationToolbar = screen.getByRole('toolbar', {
      name: 'Conversation toolbar',
    });
    const composerControls = screen.getByLabelText('Composer controls');
    expect(
      within(conversationToolbar).queryByRole('button', {
        name: 'Stop chat session',
      }),
    ).not.toBeInTheDocument();
    expect(
      within(composerControls).queryByRole('button', {
        name: 'Stop chat session',
      }),
    ).not.toBeInTheDocument();
    await act(async () => {
      mockAgentCallback.onAssistantMessage?.({
        id: 'active-response',
        messageText: '**Partial response',
        completed: false,
      });
      mockConnectionStateHandler?.('connected');
    });

    expect(
      within(conversationToolbar).queryByRole('button', {
        name: 'Stop chat session',
      }),
    ).not.toBeInTheDocument();
    const chatInputControls = within(composerControls).getByRole('group', {
      name: 'Chat input controls',
    });
    expect(chatInputControls).toHaveClass('flex', 'h-8', 'items-center');
    expect(
      within(chatInputControls)
        .getByRole('button', { name: 'Start voice conversation' })
        .querySelector('svg'),
    ).toHaveStyle({ fill: 'var(--cds-link-primary)' });
    expect(
      within(chatInputControls)
        .getByRole('button', { name: 'Stop chat session' })
        .querySelector('svg'),
    ).toHaveStyle({ fill: 'var(--cds-support-error)' });
    await user.click(
      within(chatInputControls).getByRole('button', {
        name: 'Stop chat session',
      }),
    );

    expect(mockDisconnect).toHaveBeenCalledTimes(1);
    expect(mockAddMessageChunk).toHaveBeenLastCalledWith(
      expect.objectContaining({
        final_response: expect.objectContaining({
          output: {
            generic: [
              expect.objectContaining({
                text: '**Partial response',
                streaming_metadata: expect.objectContaining({
                  stream_stopped: true,
                }),
              }),
            ],
          },
        }),
      }),
    );
    expect(
      within(composerControls).queryByRole('button', {
        name: 'Stop chat session',
      }),
    ).not.toBeInTheDocument();
  });

  it('answers the Carbon welcome request without calling the runtime', async () => {
    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await act(async () => {
      await mockCarbonChatProps.messaging.customSendMessage(
        {
          id: 'welcome-request',
          history: { is_welcome_request: true, silent: true },
          input: { text: '' },
        },
        { signal: new AbortController().signal, silent: true },
        {
          messaging: { addMessage: mockAddMessage },
          send: mockSend,
        },
      );
    });

    expect(mockAddMessage).toHaveBeenCalledWith({
      id: 'welcome-welcome-request',
      request_id: 'welcome-request',
      output: {
        generic: [
          {
            response_type: 'text',
            text: 'Hello! How can I help you today?',
          },
        ],
      },
    });
    expect(mockOnSendText).not.toHaveBeenCalled();
  });

  it('shows an empty state when the assistant has no arguments', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });
    mockGetAssistant.mockResolvedValueOnce({
      getSuccess: () => true,
      getData: () => ({
        ...mockAssistant,
        getAssistantprovidermodel: () => ({
          ...mockAssistant.getAssistantprovidermodel(),
          getTemplate: () => ({ getPromptvariablesList: () => [] }),
        }),
      }),
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(screen.getByRole('button', { name: 'Configuration' }));
    await user.click(
      screen.getByRole('button', { name: 'Arguments, 0 variables' }),
    );

    expect(screen.getByText('No arguments defined')).toBeInTheDocument();
    expect(
      screen.getByText('This assistant does not expose prompt variables.'),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('form', { name: 'Arguments' }),
    ).not.toBeInTheDocument();
  });

  it('uses numeric controls and blocks non-editable argument types', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });
    const typedVariables = [
      { name: 'retry_count', value: '3', type: 'number' },
      { name: 'knowledge', value: '', type: 'contexts' },
    ].map(variable => ({
      getName: () => variable.name,
      getDefaultvalue: () => variable.value,
      getType: () => variable.type,
    }));
    mockGetAssistant.mockResolvedValueOnce({
      getSuccess: () => true,
      getData: () => ({
        ...mockAssistant,
        getAssistantprovidermodel: () => ({
          ...mockAssistant.getAssistantprovidermodel(),
          getTemplate: () => ({ getPromptvariablesList: () => typedVariables }),
        }),
      }),
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(screen.getByRole('button', { name: 'Configuration' }));
    await user.click(
      screen.getByRole('button', { name: 'Arguments, 2 variables' }),
    );

    expect(
      screen.getByRole('spinbutton', { name: '{{retry_count}}' }),
    ).toHaveValue(3);
    expect(screen.getByText('{{knowledge}}')).toBeInTheDocument();
    expect(screen.getByText('Not editable in debugger')).toBeInTheDocument();
    expect(
      screen.getByText('Contexts arguments cannot be edited in this debugger.'),
    ).toBeInTheDocument();
  });

  it('explains unavailable voice input and prevents starting voice', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });
    mockGetAssistant.mockResolvedValueOnce({
      getSuccess: () => true,
      getData: () => ({
        ...mockAssistant,
        getDebuggerdeployment: () => ({
          getInputaudio: () => undefined,
          getOutputaudio: () =>
            mockAssistant.getDebuggerdeployment().getOutputaudio(),
        }),
      }),
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
          <Route
            path="/deployment/assistant/:assistantId/deployment/debugger"
            element={<div>Configure debugger voice</div>}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    expect(
      screen.queryByText(
        'Configure speech to text to start a voice conversation.',
      ),
    ).not.toBeVisible();
    const unavailableVoiceButton = screen.getByRole('button', {
      name: 'Voice is not configured',
    });
    expect(unavailableVoiceButton).toHaveAttribute('aria-expanded', 'false');
    expect(
      screen.queryByRole('button', { name: 'Start voice conversation' }),
    ).not.toBeInTheDocument();
    await user.click(unavailableVoiceButton);
    expect(unavailableVoiceButton).toHaveAttribute('aria-expanded', 'true');
    expect(
      screen.getByText(
        'Configure speech to text to start a voice conversation.',
      ),
    ).toBeInTheDocument();
    expect(mockSetInputChannel).not.toHaveBeenCalled();

    const configureVoiceButton = screen
      .getByText('Configure voice')
      .closest('button');
    expect(configureVoiceButton).not.toBeNull();
    await user.click(configureVoiceButton!);
    expect(screen.getByText('Configure debugger voice')).toBeInTheDocument();
  });

  it('presents voice as a connected session and returns to text mode', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    );

    const connectingStatus = screen.getByText('Connecting voice...');
    expect(connectingStatus).toBeInTheDocument();
    expect(connectingStatus.closest('.border-t')).toBeNull();
    expect(connectingStatus.closest('.cds--inline-loading')).toHaveClass(
      'whitespace-nowrap',
    );
    expect(screen.getByTestId('voice-session-dock')).toHaveClass(
      'bg-[var(--cds-chat-prompt-background,var(--cds-background))]',
    );
    expect(screen.getByTestId('voice-connecting-row')).toHaveClass(
      'flex',
      'flex-nowrap',
      'h-10',
      'w-full',
      'items-center',
    );
    expect(connectingStatus.closest('.cds--inline-loading')).toHaveStyle({
      inlineSize: 'max-content',
      whiteSpace: 'nowrap',
    });
    const cancelConnection = screen.getByRole('button', {
      name: 'Cancel voice connection',
    });
    expect(cancelConnection).not.toHaveTextContent('Cancel');
    expect(mockCarbonChatProps.input.isVisible).toBe(false);
    await act(async () => resolveVoiceConnection?.());
    expect(await screen.findByText('Listening')).toBeInTheDocument();
    await mockCarbonChatProps.messaging.customLoadHistory({
      messaging: {
        addMessage: mockAddMessage,
      },
      send: mockSend,
    });
    await act(async () => {
      mockAgentCallback.onUserMessage?.({
        id: 'voice-user-1',
        messageText: 'Where is my delivery?',
        completed: true,
      });
    });
    expect(mockSend).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 'voice-user-1',
        input: expect.objectContaining({ text: 'Where is my delivery?' }),
      }),
    );
    await act(async () => {
      await mockCarbonChatProps.messaging.customSendMessage(
        {
          id: 'voice-user-1',
          input: { message_type: 'text', text: 'Where is my delivery?' },
        },
        { signal: new AbortController().signal, silent: false },
        {
          messaging: { addMessage: mockAddMessage },
          send: mockSend,
        },
      );
    });
    expect(mockOnSendText).not.toHaveBeenCalled();
    mockVoiceAgent.messages = [
      {
        id: 'voice-response-1',
        role: 'system',
        messages: ['Your delivery is Friday.'],
      },
    ];
    await act(async () => {
      mockAgentCallback.onAssistantMessage?.({
        id: 'voice-response-1',
        messageText: '',
        completed: true,
      });
    });
    expect(mockAddMessage).toHaveBeenCalledWith(
      expect.objectContaining({ request_id: 'voice-user-1' }),
    );
    expect(screen.getByText('Built-in Microphone')).toBeInTheDocument();
    const microphoneDropdown = screen.getByRole('combobox', {
      name: 'Microphone',
    });
    expect(microphoneDropdown.closest('.cds--list-box')).toHaveClass(
      'cds--list-box--up',
    );
    const voiceControls = screen.getByRole('toolbar', {
      name: 'Voice conversation controls',
    });
    const muteButton = within(voiceControls).getByRole('button', {
      name: 'Mute microphone',
    });
    const switchToTextButton = within(voiceControls).getByRole('button', {
      name: 'Switch to text',
    });
    const endButton = within(voiceControls).getByRole('button', {
      name: 'End voice conversation',
    });
    expect(muteButton).not.toHaveTextContent('Mute microphone');
    expect(switchToTextButton).not.toHaveTextContent('Switch to text');
    expect(endButton).not.toHaveTextContent('End');

    await user.click(microphoneDropdown);
    await user.click(screen.getByRole('option', { name: 'USB Microphone' }));
    expect(mockSetActiveMediaDevice).toHaveBeenCalledWith('usb');

    await user.click(muteButton);
    expect(mockToggleMute).toHaveBeenCalledTimes(1);
    expect(screen.getByText('Microphone muted')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Unmute microphone' }),
    ).toBeInTheDocument();

    await user.click(switchToTextButton);

    expect(mockSetInputChannel).toHaveBeenCalledWith('text');
    expect(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    ).toBeInTheDocument();
    expect(mockCarbonChatProps.input.isVisible).toBe(true);
    expect(screen.queryByText('Listening')).not.toBeInTheDocument();
  });

  it('shows listening after adding voice to an existing text session', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });
    mockSetInputChannel.mockImplementation(async channel => {
      mockVoiceAgent.agentConfiguration.inputOptions.channel = channel;
      mockVoiceAgent.agentConfiguration.outputOptions.channel = channel;
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    const chatInstance = {
      messaging: { addMessage: mockAddMessage },
      send: mockSend,
    };
    await mockCarbonChatProps.messaging.customLoadHistory(chatInstance);
    await mockCarbonChatProps.messaging.customSendMessage(
      {
        id: 'text-request',
        input: { message_type: 'text', text: 'Start with text' },
      },
      { signal: new AbortController().signal, silent: false },
      chatInstance,
    );
    expect(mockVoiceAgent.isConnected).toBe(true);

    await user.click(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    );

    expect(await screen.findByText('Listening')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Cancel voice connection' }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'End voice conversation' }),
    ).toBeInTheDocument();
  });

  it('returns to text mode when voice connection setup emits an error', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });
    mockSetInputChannel.mockImplementationOnce(async channel => {
      mockVoiceAgent.agentConfiguration.inputOptions.channel = channel;
      mockVoiceAgent.agentConfiguration.outputOptions.channel = channel;
      mockAgentErrorHandler?.('client', 'Microphone permission denied');
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    );

    expect(
      await screen.findByText('Microphone permission denied'),
    ).toBeVisible();
    expect(screen.queryByText('Listening')).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    ).toBeInTheDocument();
    expect(mockVoiceAgent.agentConfiguration.inputOptions.channel).toBe('text');
    expect(mockVoiceAgent.agentConfiguration.outputOptions.channel).toBe(
      'text',
    );
  });

  it('does not correlate a retry response to a failed text request', async () => {
    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    const chatInstance = {
      messaging: { addMessage: mockAddMessage },
      send: mockSend,
    };
    await mockCarbonChatProps.messaging.customLoadHistory(chatInstance);
    await act(async () => {
      await mockCarbonChatProps.messaging.customSendMessage(
        {
          id: 'failed-request',
          input: { message_type: 'text', text: 'First attempt' },
        },
        { signal: new AbortController().signal, silent: false },
        chatInstance,
      );
      mockAgentErrorHandler?.('client', 'No connection available to send text');
      await mockCarbonChatProps.messaging.customSendMessage(
        {
          id: 'retry-request',
          input: { message_type: 'text', text: 'Retry' },
        },
        { signal: new AbortController().signal, silent: false },
        chatInstance,
      );
    });

    mockVoiceAgent.messages = [
      {
        id: 'retry-response',
        role: 'system',
        messages: ['Retry succeeded.'],
      },
    ];
    await act(async () => {
      mockAgentCallback.onAssistantMessage?.({
        id: 'retry-response',
        messageText: '',
        completed: true,
      });
    });

    expect(mockAddMessage).toHaveBeenLastCalledWith(
      expect.objectContaining({ request_id: 'retry-request' }),
    );
  });

  it('reconnects before the next text turn after editing an active argument', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    const chatInstance = {
      messaging: { addMessage: mockAddMessage },
      send: mockSend,
    };
    await mockCarbonChatProps.messaging.customLoadHistory(chatInstance);
    await mockCarbonChatProps.messaging.customSendMessage(
      {
        id: 'initial-request',
        input: { message_type: 'text', text: 'Initial turn' },
      },
      { signal: new AbortController().signal, silent: false },
      chatInstance,
    );
    await user.click(screen.getByRole('button', { name: 'Configuration' }));
    await user.click(
      screen.getByRole('button', { name: 'Arguments, 2 variables' }),
    );
    const customerName = screen.getByRole('textbox', {
      name: '{{customer_name}}',
    });
    await user.clear(customerName);
    await user.type(customerName, 'Avery');

    await mockCarbonChatProps.messaging.customSendMessage(
      {
        id: 'updated-request',
        input: { message_type: 'text', text: 'Updated turn' },
      },
      { signal: new AbortController().signal, silent: false },
      chatInstance,
    );

    expect(mockDisconnect).toHaveBeenCalledTimes(1);
    expect(mockDisconnect.mock.invocationCallOrder[0]).toBeLessThan(
      mockOnSendText.mock.invocationCallOrder[1],
    );
    expect(mockAddArgument).toHaveBeenLastCalledWith('customer_name', 'Avery');
  });

  it('finishes disconnecting when voice connection is cancelled in flight', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    );
    await user.click(
      screen.getByRole('button', { name: 'Cancel voice connection' }),
    );
    await act(async () => resolveVoiceConnection?.());

    await waitFor(() => expect(mockDisconnect).toHaveBeenCalledTimes(2));
    expect(screen.queryByText('Listening')).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    ).toBeInTheDocument();
  });

  it('waits for cancelled connection cleanup before starting voice again', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    );
    const finishCancelledConnection = resolveVoiceConnection;
    await user.click(
      screen.getByRole('button', { name: 'Cancel voice connection' }),
    );
    await user.click(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    );

    expect(mockSetInputChannel).toHaveBeenCalledTimes(1);
    await act(async () => finishCancelledConnection?.());
    await waitFor(() => expect(mockDisconnect).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(mockSetInputChannel).toHaveBeenCalledTimes(2));
    await act(async () => resolveVoiceConnection?.());
    expect(await screen.findByText('Listening')).toBeInTheDocument();
  });

  it('finishes disconnecting an in-flight voice connection on unmount', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });
    const { unmount } = render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    );
    unmount();
    await act(async () => resolveVoiceConnection?.());

    await waitFor(() => expect(mockDisconnect).toHaveBeenCalledTimes(2));
  });

  it('confirms before ending an active voice conversation', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    );
    await act(async () => resolveVoiceConnection?.());
    expect(await screen.findByText('Listening')).toBeInTheDocument();

    await user.click(
      screen.getByRole('button', { name: 'End voice conversation' }),
    );

    expect(
      screen.getByRole('heading', { name: 'End voice conversation?' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Listening')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    await waitFor(() =>
      expect(
        screen.queryByRole('heading', { name: 'End voice conversation?' }),
      ).not.toBeInTheDocument(),
    );
    expect(screen.getByText('Listening')).toBeInTheDocument();

    await user.click(
      screen.getByRole('button', { name: 'End voice conversation' }),
    );
    await user.click(screen.getByRole('button', { name: /End conversation/ }));

    expect(mockDisconnect).toHaveBeenCalledTimes(1);
    expect(
      screen.getByRole('button', { name: 'Start voice conversation' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Listening')).not.toBeInTheDocument();
  });

  it('returns to the assistant overview from the conversation header', async () => {
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
          <Route
            path="/deployment/assistant/:assistantId/overview"
            element={<div>Assistant overview</div>}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    const toolbar = screen.getByRole('toolbar', {
      name: 'Conversation toolbar',
    });
    await user.click(
      within(toolbar).getByRole('button', { name: 'Back to assistant' }),
    );

    expect(screen.getByText('Assistant overview')).toBeInTheDocument();
  });

  it('closes the overlay panel with Escape and restores toolbar focus', async () => {
    mockIsOverlayPanel = true;
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    const configurationButton = screen.getByRole('button', {
      name: 'Configuration',
    });
    await user.click(configurationButton);

    expect(
      screen.getByRole('dialog', { name: 'Session configuration' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('region', { name: 'Conversation' }),
    ).toHaveAttribute('inert');
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Close session configuration' }),
      ).toHaveFocus(),
    );

    await user.keyboard('{Escape}');

    expect(
      screen.queryByRole('dialog', { name: 'Session configuration' }),
    ).not.toBeInTheDocument();
    await waitFor(() => expect(configurationButton).toHaveFocus());
  });

  it('wraps overlay focus while Arguments is collapsed', async () => {
    mockIsOverlayPanel = true;
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(screen.getByRole('button', { name: 'Configuration' }));

    const dialog = screen.getByRole('dialog', {
      name: 'Session configuration',
    });
    const closeButton = within(dialog).getByRole('button', {
      name: 'Close session configuration',
    });
    const argumentsToggle = within(dialog).getByRole('button', {
      name: 'Arguments, 2 variables',
    });
    expect(argumentsToggle).toHaveAttribute('aria-expanded', 'false');
    await waitFor(() => expect(closeButton).toHaveFocus());

    await user.keyboard('{Shift>}{Tab}{/Shift}');
    expect(argumentsToggle).toHaveFocus();

    await user.keyboard('{Tab}');
    expect(closeButton).toHaveFocus();
  });

  it('wraps overlay focus from the final multiline argument', async () => {
    mockIsOverlayPanel = true;
    const user = userEvent.setup({
      pointerEventsCheck: PointerEventsCheckLevel.Never,
      skipHover: true,
    });

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    await user.click(screen.getByRole('button', { name: 'Configuration' }));
    const dialog = screen.getByRole('dialog', {
      name: 'Session configuration',
    });
    const closeButton = within(dialog).getByRole('button', {
      name: 'Close session configuration',
    });
    await user.click(
      within(dialog).getByRole('button', { name: 'Arguments, 2 variables' }),
    );
    const multilineArgument = within(dialog).getByRole('textbox', {
      name: '{{delivery_instructions}}',
    });
    multilineArgument.focus();

    await user.tab();

    expect(closeButton).toHaveFocus();
  });

  it('uses the Carbon dark theme when the application is dark', async () => {
    mockResolvedMode = 'dark';

    render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByRole('heading', { name: 'Delivery concierge' });
    expect(mockCarbonChatProps.injectCarbonTheme).toBe('g100');
    expect(mockCarbonChatProps.layout.customProperties).toMatchObject({
      '$chat-shell-background': '#161616',
      '$chat-prompt-background': '#262626',
      '$ai-aura-start-sm': '#161616',
      '$ai-aura-end': '#161616',
    });
  });

  it('surfaces assistant loading failures and disconnects on cleanup', async () => {
    mockGetAssistant.mockRejectedValueOnce(new Error('Unavailable'));

    const { unmount } = render(
      <MemoryRouter initialEntries={['/preview/v2/chat/assistant-123']}>
        <Routes>
          <Route
            path="/preview/v2/chat/:assistantId"
            element={<PreviewVoiceAgentV2 />}
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(
      await screen.findByText('Unable to load assistant configuration.'),
    ).toBeInTheDocument();

    unmount();
    await waitFor(() => expect(mockDisconnect).toHaveBeenCalledTimes(1));
  });

  it('redirects to not found when the assistant id is missing', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<PreviewVoiceAgentV2 />} />
          <Route path="/404" element={<div>Not found</div>} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText('Not found')).toBeInTheDocument();
  });
});

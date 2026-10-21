import { render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { getAssistantByIdWithApi } from '@/clients/assistant.client';
import { PreviewPhoneAgent } from '../index';

jest.mock('@/clients/assistant.client', () => ({
  getAssistantByIdWithApi: jest.fn(),
}));

jest.mock('@/clients/runtime.client', () => ({
  createPreviewPhoneCall: jest.fn(),
  createVoiceAgentDebuggerConnection: jest.fn(),
  createVoiceAgentSDKConnection: jest.fn(),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    token: 'token-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/app/pages/preview-agent/voice-agent/voice-agent', () => ({
  ArgumentList: () => <div>Argument fields</div>,
  ConfigEmpty: ({ label }: { label: string }) => <div>{label}</div>,
  ConfigBlock: ({
    title,
    children,
  }: {
    title: string;
    children: ReactNode;
  }) => (
    <section>
      <h3>{title}</h3>
      {children}
    </section>
  ),
  DebuggerTabHeader: ({ title }: { title: string }) => <h3>{title}</h3>,
  InfoRow: ({ label, value }: { label: string; value: string }) => (
    <div>
      {label}: {value}
    </div>
  ),
  PreviewAgentHeader: () => <header>Product header</header>,
  VoiceAgent: () => null,
}));

const audioProvider = {
  getAudioprovider: () => 'Deepgram',
  getAudiooptionsList: () => [],
};

const model = {
  getModelprovidername: () => 'OpenAI',
  getAssistantmodeloptionsList: () => [],
  getTemplate: () => undefined,
};

const assistant = {
  getName: () => 'Delivery concierge',
  getDescription: () => 'Coordinates delivery calls.',
  getAssistantprovidermodel: () => model,
  getPhonedeployment: () => ({
    getPhoneprovidername: () => 'Twilio',
    getInputaudio: () => audioProvider,
    getOutputaudio: () => audioProvider,
  }),
};

const mockedGetAssistantByIdWithApi =
  getAssistantByIdWithApi as jest.MockedFunction<
    typeof getAssistantByIdWithApi
  >;

describe('PreviewPhoneAgent', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
    mockedGetAssistantByIdWithApi.mockResolvedValue({
      getSuccess: () => true,
      getData: () => assistant,
    } as any);
  });

  it('keeps the call task primary and opens configuration on demand', async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter
        initialEntries={['/deployment/assistant/assistant-1/playground/phone']}
      >
        <Routes>
          <Route
            path="/deployment/assistant/:assistantId/playground/phone"
            element={<PreviewPhoneAgent embedded />}
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole('heading', { name: 'Debug Phone Call' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Product header')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Session configuration')).toBeNull();

    await user.click(screen.getByRole('button', { name: 'Configuration' }));

    expect(screen.getByLabelText('Session configuration')).toBeVisible();
    expect(screen.getByText(/telephony: Twilio/)).toBeVisible();
    expect(screen.getByText('Arguments')).toBeVisible();

    await user.click(
      screen.getByRole('button', { name: 'Close configuration' }),
    );
    await waitFor(() =>
      expect(screen.queryByLabelText('Session configuration')).toBeNull(),
    );
  });
});

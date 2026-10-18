import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import toast from 'react-hot-toast/headless';
import { DeleteProviderKey } from '@rapidaai/react';

import { ViewProviderCredentialDialog } from '../view-provider-credential-modal';

type MockCredential = {
  getId: () => string;
  getName: () => string;
  getProvider: () => string;
  getCreateddate: () => undefined;
  getLastuseddate: () => undefined;
};

const makeCredential = (
  id: string,
  name: string,
  provider: string,
): MockCredential => ({
  getId: () => id,
  getName: () => name,
  getProvider: () => provider,
  getCreateddate: () => undefined,
  getLastuseddate: () => undefined,
});

let mockProviderCredentials: MockCredential[] = [];
const mockHideLoader = jest.fn();
const mockReloadProviderCredentials = jest.fn();
const mockShowLoader = jest.fn();

jest.mock('@rapidaai/react', () => ({
  ConnectionConfig: { WithDebugger: jest.fn((headers: unknown) => headers) },
  DeleteProviderKey: jest.fn(),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    projectId: 'project-1',
    token: 'token-1',
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-model', () => ({
  useAllProviderCredentials: () => ({
    providerCredentials: mockProviderCredentials,
  }),
}));

jest.mock('@/context/provider-context', () => ({
  useProviderContext: () => ({
    reloadProviderCredentials: mockReloadProviderCredentials,
  }),
}));

jest.mock('@/configs', () => ({
  connectionConfig: {},
}));

jest.mock('@/utils/date', () => ({
  toHumanReadableRelativeTime: jest.fn(() => 'recently'),
}));

jest.mock('@/app/components/ui/primitives/modal', () => ({
  Modal: ({ open, children }: any) => (open ? <div>{children}</div> : null),
  ModalHeader: ({ label, onClose, title }: any) => (
    <header>
      <span>{label}</span>
      <h2>{title}</h2>
      <button type="button" onClick={onClose}>
        Header close
      </button>
    </header>
  ),
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalFooter: ({ children }: any) => <footer>{children}</footer>,
}));

jest.mock('@/app/components/ui/primitives/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  TertiaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  DangerButton: ({ children, renderIcon, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  IconOnlyButton: ({ iconDescription, renderIcon, ...props }: any) => (
    <button aria-label={iconDescription} {...props} />
  ),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  EmptyState: ({ action, onAction, subtitle, title }: any) => (
    <div role="status">
      <h3>{title}</h3>
      <p>{subtitle}</p>
      <button type="button" onClick={onAction}>
        {action}
      </button>
    </div>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  Key: () => <svg data-testid="key-icon" />,
  TrashCan: () => <svg data-testid="trash-icon" />,
}));

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
}));

describe('ViewProviderCredentialDialog', () => {
  beforeEach(() => {
    mockProviderCredentials = [];
    jest.clearAllMocks();
  });

  it('renders the credential ID and copies it from the accessible action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const writeText = jest.spyOn(navigator.clipboard, 'writeText');
    mockProviderCredentials = [
      makeCredential('credential-openai-123', 'Production key', 'openai'),
    ];

    render(
      <ViewProviderCredentialDialog
        modalOpen
        setModalOpen={jest.fn()}
        currentProvider={{
          code: 'openai',
          name: 'OpenAI',
          image: '/providers/openai.svg',
          featureList: ['llm'],
        }}
        onSetupCredential={jest.fn()}
      />,
    );

    expect(
      await screen.findByText('credential-openai-123'),
    ).toBeInTheDocument();

    await user.click(
      screen.getByRole('button', { name: 'Copy credential ID' }),
    );

    expect(writeText).toHaveBeenCalledWith('credential-openai-123');
  });

  it('filters credentials that belong to a different provider', async () => {
    mockProviderCredentials = [
      makeCredential('credential-openai-123', 'OpenAI key', 'openai'),
      makeCredential(
        'credential-elevenlabs-456',
        'ElevenLabs key',
        'elevenlabs',
      ),
    ];

    render(
      <ViewProviderCredentialDialog
        modalOpen
        setModalOpen={jest.fn()}
        currentProvider={{
          code: 'openai',
          name: 'OpenAI',
          image: '/providers/openai.svg',
          featureList: ['llm'],
        }}
        onSetupCredential={jest.fn()}
      />,
    );

    expect(
      await screen.findByText('credential-openai-123'),
    ).toBeInTheDocument();
    expect(
      screen.queryByText('credential-elevenlabs-456'),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('ElevenLabs key')).not.toBeInTheDocument();
  });

  it('opens the setup flow from the empty state', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onSetupCredential = jest.fn();

    render(
      <ViewProviderCredentialDialog
        modalOpen
        setModalOpen={jest.fn()}
        currentProvider={{
          code: 'openai',
          name: 'OpenAI',
          image: '/providers/openai.svg',
          featureList: ['llm'],
        }}
        onSetupCredential={onSetupCredential}
      />,
    );

    expect(
      screen.getByText('No provider credential to display'),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Setup Credential' }));

    expect(onSetupCredential).toHaveBeenCalledTimes(1);
  });

  it('deletes credentials and reloads on success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockProviderCredentials = [
      makeCredential('credential-openai-123', 'Production key', 'openai'),
    ];
    (DeleteProviderKey as jest.Mock).mockImplementation(
      (_config, _credentialId, callback) => {
        callback(null, { getSuccess: () => true });
      },
    );

    render(
      <ViewProviderCredentialDialog
        modalOpen
        setModalOpen={jest.fn()}
        currentProvider={{
          code: 'openai',
          name: 'OpenAI',
          image: '/providers/openai.svg',
          featureList: ['llm'],
        }}
        onSetupCredential={jest.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Delete' }));

    expect(mockShowLoader).toHaveBeenCalledTimes(1);
    expect(DeleteProviderKey).toHaveBeenCalledWith(
      {},
      'credential-openai-123',
      expect.any(Function),
      {
        authorization: 'token-1',
        projectId: 'project-1',
        userId: 'user-1',
      },
    );
    expect(mockHideLoader).toHaveBeenCalledTimes(1);
    expect(mockReloadProviderCredentials).toHaveBeenCalledTimes(1);
  });

  it('reports delete errors from the API callback', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockProviderCredentials = [
      makeCredential('credential-openai-123', 'Production key', 'openai'),
    ];
    (DeleteProviderKey as jest.Mock).mockImplementation(
      (_config, _credentialId, callback) => {
        callback(null, {
          getError: () => ({ getHumanmessage: () => 'Delete failed' }),
          getSuccess: () => false,
        });
      },
    );

    render(
      <ViewProviderCredentialDialog
        modalOpen
        setModalOpen={jest.fn()}
        currentProvider={{
          code: 'openai',
          name: 'OpenAI',
          image: '',
          featureList: ['llm'],
        }}
        onSetupCredential={jest.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Delete' }));

    expect(toast.error).toHaveBeenCalledWith('Delete failed');
  });

  it('reports the fallback delete error when no API message exists', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockProviderCredentials = [
      makeCredential('credential-openai-123', 'Production key', 'openai'),
    ];
    (DeleteProviderKey as jest.Mock).mockImplementation(
      (_config, _credentialId, callback) => {
        callback(null, {
          getError: () => undefined,
          getSuccess: () => false,
        });
      },
    );

    render(
      <ViewProviderCredentialDialog
        modalOpen
        setModalOpen={jest.fn()}
        currentProvider={{
          code: 'openai',
          name: 'OpenAI',
          image: '/providers/openai.svg',
          featureList: ['llm'],
        }}
        onSetupCredential={jest.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Delete' }));

    expect(toast.error).toHaveBeenCalledWith(
      'Unable to process your request. please try again later.',
    );
  });

  it('closes from the header and footer actions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();

    render(
      <ViewProviderCredentialDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentProvider={{
          code: 'openai',
          name: 'OpenAI',
          image: '/providers/openai.svg',
          featureList: ['llm'],
        }}
        onSetupCredential={jest.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Header close' }));
    await user.click(screen.getByRole('button', { name: 'Got it' }));

    expect(setModalOpen).toHaveBeenCalledWith(false);
    expect(setModalOpen).toHaveBeenCalledTimes(2);
  });
});

import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { ViewProviderCredentialDialog } from '../index';

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
const mockWriteText = jest.fn();

jest.mock('@rapidaai/react', () => ({
  ConnectionConfig: { WithDebugger: jest.fn() },
  DeleteProviderKey: jest.fn(),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    projectId: 'project-1',
    token: 'token-1',
  }),
}));

jest.mock('@/hooks', () => ({
  useRapidaStore: () => ({
    showLoader: jest.fn(),
    hideLoader: jest.fn(),
  }),
}));

jest.mock('@/hooks/use-model', () => ({
  useAllProviderCredentials: () => ({
    providerCredentials: mockProviderCredentials,
  }),
}));

jest.mock('@/context/provider-context', () => ({
  useProviderContext: () => ({
    reloadProviderCredentials: jest.fn(),
  }),
}));

jest.mock('@/configs', () => ({
  connectionConfig: {},
}));

jest.mock('@/utils/date', () => ({
  toHumanReadableRelativeTime: jest.fn(() => 'recently'),
}));

jest.mock('@/app/components/carbon/modal', () => ({
  Modal: ({ open, children }: any) => (open ? <div>{children}</div> : null),
  ModalHeader: ({ label, title }: any) => (
    <header>
      <span>{label}</span>
      <h2>{title}</h2>
    </header>
  ),
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalFooter: ({ children }: any) => <footer>{children}</footer>,
}));

jest.mock('@/app/components/carbon/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/carbon/button', () => ({
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

describe('ViewProviderCredentialDialog', () => {
  beforeEach(() => {
    mockProviderCredentials = [];
    mockWriteText.mockReset();
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: mockWriteText },
    });
  });

  it('renders the credential ID and copies it from the accessible action', async () => {
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

    fireEvent.click(screen.getByRole('button', { name: 'Copy credential ID' }));

    expect(mockWriteText).toHaveBeenCalledWith('credential-openai-123');
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
});

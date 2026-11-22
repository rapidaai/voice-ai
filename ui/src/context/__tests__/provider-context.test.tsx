import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import {
  ProviderContextProvider,
  useProviderContext,
} from '@/context/provider-context';
import { listOrganizationCredentials } from '@/clients';

let mockAuthId = 'user-1';
let mockProjectId = 'project-1';
let mockToken = 'token-1';

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: mockAuthId,
    projectId: mockProjectId,
    token: mockToken,
  }),
}));

jest.mock('@/clients', () => ({
  listOrganizationCredentials: jest.fn(),
}));

const mockListOrganizationCredentials = jest.mocked(
  listOrganizationCredentials,
);

function ProviderCredentialsProbe() {
  const { providerCredentials } = useProviderContext();
  return (
    <div>
      {providerCredentials.map(credential => (
        <span key={credential.getId()}>{credential.getId()}</span>
      ))}
    </div>
  );
}

function successfulResponse(id: string) {
  return {
    getSuccess: () => true,
    getDataList: () => [
      { getId: () => id, serializeBinary: () => new Uint8Array([1]) },
    ],
  };
}

describe('ProviderContextProvider', () => {
  beforeEach(() => {
    mockAuthId = 'user-1';
    mockProjectId = 'project-1';
    mockToken = 'token-1';
    mockListOrganizationCredentials.mockReset();
    window.localStorage.clear();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('keeps fetched provider credentials in memory without accessing local storage', async () => {
    let callback: ((error: null, response: any) => void) | undefined;
    const getItem = jest.spyOn(Storage.prototype, 'getItem');
    const setItem = jest.spyOn(Storage.prototype, 'setItem');
    const removeItem = jest.spyOn(Storage.prototype, 'removeItem');
    mockListOrganizationCredentials.mockImplementation(options => {
      callback = options.callback;
    });

    render(
      <ProviderContextProvider>
        <ProviderCredentialsProbe />
      </ProviderContextProvider>,
    );

    await waitFor(() =>
      expect(mockListOrganizationCredentials).toHaveBeenCalledTimes(1),
    );
    act(() => callback?.(null, successfulResponse('credential-1')));

    expect(screen.getByText('credential-1')).toBeVisible();
    expect(getItem).not.toHaveBeenCalled();
    expect(setItem).not.toHaveBeenCalled();
    expect(removeItem).not.toHaveBeenCalled();
  });

  it('clears credentials and ignores an earlier project response', async () => {
    const callbacks: Array<(error: null, response: any) => void> = [];
    mockListOrganizationCredentials.mockImplementation(options => {
      callbacks.push(options.callback);
    });

    const view = render(
      <ProviderContextProvider>
        <ProviderCredentialsProbe />
      </ProviderContextProvider>,
    );
    await waitFor(() => expect(callbacks).toHaveLength(1));
    act(() => callbacks[0](null, successfulResponse('project-1-credential')));
    expect(screen.getByText('project-1-credential')).toBeVisible();

    mockProjectId = 'project-2';
    view.rerender(
      <ProviderContextProvider>
        <ProviderCredentialsProbe />
      </ProviderContextProvider>,
    );

    await waitFor(() => expect(callbacks).toHaveLength(2));
    expect(screen.queryByText('project-1-credential')).not.toBeInTheDocument();

    act(() => callbacks[1](null, successfulResponse('project-2-credential')));
    expect(screen.getByText('project-2-credential')).toBeVisible();

    act(() => callbacks[0](null, successfulResponse('stale-credential')));
    expect(screen.getByText('project-2-credential')).toBeVisible();
    expect(screen.queryByText('stale-credential')).not.toBeInTheDocument();
  });
});

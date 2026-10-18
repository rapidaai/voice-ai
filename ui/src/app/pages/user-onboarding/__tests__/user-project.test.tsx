import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { CreateProjectPage } from '@/app/pages/user-onboarding/user-project';
import { AuthContext } from '@/context/auth-context';
import { CreateProject } from '@rapidaai/react';

const mockNavigate = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

let mockCredential = {
  user: { name: 'John' },
  authId: 'auth-1',
  token: 'token-1',
};

jest.mock('@rapidaai/react', () => ({
  ConnectionConfig: class ConnectionConfig {},
  CreateProject: jest.fn(),
}));

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => mockCredential,
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/app/components/ui/primitives/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: require('react').forwardRef(
    ({ labelText, helperText, ...props }: any, ref: any) => (
      <input ref={ref} {...props} />
    ),
  ),
  TextArea: require('react').forwardRef(
    ({ labelText, helperText, ...props }: any, ref: any) => (
      <textarea ref={ref} {...props} />
    ),
  ),
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({ children, isLoading, renderIcon, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@/app/components/ui/feedback/notification', () => ({
  Notification: ({ subtitle }: any) => <div>{subtitle}</div>,
}));

const renderWithAuth = (authorize = jest.fn()) =>
  render(
    <AuthContext.Provider value={{ authorize } as any}>
      <CreateProjectPage />
    </AuthContext.Provider>,
  );

describe('CreateProjectPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockCredential = {
      user: { name: 'John' },
      authId: 'auth-1',
      token: 'token-1',
    };
  });

  it('submits and navigates to dashboard on success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const authorize = jest.fn(success => success());
    (CreateProject as jest.Mock).mockImplementation(
      (_cfg, _name, _description, _headers, callback) => {
        callback(null, { getSuccess: () => true });
      },
    );

    renderWithAuth(authorize);

    await user.click(
      screen.getByPlaceholderText('eg: Acme Support Operations'),
    );
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support Bot'));
    await user.click(
      screen.getByPlaceholderText(
        'eg: White-label inbound voice agents for healthcare support across US and UK',
      ),
    );
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Project description'));
    await user.click(screen.getByRole('button', { name: 'Go to dashboard' }));

    await waitFor(() => {
      expect(CreateProject).toHaveBeenCalled();
    });

    expect(mockShowLoader).toHaveBeenCalledWith('overlay');
    expect(mockHideLoader).toHaveBeenCalled();
    expect(authorize).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith('/dashboard');
  });

  it('shows generic error when API returns service error', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateProject as jest.Mock).mockImplementation(
      (_cfg, _name, _description, _headers, callback) => {
        callback(new Error('boom'), null);
      },
    );

    renderWithAuth();

    await user.click(
      screen.getByPlaceholderText('eg: Acme Support Operations'),
    );
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support Bot'));
    await user.click(screen.getByRole('button', { name: 'Go to dashboard' }));

    expect(
      await screen.findByText(
        'Unable to process your request. Please try again later.',
      ),
    ).toBeInTheDocument();
  });

  it('shows project details error when API returns unsuccessful response', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateProject as jest.Mock).mockImplementation(
      (_cfg, _name, _description, _headers, callback) => {
        callback(null, { getSuccess: () => false });
      },
    );

    renderWithAuth();

    await user.click(
      screen.getByPlaceholderText('eg: Acme Support Operations'),
    );
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support Bot'));
    await user.click(screen.getByRole('button', { name: 'Go to dashboard' }));

    expect(
      await screen.findByText(
        'Unable to create project. Please check the details.',
      ),
    ).toBeInTheDocument();
  });
});

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { ChangePasswordPage } from '@/app/pages/authentication/change-password';
import { CreatePassword } from '@rapidaai/react';

const mockNavigate = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

let mockParams: Record<string, string | undefined> = {};

jest.mock('@rapidaai/react', () => ({
  ConnectionConfig: class ConnectionConfig {},
  CreatePassword: jest.fn(),
}));

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
  useParams: () => mockParams,
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/app/components/ui/primitives/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: (props: any) => <input {...props} />,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({ children, isLoading, renderIcon, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@/app/components/ui/feedback/notification', () => ({
  Notification: ({ subtitle }: any) => <div>{subtitle}</div>,
}));

jest.mock('@carbon/react', () => ({
  PasswordInput: require('react').forwardRef(
    ({ labelText, ...props }: any, ref: any) => <input ref={ref} {...props} />,
  ),
}));

describe('ChangePasswordPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockParams = {};
  });

  it('shows token expiry error when token is missing', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<ChangePasswordPage />);

    expect(
      screen.getByRole('heading', { level: 1, name: 'Set a new password' }),
    ).toBeInTheDocument();

    await user.click(screen.getAllByPlaceholderText('********')[0]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getAllByPlaceholderText('********')[1]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(
      await screen.findByText(
        'The password token is expired, please request again for reset password token.',
      ),
    ).toBeInTheDocument();
    expect(CreatePassword).not.toHaveBeenCalled();
    expect(mockShowLoader).not.toHaveBeenCalled();
  });

  it('shows mismatch error when passwords do not match', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockParams = { token: 'token-1' };
    render(<ChangePasswordPage />);

    await user.click(screen.getAllByPlaceholderText('********')[0]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret-1'));
    await user.click(screen.getAllByPlaceholderText('********')[1]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret-2'));
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(
      await screen.findByText(
        'Passwords entered do not match, please check and try again.',
      ),
    ).toBeInTheDocument();
    expect(CreatePassword).not.toHaveBeenCalled();
  });

  it('submits and navigates to sign in on success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockParams = { token: 'token-1' };
    (CreatePassword as jest.Mock).mockImplementation(
      (_cfg, _token, _password, callback) => {
        callback(null, { getSuccess: () => true });
      },
    );

    render(<ChangePasswordPage />);

    await user.click(screen.getAllByPlaceholderText('********')[0]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getAllByPlaceholderText('********')[1]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    await waitFor(() => {
      expect(CreatePassword).toHaveBeenCalled();
    });
    expect(mockShowLoader).toHaveBeenCalled();
    expect(mockHideLoader).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith('/auth/signin');
  });

  it('shows fallback error when API callback returns service error', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockParams = { token: 'token-1' };
    (CreatePassword as jest.Mock).mockImplementation(
      (_cfg, _token, _password, callback) => {
        callback(new Error('boom'), null);
      },
    );

    render(<ChangePasswordPage />);

    await user.click(screen.getAllByPlaceholderText('********')[0]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getAllByPlaceholderText('********')[1]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(
      await screen.findByText(
        'Unable to process your request. Please try again later.',
      ),
    ).toBeInTheDocument();
    expect(mockHideLoader).toHaveBeenCalled();
  });

  it('shows human message when API returns unsuccessful response with error', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockParams = { token: 'token-1' };
    (CreatePassword as jest.Mock).mockImplementation(
      (_cfg, _token, _password, callback) => {
        callback(null, {
          getSuccess: () => false,
          getError: () => ({ getHumanmessage: () => 'Token invalid' }),
        });
      },
    );

    render(<ChangePasswordPage />);

    await user.click(screen.getAllByPlaceholderText('********')[0]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getAllByPlaceholderText('********')[1]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(await screen.findByText('Token invalid')).toBeInTheDocument();
  });

  it('shows fallback error when API returns unsuccessful response without error details', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockParams = { token: 'token-1' };
    (CreatePassword as jest.Mock).mockImplementation(
      (_cfg, _token, _password, callback) => {
        callback(null, {
          getSuccess: () => false,
          getError: () => null,
        });
      },
    );

    render(<ChangePasswordPage />);

    await user.click(screen.getAllByPlaceholderText('********')[0]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getAllByPlaceholderText('********')[1]);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(
      await screen.findByText(
        'Unable to process your request. Please try again later.',
      ),
    ).toBeInTheDocument();
  });
});

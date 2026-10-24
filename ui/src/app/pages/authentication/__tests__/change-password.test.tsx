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
  FluidPasswordInput: require('react').forwardRef(
    ({ id, invalid, invalidText, labelText, ...props }: any, ref: any) => (
      <div data-fluid-input>
        <label htmlFor={id}>{labelText}</label>
        <input id={id} ref={ref} aria-invalid={invalid} {...props} />
        {invalid && <span>{invalidText}</span>}
      </div>
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

jest.mock('@carbon/react', () => ({
  FluidForm: ({ children, ...props }: any) => (
    <form {...props}>{children}</form>
  ),
  Link: ({ children, href }: any) => <a href={href}>{children}</a>,
  PasswordInput: require('react').forwardRef(
    ({ id, invalid, invalidText, labelText, ...props }: any, ref: any) => (
      <div>
        <label htmlFor={id}>{labelText}</label>
        <input id={id} ref={ref} aria-invalid={invalid} {...props} />
        {invalid && <span>{invalidText}</span>}
      </div>
    ),
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
    ).toHaveClass('text-[2rem]', 'leading-[2.5rem]', 'font-normal');
    expect(screen.getByTestId('change-password-panel')).toHaveClass(
      'max-w-[37rem]',
      'bg-surface',
    );
    expect(screen.getByTestId('change-password-header')).toHaveClass(
      'p-6',
      'sm:p-8',
    );
    expect(screen.getByTestId('change-password-form')).toHaveClass('p-0');
    expect(screen.getByTestId('change-password-input-stack')).toHaveClass(
      'mx-6',
      'sm:mx-8',
      'flex-col',
    );
    expect(screen.getByTestId('change-password-actions')).toHaveClass(
      'p-6',
      'sm:p-8',
    );
    expect(screen.getByLabelText('New password').parentElement).toHaveAttribute(
      'data-fluid-input',
    );
    expect(
      screen.getByLabelText('Confirm password').parentElement,
    ).toHaveAttribute('data-fluid-input');

    await user.type(screen.getByLabelText('New password'), 'secret');
    await user.type(screen.getByLabelText('Confirm password'), 'secret');
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(
      await screen.findByText(
        'This reset link has expired. Request a new reset link.',
      ),
    ).toBeInTheDocument();
    expect(CreatePassword).not.toHaveBeenCalled();
    expect(mockShowLoader).not.toHaveBeenCalled();
  });

  it('shows mismatch error when passwords do not match', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockParams = { token: 'token-1' };
    render(<ChangePasswordPage />);

    await user.type(screen.getByLabelText('New password'), 'secret-1');
    await user.type(screen.getByLabelText('Confirm password'), 'secret-2');
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(
      await screen.findByText('Passwords do not match.'),
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

    await user.type(screen.getByLabelText('New password'), 'secret');
    await user.type(screen.getByLabelText('Confirm password'), 'secret');
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

    await user.type(screen.getByLabelText('New password'), 'secret');
    await user.type(screen.getByLabelText('Confirm password'), 'secret');
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(
      await screen.findByText("We couldn't reset your password. Try again."),
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

    await user.type(screen.getByLabelText('New password'), 'secret');
    await user.type(screen.getByLabelText('Confirm password'), 'secret');
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

    await user.type(screen.getByLabelText('New password'), 'secret');
    await user.type(screen.getByLabelText('Confirm password'), 'secret');
    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(
      await screen.findByText("We couldn't reset your password. Try again."),
    ).toBeInTheDocument();
  });

  it('shows required field errors before checking the token', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockParams = { token: 'token-1' };
    render(<ChangePasswordPage />);

    await user.click(screen.getByRole('button', { name: 'Save new password' }));

    expect(await screen.findByText('Enter a new password.')).toBeVisible();
    expect(screen.getByText('Confirm your new password.')).toBeVisible();
    expect(CreatePassword).not.toHaveBeenCalled();
  });
});

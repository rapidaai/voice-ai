import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import userEvent from '@testing-library/user-event';

import { CreateOrganizationPage } from '@/app/pages/user-onboarding/user-organization';
import { AuthContext } from '@/context/auth-context';
import { CreateOrganization } from '@rapidaai/react';

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
  CreateOrganization: jest.fn(),
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
  Select: require('react').forwardRef(
    ({ labelText, helperText, children, ...props }: any, ref: any) => (
      <select ref={ref} {...props}>
        {children}
      </select>
    ),
  ),
  SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
}));

const renderWithAuth = (authorize = jest.fn()) =>
  render(
    <AuthContext.Provider value={{ authorize } as any}>
      <CreateOrganizationPage />
    </AuthContext.Provider>,
  );

describe('CreateOrganizationPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockCredential = {
      user: { name: 'John' },
      authId: 'auth-1',
      token: 'token-1',
    };
  });

  it('blocks submission when the required industry is missing', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithAuth();

    const industry = screen.getByPlaceholderText(
      'eg: Agency services, healthcare, finance',
    );
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(industry).toBeInvalid();
    expect(CreateOrganization).not.toHaveBeenCalled();
  });

  it('submits and navigates to onboarding project on success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const authorize = jest.fn(success => success());
    (CreateOrganization as jest.Mock).mockImplementation(
      (_cfg, _name, _size, _industry, _headers, callback) => {
        callback(null, { getSuccess: () => true });
      },
    );

    renderWithAuth(authorize);

    const organizationName = screen.getByPlaceholderText(
      'eg: Acme Voice Studio',
    );
    await user.click(organizationName);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste('My Org');
    await user.type(
      screen.getByPlaceholderText('eg: Agency services, healthcare, finance'),
      'Software',
    );
    await user.selectOptions(screen.getByRole('combobox'), 'agency');
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    await waitFor(() => {
      expect(CreateOrganization).toHaveBeenCalled();
    });

    expect(mockShowLoader).toHaveBeenCalledWith('overlay');
    expect(authorize).toHaveBeenCalled();
    expect(mockHideLoader).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith('/onboarding/project');
  });

  it('shows generic error when API returns service error', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateOrganization as jest.Mock).mockImplementation(
      (_cfg, _name, _size, _industry, _headers, callback) => {
        callback(new Error('boom'), null);
      },
    );

    renderWithAuth();

    await user.type(
      screen.getByPlaceholderText('eg: Agency services, healthcare, finance'),
      'Software',
    );
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(
      await screen.findByText(
        'Unable to process your request. Please try again later.',
      ),
    ).toBeInTheDocument();
  });

  it('shows credential error when API returns unsuccessful response', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateOrganization as jest.Mock).mockImplementation(
      (_cfg, _name, _size, _industry, _headers, callback) => {
        callback(null, { getSuccess: () => false });
      },
    );

    renderWithAuth();

    await user.type(
      screen.getByPlaceholderText('eg: Agency services, healthcare, finance'),
      'Software',
    );
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(
      await screen.findByText('Please provide valid credentials to sign in.'),
    ).toBeInTheDocument();
  });
});

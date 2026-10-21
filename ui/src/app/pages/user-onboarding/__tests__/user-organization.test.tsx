import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
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
    (
      { labelText, helperText, invalid, invalidText, ...props }: any,
      ref: any,
    ) => (
      <label>
        {labelText}
        <input ref={ref} aria-invalid={invalid} {...props} />
        {invalid ? <span>{invalidText}</span> : null}
      </label>
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
    (
      { labelText, helperText, children, invalid, invalidText, ...props }: any,
      ref: any,
    ) => (
      <label>
        {labelText}
        <select ref={ref} aria-invalid={invalid} {...props}>
          {children}
        </select>
        {invalid ? <span>{invalidText}</span> : null}
      </label>
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
      'e.g., Agency services, healthcare, finance',
    );
    await user.clear(screen.getByLabelText('Organization name'));
    await user.type(screen.getByLabelText('Organization name'), '   ');
    await user.type(industry, '   ');
    await user.click(
      screen.getByRole('button', { name: 'Create organization' }),
    );

    expect(
      await screen.findByText('Enter an organization name.'),
    ).toBeInTheDocument();
    expect(industry).toBeInvalid();
    expect(
      await screen.findByText('Enter your organization industry.'),
    ).toBeInTheDocument();
    expect(CreateOrganization).not.toHaveBeenCalled();
  });

  it('requires an operating model before submission', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithAuth();

    await user.type(
      screen.getByPlaceholderText('e.g., Agency services, healthcare, finance'),
      'Software',
    );
    await user.click(
      screen.getByRole('button', { name: 'Create organization' }),
    );

    expect(
      await screen.findByText('Select an operating model.'),
    ).toBeInTheDocument();
    expect(CreateOrganization).not.toHaveBeenCalled();
  });

  it('leaves the organization name empty when the profile has no name', () => {
    mockCredential = {
      user: {} as any,
      authId: 'auth-1',
      token: 'token-1',
    };

    renderWithAuth();

    expect(screen.getByLabelText('Organization name')).toHaveValue('');
  });

  it('submits and navigates to onboarding project on success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    let finishAuthorization: (() => void) | undefined;
    const authorize = jest.fn(success => {
      finishAuthorization = success;
    });
    (CreateOrganization as jest.Mock).mockImplementation(
      (_cfg, _name, _size, _industry, _headers, callback) => {
        callback(null, { getSuccess: () => true });
      },
    );

    renderWithAuth(authorize);

    const organizationName = screen.getByPlaceholderText(
      'e.g., Acme Voice Studio',
    );
    await user.click(organizationName);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste('My Org');
    await user.type(
      screen.getByPlaceholderText('e.g., Agency services, healthcare, finance'),
      'Software',
    );
    await user.selectOptions(screen.getByRole('combobox'), 'agency');
    await user.click(
      screen.getByRole('button', { name: 'Create organization' }),
    );

    await waitFor(() => {
      expect(CreateOrganization).toHaveBeenCalled();
    });

    expect(mockShowLoader).toHaveBeenCalledWith('overlay');
    expect(authorize).toHaveBeenCalled();
    expect(mockHideLoader).not.toHaveBeenCalled();

    act(() => finishAuthorization?.());

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
      screen.getByPlaceholderText('e.g., Agency services, healthcare, finance'),
      'Software',
    );
    await user.selectOptions(screen.getByRole('combobox'), 'agency');
    await user.click(
      screen.getByRole('button', { name: 'Create organization' }),
    );

    expect(
      await screen.findByText(
        'Unable to create your organization. Please try again.',
      ),
    ).toBeInTheDocument();

    await user.type(screen.getByLabelText('Organization name'), ' updated');
    expect(
      screen.queryByText(
        'Unable to create your organization. Please try again.',
      ),
    ).not.toBeInTheDocument();
  });

  it('shows the actionable API error when creation is unsuccessful', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateOrganization as jest.Mock).mockImplementation(
      (_cfg, _name, _size, _industry, _headers, callback) => {
        callback(null, {
          getSuccess: () => false,
          getError: () => ({
            getHumanmessage: () => 'An organization with this name exists.',
          }),
        });
      },
    );

    renderWithAuth();

    await user.type(
      screen.getByPlaceholderText('e.g., Agency services, healthcare, finance'),
      'Software',
    );
    await user.selectOptions(screen.getByRole('combobox'), 'agency');
    await user.click(
      screen.getByRole('button', { name: 'Create organization' }),
    );

    expect(
      await screen.findByText('An organization with this name exists.'),
    ).toBeInTheDocument();
  });
});

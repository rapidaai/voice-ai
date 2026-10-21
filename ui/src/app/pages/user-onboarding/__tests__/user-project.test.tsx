import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
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
    let finishAuthorization: (() => void) | undefined;
    const authorize = jest.fn(success => {
      finishAuthorization = success;
    });
    (CreateProject as jest.Mock).mockImplementation(
      (_cfg, _name, _description, _headers, callback) => {
        callback(null, { getSuccess: () => true });
      },
    );

    renderWithAuth(authorize);

    await user.click(
      screen.getByPlaceholderText('e.g., Acme Support Operations'),
    );
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support Bot'));
    await user.click(
      screen.getByPlaceholderText(
        'e.g., White-label inbound voice agents for healthcare support across US and UK',
      ),
    );
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Project description'));
    await user.click(screen.getByRole('button', { name: 'Create project' }));

    await waitFor(() => {
      expect(CreateProject).toHaveBeenCalled();
    });

    expect(mockShowLoader).toHaveBeenCalledWith('overlay');
    expect(authorize).toHaveBeenCalled();
    expect(mockHideLoader).not.toHaveBeenCalled();

    act(() => finishAuthorization?.());

    expect(mockHideLoader).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith('/dashboard');
  });

  it('requires a project name before submission', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithAuth();

    const projectName = screen.getByLabelText('Project name');
    await user.clear(projectName);
    await user.type(projectName, '   ');
    await user.click(screen.getByRole('button', { name: 'Create project' }));

    expect(
      await screen.findByText('Enter a project name.'),
    ).toBeInTheDocument();
    expect(CreateProject).not.toHaveBeenCalled();
  });

  it('leaves the project name empty when the profile has no name', () => {
    mockCredential = {
      user: {} as any,
      authId: 'auth-1',
      token: 'token-1',
    };

    renderWithAuth();

    expect(screen.getByLabelText('Project name')).toHaveValue('');
  });

  it('does not return to the completed organization step', () => {
    renderWithAuth();

    const createButton = screen.getByRole('button', {
      name: 'Create project',
    });
    expect(createButton).toHaveClass('!w-full', '!max-w-none');
    expect(screen.queryByRole('button', { name: 'Back' })).not.toBeInTheDocument();
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
      screen.getByPlaceholderText('e.g., Acme Support Operations'),
    );
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support Bot'));
    await user.click(screen.getByRole('button', { name: 'Create project' }));

    expect(
      await screen.findByText(
        'Unable to create your project. Please try again.',
      ),
    ).toBeInTheDocument();

    await user.type(screen.getByLabelText('Project name'), ' updated');
    expect(
      screen.queryByText('Unable to create your project. Please try again.'),
    ).not.toBeInTheDocument();
  });

  it('shows project details error when API returns unsuccessful response', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (CreateProject as jest.Mock).mockImplementation(
      (_cfg, _name, _description, _headers, callback) => {
        callback(null, {
          getSuccess: () => false,
          getError: () => ({
            getHumanmessage: () => 'A project with this name already exists.',
          }),
        });
      },
    );

    renderWithAuth();

    await user.click(
      screen.getByPlaceholderText('e.g., Acme Support Operations'),
    );
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Support Bot'));
    await user.click(screen.getByRole('button', { name: 'Create project' }));

    expect(
      await screen.findByText('A project with this name already exists.'),
    ).toBeInTheDocument();
  });
});

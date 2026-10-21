import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { SignInPage } from '@/app/pages/authentication/sign-in';
import { SignUpPage } from '@/app/pages/authentication/sign-up';
import { AuthContext } from '@/context/auth-context';
import { AuthenticateUser, Google, RegisterUser } from '@rapidaai/react';

const mockNavigate = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockGoTo = jest.fn();
const mockTheme = {
  links: {
    terms: 'https://example.com/terms',
    privacy: 'https://example.com/privacy',
    support: 'mailto:support@example.com',
  },
};

let mockSearchParams = new URLSearchParams();
let mockLocationState: any = undefined;
let mockParams: Record<string, string | undefined> = {};

let mockWorkspace = {
  authentication: {
    signIn: {
      providers: {
        password: true,
        google: true,
        linkedin: false,
        github: false,
      },
    },
    signUp: {
      enable: true,
      providers: {
        password: true,
        google: true,
        linkedin: false,
        github: false,
      },
    },
  },
};

jest.mock('@rapidaai/react', () => {
  class ConnectionConfig {
    static WithDebugger(config: unknown) {
      return config;
    }
  }

  return {
    ConnectionConfig,
    AuthenticateUser: jest.fn(),
    Google: jest.fn(),
    Linkedin: jest.fn(),
    Github: jest.fn(),
    RegisterUser: jest.fn(),
  };
});

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
  useSearchParams: () => [mockSearchParams],
  useLocation: () => ({ state: mockLocationState }),
  useParams: () => mockParams,
}));

jest.mock('@/workspace', () => ({
  useWorkspace: () => mockWorkspace,
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goTo: mockGoTo,
  }),
}));

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => ({ theme: mockTheme }),
}));

jest.mock('@/configs', () => ({
  connectionConfig: {},
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/app/components/ui/primitives/buttons/social-button-group', () => ({
  SocialButtonGroup: ({
    actionLabel,
    google,
    linkedin,
  }: {
    actionLabel?: string;
    google: boolean;
    linkedin: boolean;
  }) => (
    <div
      data-action-label={actionLabel}
      data-google={google}
      data-linkedin={linkedin}
      data-testid="social-buttons"
    />
  ),
}));

jest.mock('@/app/components/ui/primitives/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  FluidTextInput: require('react').forwardRef(
    ({ id, invalid, invalidText, labelText, ...props }: any, ref: any) => (
      <div data-fluid-input>
        <label htmlFor={id}>{labelText}</label>
        <input ref={ref} id={id} aria-invalid={invalid} {...props} />
        {invalid && <span>{invalidText}</span>}
      </div>
    ),
  ),
  FluidPasswordInput: require('react').forwardRef(
    ({ id, invalid, invalidText, labelText, ...props }: any, ref: any) => (
      <div data-fluid-input>
        <label htmlFor={id}>{labelText}</label>
        <input ref={ref} id={id} aria-invalid={invalid} {...props} />
        {invalid && <span>{invalidText}</span>}
      </div>
    ),
  ),
  TextInput: require('react').forwardRef(
    ({ id, invalid, invalidText, labelText, ...props }: any, ref: any) => (
      <div>
        <label htmlFor={id}>{labelText}</label>
        <input ref={ref} id={id} aria-invalid={invalid} {...props} />
        {invalid && <span>{invalidText}</span>}
      </div>
    ),
  ),
}));

jest.mock('@/app/components/ui/feedback/notification', () => ({
  Notification: ({ subtitle }: { subtitle: string }) => <div>{subtitle}</div>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({
    children,
    isLoading: _i,
    renderIcon: _r,
    hasIconOnly: _h,
    iconDescription: _d,
    ...props
  }: any) => <button {...props}>{children}</button>,
}));

jest.mock('@carbon/react', () => ({
  FluidForm: ({ children, ...props }: any) => (
    <form {...props}>{children}</form>
  ),
  Link: ({ href, inline: _inline, children, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
  PasswordInput: require('react').forwardRef(
    ({ id, invalid, invalidText, labelText, ...props }: any, ref: any) => (
      <div>
        <label htmlFor={id}>{labelText}</label>
        <input ref={ref} id={id} aria-invalid={invalid} {...props} />
        {invalid && <span>{invalidText}</span>}
      </div>
    ),
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  ArrowRight: () => null,
}));

const renderWithAuth = (
  ui: React.ReactElement,
  setAuthentication = jest.fn(),
) => {
  render(
    <AuthContext.Provider value={{ setAuthentication } as any}>
      {ui}
    </AuthContext.Provider>,
  );
  return { setAuthentication };
};

describe('Authentication pages', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockSearchParams = new URLSearchParams();
    mockLocationState = undefined;
    mockParams = {};
    mockWorkspace = {
      authentication: {
        signIn: {
          providers: {
            password: true,
            google: true,
            linkedin: false,
            github: false,
          },
        },
        signUp: {
          enable: true,
          providers: {
            password: true,
            google: false,
            linkedin: true,
            github: false,
          },
        },
      },
    };
  });

  it('sign-in submits credentials and navigates to dashboard on success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setAuthentication = jest.fn((_auth, cb) => cb());
    (AuthenticateUser as jest.Mock).mockImplementation(
      (_cfg, _email, _password, callback) => {
        callback(null, {
          getSuccess: () => true,
          getData: () => ({ id: 'auth-1' }),
        });
      },
    );

    renderWithAuth(<SignInPage />, setAuthentication);

    await user.click(screen.getByLabelText('Email address'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('john@example.com'));
    await user.click(screen.getByLabelText('Password'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getByRole('button', { name: 'Log in' }));

    await waitFor(() => {
      expect(AuthenticateUser).toHaveBeenCalled();
    });

    expect(mockShowLoader).toHaveBeenCalled();
    expect(mockHideLoader).toHaveBeenCalled();
    expect(setAuthentication).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith('/dashboard');
  });

  it('sign-in renders Carbon login terminology and action hierarchy', () => {
    renderWithAuth(<SignInPage />);

    expect(screen.getByTestId('signin-panel')).toHaveClass(
      'max-w-[37rem]',
      'bg-surface',
    );
    expect(screen.getByTestId('signin-panel')).not.toHaveClass('border');
    expect(screen.getByTestId('signin-header')).toHaveClass('p-6', 'sm:p-8');
    expect(screen.getByTestId('signin-header')).not.toHaveClass('border-b');
    expect(screen.getByTestId('signin-form')).toHaveClass('p-0');
    expect(screen.getByTestId('signin-input-stack')).toHaveClass(
      'flex-col',
      'mx-6',
      'sm:mx-8',
    );
    expect(screen.getByTestId('signin-actions')).toHaveClass('p-6', 'sm:p-8');
    expect(screen.getByTestId('signin-alternates')).toHaveClass(
      'p-6',
      'sm:p-8',
    );
    expect(screen.getByTestId('signin-alternates')).not.toHaveClass('border-t');
    expect(
      screen.getByRole('heading', { level: 1, name: 'Log in' }),
    ).toHaveClass(
      'text-[2rem]',
      'leading-[2.5rem]',
      'font-normal',
      'tracking-[0]',
    );
    const subtitle = screen
      .getByRole('link', { name: 'Create an account' })
      .closest('p');
    expect(subtitle).toHaveClass(
      'mt-2',
      'text-[1rem]',
      'font-normal',
      'leading-[1.5rem]',
      'tracking-[0]',
    );
    expect(subtitle).toHaveTextContent(
      'Don’t have an account? Create an account',
    );
    const signUpLink = screen.getByRole('link', { name: 'Create an account' });
    expect(signUpLink).toHaveAttribute('href', '/auth/signup');
    const forgotPasswordLink = screen.getByRole('link', {
      name: 'Forgot password?',
    });
    expect(forgotPasswordLink).toHaveAttribute(
      'href',
      '/auth/forgot-password',
    );
    expect(forgotPasswordLink.firstElementChild).toHaveClass(
      'text-[0.875rem]',
      'leading-[1.125rem]',
      'tracking-[0.16px]',
    );
    expect(
      screen.getByLabelText('Email address').parentElement,
    ).toHaveAttribute('data-fluid-input');
    expect(screen.getByLabelText('Password').parentElement).toHaveAttribute(
      'data-fluid-input',
    );
    expect(screen.getByRole('button', { name: 'Log in' })).toHaveClass(
      'mt-6!',
      'w-full!',
      'max-w-none!',
    );
    expect(screen.getByTestId('social-buttons')).toHaveAttribute(
      'data-action-label',
      'Log in',
    );
    expect(screen.getByTestId('social-buttons')).toBeInTheDocument();
  });

  it('sign-in shows field-level errors before calling the client', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithAuth(<SignInPage />);

    await user.click(screen.getByRole('button', { name: 'Log in' }));

    expect(await screen.findByText('Enter your email address.')).toBeVisible();
    expect(screen.getByText('Enter your password.')).toBeVisible();
    expect(AuthenticateUser).not.toHaveBeenCalled();
  });

  it('sign-in rejects a malformed email before calling the client', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithAuth(<SignInPage />);

    await user.type(screen.getByLabelText('Email address'), 'not-an-email');
    await user.type(screen.getByLabelText('Password'), 'secret');
    await user.click(screen.getByRole('button', { name: 'Log in' }));

    expect(
      await screen.findByText('Enter a valid email address.'),
    ).toBeVisible();
    expect(AuthenticateUser).not.toHaveBeenCalled();
  });

  it('sign-in clears the password and focuses email after rejection', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (AuthenticateUser as jest.Mock).mockImplementation(
      (_cfg, _email, _password, callback) => {
        callback(null, {
          getSuccess: () => false,
          getError: () => ({
            getHumanmessage: () => 'Invalid email or password.',
          }),
        });
      },
    );
    renderWithAuth(<SignInPage />);

    const email = screen.getByLabelText('Email address');
    const password = screen.getByLabelText('Password');
    await user.type(email, 'user@example.test');
    await user.type(password, 'incorrect');
    await user.click(screen.getByRole('button', { name: 'Log in' }));

    expect(await screen.findByText('Invalid email or password.')).toBeVisible();
    expect(password).toHaveValue('');
    expect(email).toHaveFocus();
  });

  it('sign-in hides sign-up prompt when workspace sign-up is disabled', () => {
    mockWorkspace.authentication.signUp.enable = false;

    renderWithAuth(<SignInPage />);

    expect(
      screen.queryByRole('link', { name: 'Create an account' }),
    ).not.toBeInTheDocument();
  });

  it('sign-in triggers social google auth when code/state are present', async () => {
    mockSearchParams = new URLSearchParams('state=google&code=abc123');

    renderWithAuth(<SignInPage />);

    await waitFor(() => {
      expect(Google).toHaveBeenCalled();
    });

    expect(mockShowLoader).toHaveBeenCalled();
  });

  it('sign-up disabled shows 403 and routes back to signin', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockWorkspace.authentication.signUp.enable = false;

    renderWithAuth(<SignUpPage />);

    expect(screen.getByText('403')).toBeInTheDocument();
    expect(
      screen.getByText('Account creation unavailable'),
    ).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Go to log in' }));
    expect(mockGoTo).toHaveBeenCalledWith('/');
  });

  it('sign-up prefills email from location state and submits register', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockLocationState = { email: 'prefilled@example.com' };
    mockParams = { next: '/workspace/overview' };

    const setAuthentication = jest.fn((_auth, cb) => cb());
    (RegisterUser as jest.Mock).mockImplementation(
      (_cfg, _email, _password, _name, callback) => {
        callback(null, {
          getSuccess: () => true,
          getData: () => ({ id: 'auth-2' }),
        });
      },
    );

    renderWithAuth(<SignUpPage />, setAuthentication);

    const emailInput = screen.getByLabelText(
      'Email address',
    ) as HTMLInputElement;
    await waitFor(() => {
      expect(emailInput.value).toBe('prefilled@example.com');
    });

    await user.click(screen.getByLabelText('Full name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('John Doe'));
    await user.click(screen.getByLabelText('Password'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));

    await user.click(screen.getByRole('button', { name: 'Create account' }));

    await waitFor(() => {
      expect(RegisterUser).toHaveBeenCalled();
    });

    expect(mockShowLoader).toHaveBeenCalledWith('overlay');
    expect(mockHideLoader).toHaveBeenCalled();
    expect(setAuthentication).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith('/workspace/overview');
  });

  it('sign-up renders updated header and policy links', () => {
    renderWithAuth(<SignUpPage />);

    expect(screen.getByTestId('signup-panel')).toHaveClass(
      'max-w-[37rem]',
      'bg-surface',
    );
    expect(screen.getByTestId('signup-header')).toHaveClass('p-6', 'sm:p-8');
    expect(screen.getByTestId('signup-form')).toHaveClass('p-0');
    expect(screen.getByTestId('signup-input-stack')).toHaveClass(
      'mx-6',
      'sm:mx-8',
      'flex-col',
    );
    expect(screen.getByTestId('signup-actions')).toHaveClass('p-6', 'sm:p-8');
    expect(screen.getByTestId('signup-alternates')).toHaveClass(
      'p-6',
      'sm:p-8',
    );
    expect(screen.getByTestId('signup-alternates')).not.toHaveClass('border-t');
    expect(
      screen.getByRole('heading', { level: 1, name: 'Create account' }),
    ).toHaveClass('text-[2rem]', 'leading-[2.5rem]', 'font-normal');
    expect(screen.getByRole('link', { name: 'Log in' })).toHaveAttribute(
      'href',
      '/auth/signin',
    );
    expect(
      screen.getByRole('link', { name: 'Terms and conditions' }),
    ).toHaveAttribute('href', mockTheme.links.terms);
    expect(
      screen.getByRole('link', { name: 'Privacy policy' }),
    ).toHaveAttribute('href', mockTheme.links.privacy);
    expect(screen.getByTestId('social-buttons')).toHaveAttribute(
      'data-action-label',
      'Sign up',
    );
    expect(screen.getByTestId('social-buttons')).toHaveAttribute(
      'data-google',
      'false',
    );
    expect(screen.getByTestId('social-buttons')).toHaveAttribute(
      'data-linkedin',
      'true',
    );
    expect(screen.getByLabelText('Full name').parentElement).toHaveAttribute(
      'data-fluid-input',
    );
    expect(
      screen.getByLabelText('Email address').parentElement,
    ).toHaveAttribute('data-fluid-input');
    expect(screen.getByLabelText('Password').parentElement).toHaveAttribute(
      'data-fluid-input',
    );
    expect(screen.getByText(/By creating an account/).closest('p')).toHaveClass(
      'text-[0.75rem]',
      'leading-[1rem]',
      'tracking-[0.32px]',
    );
    expect(screen.getByRole('button', { name: 'Create account' })).toHaveClass(
      'mt-6!',
    );
  });

  it('sign-up shows field-level errors before calling the client', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithAuth(<SignUpPage />);

    await user.click(screen.getByRole('button', { name: 'Create account' }));

    expect(await screen.findByText('Enter your full name.')).toBeVisible();
    expect(screen.getByText('Enter your email address.')).toBeVisible();
    expect(screen.getByText('Enter a password.')).toBeVisible();
    expect(RegisterUser).not.toHaveBeenCalled();
  });

  it('sign-up rejects a malformed email before calling the client', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderWithAuth(<SignUpPage />);

    await user.type(screen.getByLabelText('Full name'), 'John Doe');
    await user.type(screen.getByLabelText('Email address'), 'not-an-email');
    await user.type(screen.getByLabelText('Password'), 'secret');
    await user.click(screen.getByRole('button', { name: 'Create account' }));

    expect(
      await screen.findByText('Enter a valid email address.'),
    ).toBeVisible();
    expect(RegisterUser).not.toHaveBeenCalled();
  });

  it('sign-up shows API human error message on register failure', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (RegisterUser as jest.Mock).mockImplementation(
      (_cfg, _email, _password, _name, callback) => {
        callback(null, {
          getSuccess: () => false,
          getError: () => ({ getHumanmessage: () => 'Email already exists' }),
        });
      },
    );

    renderWithAuth(<SignUpPage />);

    await user.click(screen.getByLabelText('Full name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('John Doe'));
    await user.click(screen.getByLabelText('Email address'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('john@example.com'));
    await user.click(screen.getByLabelText('Password'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));

    await user.click(screen.getByRole('button', { name: 'Create account' }));

    expect(await screen.findByText('Email already exists')).toBeInTheDocument();
  });
});

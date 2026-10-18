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
  TextInput: require('react').forwardRef(
    ({ labelText: _labelText, ...props }: any, ref: any) => (
      <input ref={ref} {...props} />
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
  Link: ({ href, inline: _inline, children, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
  PasswordInput: require('react').forwardRef(
    ({ labelText: _labelText, ...props }: any, ref: any) => (
      <input ref={ref} {...props} />
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

    await user.click(screen.getByPlaceholderText('eg: john@example.com'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('john@example.com'));
    await user.click(screen.getByPlaceholderText('******'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    await waitFor(() => {
      expect(AuthenticateUser).toHaveBeenCalled();
    });

    expect(mockShowLoader).toHaveBeenCalled();
    expect(mockHideLoader).toHaveBeenCalled();
    expect(setAuthentication).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith('/dashboard');
  });

  it('sign-in renders updated header and links', () => {
    renderWithAuth(<SignInPage />);

    expect(
      screen.getByRole('heading', { level: 1, name: 'Sign in' }),
    ).toBeInTheDocument();
    const signUpLink = screen.getByRole('link', { name: 'Sign up' });
    expect(signUpLink).toHaveAttribute('href', '/auth/signup');
    expect(signUpLink).toHaveClass('!underline');
    expect(
      screen.getByRole('link', { name: "Can't sign in?" }),
    ).toHaveAttribute('href', '/auth/forgot-password');
    expect(screen.getByTestId('social-buttons')).toBeInTheDocument();
  });

  it('sign-in hides sign-up prompt when workspace sign-up is disabled', () => {
    mockWorkspace.authentication.signUp.enable = false;

    renderWithAuth(<SignInPage />);

    expect(
      screen.queryByRole('link', { name: 'Sign up' }),
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

    await user.click(screen.getByRole('button', { name: 'Go to sign in' }));
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

    const emailInput = screen.getByPlaceholderText(
      'eg: john@example.com',
    ) as HTMLInputElement;
    await waitFor(() => {
      expect(emailInput.value).toBe('prefilled@example.com');
    });

    await user.click(screen.getByPlaceholderText('eg: John Doe'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('John Doe'));
    await user.click(screen.getByPlaceholderText('********'));
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

    expect(
      screen.getByRole('heading', { level: 1, name: 'Create account' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute(
      'href',
      '/auth/signin',
    );
    expect(
      screen.getByRole('link', { name: 'Terms and Conditions' }),
    ).toHaveAttribute('href', mockTheme.links.terms);
    expect(
      screen.getByRole('link', { name: 'Privacy Policy' }),
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

    await user.click(screen.getByPlaceholderText('eg: John Doe'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('John Doe'));
    await user.click(screen.getByPlaceholderText('eg: john@example.com'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('john@example.com'));
    await user.click(screen.getByPlaceholderText('********'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('secret'));

    await user.click(screen.getByRole('button', { name: 'Create account' }));

    expect(await screen.findByText('Email already exists')).toBeInTheDocument();
  });
});

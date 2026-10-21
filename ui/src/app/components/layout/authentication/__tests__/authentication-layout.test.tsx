import React from 'react';
import { render, screen, within } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { AuthenticationLayout } from '../authentication-layout';

const baseTheme = developmentConfig.theme as unknown as ThemeManifest;

const renderLayout = (
  theme: ThemeManifest = baseTheme,
  initialPath = '/auth/signin',
) =>
  render(
    <ThemeProvider
      theme={{ ...theme, defaultMode: 'light', allowModeSelection: false }}
    >
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/auth" element={<AuthenticationLayout />}>
            <Route path="signin" element={<h1>Log in</h1>} />
            <Route path="signup" element={<h1>Create account</h1>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  );

describe('AuthenticationLayout', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
  });

  it('renders the branded shell around one authentication main landmark', () => {
    renderLayout();

    const [authentication] = screen.getAllByRole('main');
    expect(screen.getAllByRole('main')).toHaveLength(1);
    expect(authentication).toHaveAccessibleName('Authentication');
    const banner = screen.getByRole('banner');
    expect(banner).toHaveAccessibleName(`${baseTheme.brand.name} Platform`);
    expect(banner).toHaveClass('cds--header', 'relative!', 'h-12!');
    expect(
      within(banner).getByRole('link', { name: baseTheme.brand.name }),
    ).toHaveClass('cds--header__name');
    expect(
      within(banner).getByRole('link', { name: baseTheme.brand.name }),
    ).toHaveAttribute('href', '/');
    expect(screen.queryByTestId('auth-header-grid')).not.toBeInTheDocument();
    expect(screen.getByRole('contentinfo')).toHaveAccessibleName(
      `${baseTheme.brand.name} platform footer`,
    );
    expect(
      within(screen.getByRole('contentinfo')).getByRole('link', {
        name: `${baseTheme.brand.name} Platform`,
      }),
    ).toHaveClass('cds--header__name');
    expect(
      within(screen.getByRole('contentinfo')).getByRole('navigation'),
    ).toHaveClass('cds--header__nav');
    expect(
      within(screen.getByRole('contentinfo')).getAllByRole('link'),
    ).toHaveLength(6);
    expect(screen.getByAltText(baseTheme.brand.name)).toBeInTheDocument();
    expect(screen.getByAltText(baseTheme.brand.name).parentElement).toHaveClass(
      'flex',
      'items-center',
      'whitespace-nowrap',
    );

    const accountAccess = screen.getByRole('region', {
      name: 'Account access',
    });
    expect(
      within(accountAccess).getByRole('heading', { name: 'Log in' }),
    ).toBeInTheDocument();
    expect(authentication).toContainElement(accountAccess);
  });

  it('centers account access on the Carbon layer without decorative borders', () => {
    renderLayout();

    const authentication = screen.getByRole('main', {
      name: 'Authentication',
    });
    const accountAccess = screen.getByRole('region', {
      name: 'Account access',
    });
    const authMain = screen.getByTestId('auth-main');

    expect(authentication).toContainElement(accountAccess);
    expect(authMain).toHaveClass(
      'flex-1',
      'min-h-0',
      'bg-layer',
      'p-4',
      'sm:p-10',
    );
    expect(accountAccess).toHaveClass(
      'max-w-[37rem]',
      'mx-auto',
      'flex',
      'justify-center',
    );
    expect(screen.queryByTestId('auth-center-box')).not.toBeInTheDocument();
    expect(screen.queryByTestId('auth-form-rail')).not.toBeInTheDocument();
    expect(screen.queryByTestId('auth-illustration')).not.toBeInTheDocument();
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument();
  });

  it('keeps every authentication page on the shared form measure', () => {
    renderLayout(baseTheme, '/auth/signup');

    expect(screen.getByRole('region', { name: 'Account access' })).toHaveClass(
      'max-w-[37rem]',
    );
  });

  it('uses Carbon UI Shell navigation for the authentication footer', () => {
    renderLayout();

    const footer = screen.getByRole('contentinfo');
    const footerNavigation = within(footer).getByRole('navigation');

    expect(footer).toHaveClass('h-12', 'bg-shell', 'border-border-subtle');
    expect(footerNavigation).toHaveClass('cds--header__nav');
    expect(within(footerNavigation).getAllByRole('link')).toHaveLength(5);
    expect(
      within(footerNavigation).getByRole('link', { name: 'Support' }),
    ).toBeInTheDocument();
    expect(
      within(footer).getByRole('link', {
        name: `${baseTheme.brand.name} Platform`,
      }),
    ).toHaveAttribute('href', '/');
  });
});

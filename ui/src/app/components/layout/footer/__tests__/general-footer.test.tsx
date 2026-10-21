import React from 'react';
import { render, screen } from '@testing-library/react';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { GeneralFooter } from '../general-footer';

const theme = developmentConfig.theme as unknown as ThemeManifest;

describe('GeneralFooter', () => {
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

  it('renders the themed footer landmark and brand link', () => {
    render(
      <ThemeProvider theme={theme}>
        <GeneralFooter className="custom-footer" data-testid="footer" />
      </ThemeProvider>,
    );

    expect(screen.getByRole('contentinfo')).toHaveAttribute(
      'aria-label',
      `${theme.brand.name} platform footer`,
    );
    expect(screen.getByTestId('footer')).toHaveClass(
      'custom-footer',
      'bg-shell',
      'border-border-subtle',
    );
    expect(
      screen.getByRole('link', { name: `${theme.brand.name} Platform` }),
    ).toHaveAttribute('href', '/');
    expect(screen.getByText(`${theme.brand.name} Platform`)).toHaveClass(
      'text-[0.875rem]',
      'font-[600]',
      'leading-[1.125rem]',
      'tracking-[0.16px]',
    );
    expect(screen.getByRole('navigation')).toHaveAttribute(
      'aria-label',
      `${theme.brand.name} footer links`,
    );
  });

  it('renders all configured footer links', () => {
    render(
      <ThemeProvider theme={theme}>
        <GeneralFooter />
      </ThemeProvider>,
    );

    const links = [
      ['Terms and Conditions', theme.links.terms],
      ['Privacy Policy', theme.links.privacy],
      ['Documentation', theme.links.documentation],
      ['Source', theme.links.source],
      ['Support', theme.links.support],
    ] as const;

    links.forEach(([name, href]) => {
      const link = screen.getByRole('link', { name });
      expect(link).toHaveAttribute('href', href);
      expect(link.closest('li')).toHaveClass('[&>a]:bg-transparent!');
      expect(screen.getByText(name)).toHaveClass(
        'text-[0.875rem]',
        'font-normal',
        'leading-[1.125rem]',
        'tracking-[0.16px]',
      );
    });
  });
});

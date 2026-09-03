import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { PrivacyPage } from '../privacy';
import { TermsPage } from '../terms';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';

const theme: ThemeManifest = {
  schemaVersion: 1,
  id: 'custom',
  brand: {
    name: 'Acme Voice',
  },
  links: {
    documentation: '#',
    source: '#',
    support: 'mailto:help@acme.example',
    terms: '#',
    privacy: '#',
  },
  defaultMode: 'light',
  allowModeSelection: false,
  colors: {
    light: {
      primary: '#111111',
      primaryHover: '#222222',
      primaryActive: '#333333',
      onPrimary: '#ffffff',
    },
    dark: {
      primary: '#eeeeee',
      primaryHover: '#dddddd',
      primaryActive: '#cccccc',
      onPrimary: '#000000',
    },
  },
};

const renderWithTheme = (component: React.ReactElement) =>
  render(<ThemeProvider theme={theme}>{component}</ThemeProvider>);

describe('static legal page brand copy', () => {
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

  it('renders privacy copy from the configured brand and support link', () => {
    renderWithTheme(<PrivacyPage />);

    expect(document.body).toHaveTextContent(
      'Acme Voice ("we", "us", or "our")',
    );
    expect(document.body).toHaveTextContent(
      'When you create an account on Acme Voice',
    );
    expect(
      screen.getAllByRole('link', { name: 'help@acme.example' }),
    ).toHaveLength(2);
    expect(document.body).not.toHaveTextContent('Rapida');
  });

  it('renders terms copy from the configured brand and support link', () => {
    renderWithTheme(<TermsPage />);

    expect(document.body).toHaveTextContent(
      'the Acme Voice platform ("Platform")',
    );
    expect(document.body).toHaveTextContent('ACME VOICE SHALL NOT BE LIABLE');
    expect(
      screen.getByRole('link', { name: 'help@acme.example' }),
    ).toHaveAttribute('href', 'mailto:help@acme.example');
    expect(document.body).not.toHaveTextContent('Rapida');
  });
});

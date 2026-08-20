import React from 'react';
import { render, screen } from '@testing-library/react';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { GeneralFooter } from './general-footer';

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

  it('uses semantic theme borders and configured links', () => {
    render(
      <ThemeProvider theme={theme}>
        <GeneralFooter />
      </ThemeProvider>,
    );

    expect(screen.getByRole('banner')).toHaveClass('border-border-subtle!');
    expect(screen.getByRole('link', { name: 'Documentation' })).toHaveAttribute(
      'href',
      theme.links.documentation,
    );
  });
});

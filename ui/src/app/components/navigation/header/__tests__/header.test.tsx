import React from 'react';
import { render, screen } from '@testing-library/react';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { Header } from '../index';

const theme = developmentConfig.theme as unknown as ThemeManifest;

const renderHeader = (ariaLabel?: string) =>
  render(
    <ThemeProvider theme={theme}>
      <Header aria-label={ariaLabel} />
    </ThemeProvider>,
  );

describe('Header', () => {
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

  it('uses the brand label by default', () => {
    renderHeader();

    expect(screen.getByRole('banner')).toHaveAttribute(
      'aria-label',
      'Rapida AI Platform',
    );
  });

  it('preserves a caller-provided accessible label', () => {
    renderHeader('Project navigation');

    expect(screen.getByRole('banner')).toHaveAttribute(
      'aria-label',
      'Project navigation',
    );
  });
});

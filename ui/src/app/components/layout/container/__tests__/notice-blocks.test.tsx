import React from 'react';
import { render, screen } from '@testing-library/react';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import type { ThemeManifest } from '@/theme/types';
import {
  BlueNoticeBlock,
  GreenNoticeBlock,
  RedNoticeBlock,
  YellowNoticeBlock,
} from '../message/notice-block';
import { DocNoticeBlock } from '../message/notice-block/doc-notice-block';

const theme = developmentConfig.theme as unknown as ThemeManifest;

const renderWithTheme = (element: React.ReactElement) =>
  render(<ThemeProvider theme={theme}>{element}</ThemeProvider>);

describe('layout notice blocks', () => {
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

  it('renders status notice variants with tone classes and div attributes', () => {
    render(
      <>
        <BlueNoticeBlock className="info-extra" data-testid="info">
          Info message
        </BlueNoticeBlock>
        <GreenNoticeBlock data-testid="success">Saved</GreenNoticeBlock>
        <YellowNoticeBlock data-testid="warning">Check setup</YellowNoticeBlock>
      </>,
    );

    expect(screen.getByTestId('info')).toHaveClass(
      'info-extra',
      'border-l-support-info',
      'text-foreground',
    );
    expect(screen.getByTestId('info')).toHaveAttribute('role', 'status');
    expect(screen.getByTestId('success')).toHaveClass(
      'border-l-support-success',
    );
    expect(screen.getByTestId('warning')).toHaveClass(
      'border-l-support-warning',
    );
  });

  it('renders error notices as alerts', () => {
    render(<RedNoticeBlock data-testid="error">Fix settings</RedNoticeBlock>);

    expect(screen.getByRole('alert')).toHaveTextContent('Fix settings');
    expect(screen.getByTestId('error')).toHaveClass('border-l-support-error');
  });

  it('renders documentation links from theme paths or explicit URLs', () => {
    const { rerender } = renderWithTheme(
      <DocNoticeBlock
        className="doc-extra"
        data-testid="doc-notice"
        docPath="/knowledge/overview"
      >
        Read before continuing.
      </DocNoticeBlock>,
    );

    expect(screen.getByRole('note')).toHaveClass('doc-extra');
    expect(screen.getByText('Read before continuing.')).toHaveClass(
      'text-foreground',
    );
    expect(
      screen.getByRole('link', { name: 'Read documentation' }),
    ).toHaveAttribute('href', 'https://doc.rapida.ai/knowledge/overview');

    rerender(
      <ThemeProvider theme={theme}>
        <DocNoticeBlock docUrl="https://example.com/docs" linkText="Open guide">
          Explicit docs
        </DocNoticeBlock>
      </ThemeProvider>,
    );

    expect(screen.getByRole('link', { name: 'Open guide' })).toHaveAttribute(
      'href',
      'https://example.com/docs',
    );
  });
});

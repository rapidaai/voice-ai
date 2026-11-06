import React from 'react';
import developmentConfig from '@/configs/config.development.json';

const mockRender = jest.fn();
const mockCreateRoot = jest.fn(() => ({ render: mockRender }));
const mockApplyThemeToDocument = jest.fn();
const mockInitializeAnalytics = jest.fn();
let mockConfiguredTheme: unknown;

jest.mock('react-dom/client', () => ({
  __esModule: true,
  default: { createRoot: mockCreateRoot },
}));
jest.mock('@/configs', () => ({
  get CONFIG() {
    return { theme: mockConfiguredTheme };
  },
}));
jest.mock('@/theme/theme-provider', () => ({
  applyThemeToDocument: mockApplyThemeToDocument,
  getInitialThemeState: () => ({ resolvedMode: 'light' }),
  ThemeProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));
jest.mock('@/react-web-analytics', () => ({
  initializeAnalytics: mockInitializeAnalytics,
}));
jest.mock('@/app', () => ({ App: () => null }));
jest.mock('react-helmet-async', () => ({
  HelmetProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));
describe('application entry point', () => {
  beforeEach(() => {
    jest.resetModules();
    mockConfiguredTheme = developmentConfig.theme;
    mockRender.mockClear();
    mockCreateRoot.mockClear();
    mockCreateRoot.mockImplementation(() => ({ render: mockRender }));
    mockApplyThemeToDocument.mockClear();
    mockInitializeAnalytics.mockClear();
    document.body.innerHTML = '<div id="root"></div>';
    jest.spyOn(console, 'error').mockImplementation();
  });

  afterEach(() => jest.restoreAllMocks());

  it('keeps permissive version 1 values and continues startup', () => {
    mockConfiguredTheme = {
      ...developmentConfig.theme,
      schemaVersion: 1,
      links: {
        ...developmentConfig.theme.links,
        support: 'client support channel',
      },
      colors: {
        ...developmentConfig.theme.colors,
        dark: {
          ...developmentConfig.theme.colors.dark,
          primaryHover: '#7ccdf7l',
        },
      },
    };

    jest.isolateModules(() => require('./index'));

    expect(document.getElementById('root')).not.toHaveTextContent(
      'Application configuration error',
    );
    expect(mockApplyThemeToDocument).toHaveBeenCalledWith(
      expect.objectContaining({
        links: expect.objectContaining({ support: 'client support channel' }),
        colors: expect.objectContaining({
          dark: expect.objectContaining({ primaryHover: '#7ccdf7l' }),
        }),
      }),
      'light',
    );
    expect(mockInitializeAnalytics).toHaveBeenCalledTimes(1);
    expect(mockCreateRoot).toHaveBeenCalledWith(
      document.getElementById('root'),
    );
    expect(mockRender).toHaveBeenCalledTimes(1);
  });

  it('applies a valid theme before rendering the application', () => {
    jest.isolateModules(() => require('./index'));

    expect(mockApplyThemeToDocument).toHaveBeenCalledWith(
      expect.objectContaining({
        ...developmentConfig.theme,
        colors: {
          light: {
            ...developmentConfig.theme.colors.light,
            carbonTheme: 'white',
          },
          dark: {
            ...developmentConfig.theme.colors.dark,
            carbonTheme: 'g100',
          },
        },
      }),
      'light',
    );
    expect(mockInitializeAnalytics).toHaveBeenCalledTimes(1);
    expect(mockCreateRoot).toHaveBeenCalledWith(
      document.getElementById('root'),
    );
    expect(mockRender).toHaveBeenCalledTimes(1);
    expect(mockApplyThemeToDocument.mock.invocationCallOrder[0]).toBeLessThan(
      mockRender.mock.invocationCallOrder[0],
    );
  });

  it('renders fixed neutral output before analytics or React for invalid version 2', () => {
    mockConfiguredTheme = {
      ...developmentConfig.theme,
      schemaVersion: 2,
      colors: {
        ...developmentConfig.theme.colors,
        light: {
          carbonTheme: 'white',
          primary: '#0353e9',
          primaryHover: '#002d9c',
          onPrimary: '#ffffff',
        },
      },
    };

    jest.isolateModules(() => require('./index'));

    expect(document.getElementById('root')).toHaveTextContent(
      'Application configuration error',
    );
    expect(document.documentElement).not.toHaveAttribute('data-brand');
    expect(
      document.documentElement.style.getPropertyValue('--brand-primary'),
    ).toBe('');
    expect(mockApplyThemeToDocument).not.toHaveBeenCalled();
    expect(mockInitializeAnalytics).not.toHaveBeenCalled();
    expect(mockCreateRoot).not.toHaveBeenCalled();
    expect(mockRender).not.toHaveBeenCalled();
  });
});

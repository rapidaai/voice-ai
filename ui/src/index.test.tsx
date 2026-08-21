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
jest.mock('@/context/provider-credential-modal-context', () => ({
  ProviderCredentialModalProvider: ({
    children,
  }: {
    children: React.ReactNode;
  }) => <>{children}</>,
}));
jest.mock('@/workspace', () => ({
  WorkspaceProvider: ({ children }: { children: React.ReactNode }) => (
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

  it('shows the specific config diagnostic and stops startup side effects', () => {
    mockConfiguredTheme = {
      ...developmentConfig.theme,
      colors: {
        ...developmentConfig.theme.colors,
        dark: {
          ...developmentConfig.theme.colors.dark,
          primaryHover: '#7ccdf7l',
        },
      },
    };

    jest.isolateModules(() => require('./index'));

    expect(document.getElementById('root')).toHaveTextContent(
      'Application configuration error: CONFIG.theme.colors.dark.primaryHover must be a 6-digit hexadecimal color such as #0f62fe.',
    );
    expect(document.getElementById('root')).not.toHaveTextContent('#7ccdf7l');
    expect(mockApplyThemeToDocument).not.toHaveBeenCalled();
    expect(mockInitializeAnalytics).not.toHaveBeenCalled();
    expect(mockCreateRoot).not.toHaveBeenCalled();
  });

  it('applies the validated theme before rendering the application', () => {
    jest.isolateModules(() => require('./index'));

    expect(mockApplyThemeToDocument).toHaveBeenCalledWith(
      developmentConfig.theme,
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
});

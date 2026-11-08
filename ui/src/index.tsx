import * as React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from '@/app';
import { HelmetProvider } from 'react-helmet-async';
import { initializeAnalytics } from '@/react-web-analytics';
import {
  applyThemeToDocument,
  getInitialThemeState,
  ThemeProvider,
} from '@/theme/theme-provider';
import { CONFIG } from '@/configs';
import { parseThemeManifest } from '@/theme/theme-config';

const rootElement = document.getElementById('root') as HTMLElement;
const themeResult = parseThemeManifest(CONFIG.theme);

if (!themeResult.ok) {
  document.title = 'Application configuration error';
  rootElement.setAttribute('role', 'alert');
  rootElement.textContent = 'Application configuration error';
} else {
  const { theme } = themeResult;
  applyThemeToDocument(theme, getInitialThemeState(theme).resolvedMode);
  initializeAnalytics();

  ReactDOM.createRoot(rootElement).render(
    <HelmetProvider>
      <React.StrictMode>
        <ThemeProvider theme={theme}>
          <App />
        </ThemeProvider>
      </React.StrictMode>
    </HelmetProvider>,
  );
}

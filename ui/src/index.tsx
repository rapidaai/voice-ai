import 'react-app-polyfill/ie11';
import 'react-app-polyfill/stable';
import * as React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from '@/app';
import { HelmetProvider } from 'react-helmet-async';
import { ProviderCredentialModalProvider } from '@/context/provider-credential-modal-context';
import { WorkspaceProvider } from '@/workspace';
import { initializeAnalytics } from '@/react-web-analytics';
import {
  applyThemeToDocument,
  getInitialThemeState,
  ThemeProvider,
} from '@/theme/theme-provider';
import { CONFIG } from '@/configs';
import { getValidatedThemeOrRenderError } from '@/theme/theme-bootstrap';

const rootElement = document.getElementById('root') as HTMLElement;
const theme = getValidatedThemeOrRenderError(CONFIG.theme, rootElement);

if (theme) {
  applyThemeToDocument(theme, getInitialThemeState(theme).resolvedMode);
  initializeAnalytics();

  ReactDOM.createRoot(rootElement).render(
    <HelmetProvider>
      <React.StrictMode>
        <ThemeProvider theme={theme}>
          <ProviderCredentialModalProvider>
            <WorkspaceProvider>
              <App />
            </WorkspaceProvider>
          </ProviderCredentialModalProvider>
        </ThemeProvider>
      </React.StrictMode>
    </HelmetProvider>,
  );
}

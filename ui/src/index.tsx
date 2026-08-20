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
import { getBootstrapTheme, loadThemeManifest } from '@/theme/theme-loader';

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement,
);
initializeAnalytics();

const renderApp = (theme: Awaited<ReturnType<typeof loadThemeManifest>>) => {
  applyThemeToDocument(theme, getInitialThemeState(theme).resolvedMode);

  root.render(
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
};

async function bootstrap() {
  const bootstrapTheme = getBootstrapTheme();
  renderApp(bootstrapTheme);

  const runtimeTheme = await loadThemeManifest();
  if (JSON.stringify(runtimeTheme) !== JSON.stringify(bootstrapTheme)) {
    renderApp(runtimeTheme);
  }
}

void bootstrap();

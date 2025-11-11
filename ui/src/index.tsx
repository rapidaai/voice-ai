import 'react-app-polyfill/ie11';
import 'react-app-polyfill/stable';
import * as React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from '@/app';
import { HelmetProvider } from 'react-helmet-async';
import * as Sentry from '@sentry/react';
import { EnvironmentProvider } from '@/context/environment-context';
import { DarkModeProvider } from '@/context/dark-mode-context';
import { WorkspaceProvider } from '@/context/workplace-context';
const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement,
);

if (process.env.NODE_ENV && process.env.NODE_ENV === 'production') {
  Sentry.init({
    dsn: 'https://15153cb4befe6a0ae4249f10ff87c0b6@o4506771747831808.ingest.sentry.io/4506771748945920',
    integrations: [
      Sentry.browserTracingIntegration(),
      Sentry.replayIntegration({
        maskAllText: false,
        blockAllMedia: false,
      }),
    ],
    tracesSampleRate: 1.0, //  Capture 100% of the transactions
    tracePropagationTargets: [/^https:\/\/rapida\.ai\/api/],
    replaysSessionSampleRate: 0.1, // This sets the sample rate at 10%. You may want to change it to 100% while in development and then sample at a lower rate in production.
    replaysOnErrorSampleRate: 1.0, // If you're not already sampling the entire session, change the sample rate to 100% when sampling sessions where errors occur.
  });
}

root.render(
  <HelmetProvider>
    <React.StrictMode>
      <EnvironmentProvider>
        <DarkModeProvider>
          <WorkspaceProvider>
            <App />
          </WorkspaceProvider>
        </DarkModeProvider>
      </EnvironmentProvider>
    </React.StrictMode>
  </HelmetProvider>,
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
// reportWebVitals();

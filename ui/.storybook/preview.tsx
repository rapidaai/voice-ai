import React from 'react';
import { Theme as CarbonTheme } from '@carbon/react';
import type { Preview } from '@storybook/react-vite';
import { HelmetProvider } from 'react-helmet-async';
import developmentConfig from '../src/configs/config.development.json';
import { ThemeProvider } from '../src/theme/theme-provider';
import type { ResolvedThemeMode, ThemeManifest } from '../src/theme/types';
import '../src/styles/global-styles';
import '../src/monaco';

const theme = developmentConfig.theme as ThemeManifest;

const themeByMode: Record<ResolvedThemeMode, ThemeManifest> = {
  light: {
    ...theme,
    defaultMode: 'light',
    allowModeSelection: false,
  },
  dark: {
    ...theme,
    defaultMode: 'dark',
    allowModeSelection: false,
  },
};

const preview: Preview = {
  decorators: [
    (Story, context) => {
      const background = context.globals.backgrounds;
      const selectedBackground =
        typeof background === 'string' ? background : background?.value;
      const mode = selectedBackground === 'dark' ? 'dark' : 'light';
      const carbonTheme = mode === 'dark' ? 'g100' : 'white';

      return (
        <HelmetProvider>
          <ThemeProvider theme={themeByMode[mode]}>
            <CarbonTheme
              theme={carbonTheme}
              className="min-h-screen bg-[var(--cds-background)] p-4 font-sans text-[var(--cds-text-primary)]"
            >
              <Story />
            </CarbonTheme>
          </ThemeProvider>
        </HelmetProvider>
      );
    },
  ],
  initialGlobals: {
    backgrounds: { value: 'light' },
  },
  parameters: {
    backgrounds: {
      default: 'light',
      options: {
        light: { name: 'Light', value: '#ffffff' },
        dark: { name: 'Dark', value: '#0a0a0a' },
      },
    },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
  },
};

export default preview;

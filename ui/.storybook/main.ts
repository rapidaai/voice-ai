import path from 'node:path';
import { fileURLToPath } from 'node:url';
import type { StorybookConfig } from '@storybook/react-vite';
import { mergeConfig } from 'vite';

const storybookDirectory = path.dirname(fileURLToPath(import.meta.url));

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(js|jsx|mjs|ts|tsx)'],
  addons: [
    '@storybook/addon-docs',
    '@storybook/addon-a11y',
    '@chromatic-com/storybook',
  ],
  framework: {
    name: '@storybook/react-vite',
    options: {},
  },
  typescript: {
    check: false,
    reactDocgen: false,
  },
  staticDirs: ['../public'],
  viteFinal: async storybookConfig =>
    mergeConfig(storybookConfig, {
      define: {
        'process.env': JSON.stringify({
          MODE: 'development',
          NODE_ENV: 'development',
          VITE_MODE: 'development',
          VITE_GA_MEASUREMENT_ID: process.env.VITE_GA_MEASUREMENT_ID,
          REACT_APP_GA_MEASUREMENT_ID:
            process.env.REACT_APP_GA_MEASUREMENT_ID,
        }),
      },
      resolve: {
        alias: {
          '@': path.resolve(storybookDirectory, '../src'),
          '@rapidaai/react': path.resolve(
            storybookDirectory,
            '../node_modules/@rapidaai/react/dist/index.js',
          ),
        },
      },
      worker: {
        format: 'es',
      },
    }),
};

export default config;

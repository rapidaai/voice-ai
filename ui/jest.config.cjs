const os = require('node:os');

const shouldUseAllure = process.env.ALLURE_REPORT === 'true';

module.exports = {
  roots: ['<rootDir>/src'],
  collectCoverageFrom: [
    'src/**/*.{js,jsx,ts,tsx}',
    'public/**/*',
    '!src/**/*.d.ts',
    '!src/**/*.stories.{js,jsx,ts,tsx}',
    '!src/**/Loadable.{js,jsx,ts,tsx}',
    '!src/**/messages.ts',
    '!src/**/types.ts',
    '!src/index.tsx',
    '!src/providers/provider-assets.ts',
  ],
  coverageThreshold: {
    global: {
      branches: 5,
      functions: 5,
      lines: 5,
      statements: 5,
    },
  },
  setupFilesAfterEnv: ['<rootDir>/src/setup-tests.ts'],
  testEnvironment: shouldUseAllure ? 'allure-jest/jsdom' : 'jsdom',
  testEnvironmentOptions: shouldUseAllure
    ? {
        resultsDir: 'allure-results',
        environmentInfo: {
          project: 'rapida.ai ui',
          testRunner: 'jest',
          node: process.version,
          os: `${os.type()} ${os.release()}`,
        },
        globalLabels: {
          layer: 'ui',
        },
      }
    : {},
  testMatch: [
    '<rootDir>/src/**/__tests__/**/*.{js,jsx,ts,tsx}',
    '<rootDir>/src/**/*.{spec,test}.{js,jsx,ts,tsx}',
  ],
  transform: {
    '^.+\\.(js|jsx|mjs|cjs|ts|tsx)$': [
      'babel-jest',
      {
        presets: [['react-app', { runtime: 'automatic' }]],
        babelrc: false,
        configFile: false,
      },
    ],
  },
  transformIgnorePatterns: [
    '[/\\\\]node_modules[/\\\\].+\\.(js|jsx|mjs|cjs|ts|tsx)$',
  ],
  moduleNameMapper: {
    '^@/monaco$': '<rootDir>/scripts/jest/monaco-mock.cjs',
    '^@/providers/provider-assets$':
      '<rootDir>/src/providers/provider-assets.jest.ts',
    '^@/(.*)$': '<rootDir>/src/$1',
    '^src/(.*)$': '<rootDir>/src/$1',
    '^react-native$': 'react-native-web',
    '^.+\\.module\\.(css|sass|scss)$': 'identity-obj-proxy',
    '^.+\\.(css|sass|scss)$': '<rootDir>/scripts/jest/style-mock.cjs',
    '\\?worker$': '<rootDir>/scripts/jest/worker-mock.cjs',
    '^.+\\.(bmp|gif|jpe?g|png|svg|webp|avif|woff2?|ttf|otf)$':
      '<rootDir>/scripts/jest/file-mock.cjs',
  },
  moduleFileExtensions: [
    'web.js',
    'js',
    'web.ts',
    'ts',
    'web.tsx',
    'tsx',
    'json',
    'web.jsx',
    'jsx',
    'node',
  ],
};

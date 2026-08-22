const path = require('path');
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

module.exports = {
  jest: {
    configure: jestConfig => {
      jestConfig.moduleNameMapper = {
        ...jestConfig.moduleNameMapper,
        '^@/(.*)$': '<rootDir>/src/$1',
      };
      jestConfig.setupFilesAfterEnv = [
        ...(jestConfig.setupFilesAfterEnv || []),
        '<rootDir>/src/setup-tests.ts',
      ];
      return jestConfig;
    },
  },
  webpack: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
    plugins: [new MonacoWebpackPlugin()],
    configure: webpackConfig => {
      webpackConfig.output = {
        ...webpackConfig.output,
        publicPath: '/', // Ensures assets are loaded with relative paths in Electron
      };
      webpackConfig.plugins = webpackConfig.plugins.filter(
        plugin => plugin?.constructor?.name !== 'ForkTsCheckerWebpackPlugin',
      );
      return webpackConfig;
    },
  },
};

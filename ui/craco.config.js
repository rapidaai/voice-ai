const path = require('path');
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

module.exports = {
  webpack: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
    plugins: [new MonacoWebpackPlugin()],
    configure: webpackConfig => {
      const isElectron = process.env.IS_ELECTRON === 'true';
      webpackConfig.output = {
        ...webpackConfig.output,
        publicPath: isElectron ? './' : '/', // Ensures assets are loaded with relative paths in Electron
      };
      // Disable the minimizer
      webpackConfig.optimization = {
        ...webpackConfig.optimization,
        minimize: false,
      };
      return webpackConfig;
    },
  },
};

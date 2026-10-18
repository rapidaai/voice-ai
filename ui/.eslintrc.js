const fs = require('fs');
const path = require('path');

const prettierOptions = JSON.parse(
  fs.readFileSync(path.resolve(__dirname, '.prettierrc'), 'utf8'),
);

module.exports = {
  extends: ['react-app', 'prettier', 'prettier/react'],
  plugins: ['prettier', 'testing-library'],
  rules: {
    'react-hooks/exhaustive-deps': 'off',
    'prettier/prettier': ['error', prettierOptions],
  },
  overrides: [
    {
      files: ['**/*.ts?(x)'],
      rules: { 'prettier/prettier': ['warn', prettierOptions] },
    },
    {
      files: ['**/*.{test,spec}.{js,jsx,ts,tsx}'],
      rules: { 'testing-library/prefer-user-event': 'error' },
    },
  ],
};

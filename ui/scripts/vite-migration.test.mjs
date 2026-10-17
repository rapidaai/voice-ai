import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import test from 'node:test';

const uiRoot = resolve(import.meta.dirname, '..');

test('uses Vite for application commands without CRA or CRACO', () => {
  const packageJson = JSON.parse(
    readFileSync(resolve(uiRoot, 'package.json'), 'utf8'),
  );
  const dependencies = {
    ...packageJson.dependencies,
    ...packageJson.devDependencies,
  };

  assert.match(packageJson.scripts['start:dev'], /vite/);
  assert.match(packageJson.scripts.build, /vite build/);
  assert.equal(packageJson.scripts.test.startsWith('jest'), true);
  assert.equal(dependencies['@craco/craco'], undefined);
  assert.equal(dependencies['react-scripts'], undefined);
  assert.equal(dependencies['monaco-editor-webpack-plugin'], undefined);
  assert.equal(dependencies.tspath, undefined);
  assert.equal(dependencies['@storybook/react-webpack5'], undefined);
  assert.equal(dependencies['@storybook/react-vite'] !== undefined, true);
});

test('keeps Node and lint tooling aligned with the Vite development runtime', () => {
  const packageJson = JSON.parse(
    readFileSync(resolve(uiRoot, 'package.json'), 'utf8'),
  );

  assert.match(packageJson.devDependencies['@types/node'], /^\^22\./);
  assert.equal(packageJson.dependencies['eslint-config-react-app'], undefined);
  assert.equal(
    packageJson.devDependencies['eslint-config-react-app'],
    '7.0.1',
  );
});

test('uses the root Vite entry point and Vite client declarations', () => {
  assert.equal(existsSync(resolve(uiRoot, 'public', 'index.html')), false);
  const indexHtml = readFileSync(resolve(uiRoot, 'index.html'), 'utf8');
  assert.match(indexHtml, /src="\/src\/index\.tsx"/);

  const declarations = readFileSync(
    resolve(uiRoot, 'src', 'vite-env.d.ts'),
    'utf8',
  );
  assert.match(declarations, /vite\/client/);
  assert.equal(existsSync(resolve(uiRoot, 'src', 'react-app-env.d.ts')), false);
});

test('keeps the deployment output contract in the Vite build', () => {
  const viteConfig = readFileSync(
    resolve(uiRoot, 'vite.config.ts'),
    'utf8',
  );

  assert.match(viteConfig, /outDir:\s*'build'/);
  assert.match(viteConfig, /manifest:\s*true/);
  assert.match(viteConfig, /static\/js/);
});

test('loads the global style entry for its CSS side effects', () => {
  const appEntry = readFileSync(
    resolve(uiRoot, 'src', 'app', 'index.tsx'),
    'utf8',
  );

  assert.match(appEntry, /import ['"]@\/styles\/global-styles['"];?/);
  assert.doesNotMatch(appEntry, /import \{\} from ['"]@\/styles\/global-styles/);
});

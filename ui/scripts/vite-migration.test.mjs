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

test('keeps production NODE_ENV separate from the selected app mode', async () => {
  const { resolveConfig } = await import('vite');
  const configFile = resolve(uiRoot, 'vite.config.ts');
  const developmentConfig = await resolveConfig(
    { configFile, mode: 'development' },
    'build',
    'development',
  );
  const productionConfig = await resolveConfig(
    { configFile, mode: 'production' },
    'build',
    'production',
  );

  assert.equal(
    developmentConfig.define['process.env.NODE_ENV'],
    JSON.stringify('production'),
  );
  assert.deepEqual(JSON.parse(developmentConfig.define['process.env']), {
    MODE: 'development',
    NODE_ENV: 'production',
    VITE_MODE: 'development',
  });
  assert.equal(
    productionConfig.define['process.env.NODE_ENV'],
    JSON.stringify('production'),
  );
  assert.deepEqual(JSON.parse(productionConfig.define['process.env']), {
    MODE: 'production',
    NODE_ENV: 'production',
    VITE_MODE: 'production',
  });
});

test('registers every language used by the Monaco editors', () => {
  const monacoSetup = readFileSync(
    resolve(uiRoot, 'src', 'monaco.ts'),
    'utf8',
  );

  assert.match(monacoSetup, /language\/json\/monaco\.contribution/);
  assert.match(monacoSetup, /language\/typescript\/monaco\.contribution/);
  assert.match(
    monacoSetup,
    /basic-languages\/javascript\/javascript\.contribution/,
  );
  assert.match(
    monacoSetup,
    /basic-languages\/typescript\/typescript\.contribution/,
  );
  assert.match(monacoSetup, /language\/html\/monaco\.contribution/);
  assert.match(monacoSetup, /basic-languages\/html\/html\.contribution/);
  assert.match(monacoSetup, /basic-languages\/twig\/twig\.contribution/);
});

test('loads the global style entry for its CSS side effects', () => {
  const appEntry = readFileSync(
    resolve(uiRoot, 'src', 'app', 'index.tsx'),
    'utf8',
  );

  assert.match(appEntry, /import ['"]@\/styles\/global-styles['"];?/);
  assert.doesNotMatch(appEntry, /import \{\} from ['"]@\/styles\/global-styles/);
});

import assert from 'node:assert/strict';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

import {
  SHELL_CONTRACTS,
  THEME_CONFIG_PATHS,
  checkThemeContract,
  findHardcodedColorUtilities,
  validateLegacyThemeRemoval,
  validateShellSource,
  validateSingleSourceTheme,
  validateThemeManifest,
} from './check-theme-contract.mjs';

const checkerPath = resolve(
  dirname(fileURLToPath(import.meta.url)),
  'check-theme-contract.mjs',
);

const validManifest = {
  schemaVersion: 1,
  id: 'enterprise-default',
  brand: {
    name: 'Enterprise Voice',
    logos: {
      full: {
        light: '/brand/full-light.svg',
        dark: '/brand/full-dark.svg',
      },
      compact: {
        light: '/brand/compact-light.svg',
        dark: '/brand/compact-dark.svg',
      },
    },
    favicon: '/brand/favicon.svg',
  },
  links: {
    documentation: 'https://example.com/docs',
    source: 'https://example.com/source',
    support: 'mailto:support@example.com',
    terms: '/terms',
    privacy: '/privacy',
  },
  defaultMode: 'system',
  allowModeSelection: true,
  colors: {
    light: {
      primary: '#2563eb',
      primaryHover: '#1d4ed8',
      primaryActive: '#1e40af',
      onPrimary: '#ffffff',
    },
    dark: {
      primary: '#60a5fa',
      primaryHover: '#93c5fd',
      primaryActive: '#bfdbfe',
      onPrimary: '#161616',
    },
  },
};

const writeFixtureFile = (repoRoot, relativePath, contents) => {
  const filePath = resolve(repoRoot, relativePath);
  mkdirSync(dirname(filePath), { recursive: true });
  writeFileSync(filePath, contents);
};

const createFixtureRepo = context => {
  const repoRoot = mkdtempSync(resolve(tmpdir(), 'theme-contract-'));
  context.after(() => rmSync(repoRoot, { recursive: true, force: true }));

  for (const [sourcePath, requiredTokens] of Object.entries(SHELL_CONTRACTS)) {
    writeFixtureFile(
      repoRoot,
      sourcePath,
      `export const shellClassName = ${JSON.stringify(requiredTokens.join(' '))};\n`,
    );
  }

  for (const configPath of THEME_CONFIG_PATHS) {
    writeFixtureFile(
      repoRoot,
      configPath,
      `${JSON.stringify({ theme: validManifest }, null, 2)}\n`,
    );
  }
  writeFixtureFile(repoRoot, 'ui/src/index.tsx', 'export {};\n');
  writeFixtureFile(repoRoot, 'ui/public/index.html', '<div id="root"></div>\n');

  return repoRoot;
};

test('accepts a valid enterprise theme repository', context => {
  const repoRoot = createFixtureRepo(context);

  assert.deepEqual(checkThemeContract(repoRoot), []);
});

test('rejects the legacy context and legacy provider or hook references', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/src/context/dark-mode-context.tsx',
    'export const DarkModeProvider = () => null;\n',
  );
  writeFixtureFile(
    repoRoot,
    'ui/src/legacy-consumer.tsx',
    "import { DarkModeContext, DarkModeProvider, useDarkMode } from './context/dark-mode-context';\n",
  );

  assert.deepEqual(validateLegacyThemeRemoval(repoRoot), [
    'ui/src/context/dark-mode-context.tsx: legacy dark-mode context must not exist',
    'ui/src/legacy-consumer.tsx: reference to legacy DarkModeContext is not allowed',
    'ui/src/legacy-consumer.tsx: reference to legacy DarkModeProvider is not allowed',
    'ui/src/legacy-consumer.tsx: reference to legacy dark-mode-context is not allowed',
    'ui/src/legacy-consumer.tsx: reference to legacy useDarkMode is not allowed',
  ]);
});

test('rejects a shell source missing a required semantic utility', () => {
  assert.deepEqual(
    validateShellSource(
      'const classes = "bg-shell text-foreground";',
      ['bg-shell', 'border-border-subtle'],
      'ui/src/shell.tsx',
    ),
    [
      'ui/src/shell.tsx: missing required semantic utility border-border-subtle',
    ],
  );
});

test('rejects variant hardcoded palettes while allowing non-color utilities', () => {
  const source = 'className="text-[10px] border-b dark:bg-gray-900"';

  assert.deepEqual(findHardcodedColorUtilities(source), ['dark:bg-gray-900']);
  assert.deepEqual(validateShellSource(source, [], 'ui/src/shell.tsx'), [
    'ui/src/shell.tsx: hardcoded structural color utility is not allowed: dark:bg-gray-900',
  ]);
});

test('rejects a malformed and incomplete enterprise theme manifest', () => {
  const manifestPath = 'ui/src/configs/config.development.json#theme';
  const diagnostics = validateThemeManifest(
    {
      schemaVersion: 2,
      id: ' ',
      brand: { name: '' },
      links: { documentation: ['javascript', 'alert(1)'].join(':') },
      defaultMode: 'sepia',
      allowModeSelection: 'yes',
      colors: {
        light: { primary: 'red; background: black' },
        dark: {},
      },
    },
    manifestPath,
  );

  assert.ok(
    diagnostics.includes(`${manifestPath}: schemaVersion must equal 1`),
  );
  assert.ok(
    diagnostics.includes(`${manifestPath}: id must be a nonempty string`),
  );
  assert.ok(
    diagnostics.includes(
      `${manifestPath}: links.documentation must be root-relative or use https/mailto`,
    ),
  );
  assert.ok(
    diagnostics.includes(
      `${manifestPath}: colors.light.primary must be a six-digit hexadecimal color`,
    ),
  );
  assert.ok(diagnostics.length > 10);
});

test('returns deterministic, sorted diagnostics', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/src/configs/config.development.json',
    JSON.stringify({ theme: { schemaVersion: 0 } }),
  );
  writeFixtureFile(
    repoRoot,
    'ui/src/app/components/aside/index.tsx',
    'const classes = "bg-white";\n',
  );
  writeFixtureFile(
    repoRoot,
    'ui/src/z-legacy.ts',
    'export const mode = useDarkMode();\n',
  );

  const firstResult = checkThemeContract(repoRoot);
  const secondResult = checkThemeContract(repoRoot);

  assert.deepEqual(firstResult, [...firstResult].sort());
  assert.deepEqual(secondResult, firstResult);
  assert.ok(firstResult.length > 1);
});

test('CLI exits nonzero and reports diagnostics for an invalid repository', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/src/configs/config.development.json',
    '{ invalid json',
  );

  const result = spawnSync(process.execPath, [checkerPath, repoRoot], {
    encoding: 'utf8',
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Theme contract check failed:/);
  assert.match(result.stderr, /could not parse UI config/);
});

test('rejects duplicate theme sources and inline bootstrap scripts', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(repoRoot, 'ui/public/theme.json', '{}');
  writeFixtureFile(repoRoot, 'ui/public/index.html', '<script>boot()</script>');

  assert.deepEqual(validateSingleSourceTheme(repoRoot), [
    'ui/public/index.html: inline bootstrap scripts are not allowed; use CONFIG.theme',
    'ui/public/theme.json: standalone theme manifest must not exist; use CONFIG.theme',
  ]);
});

test('allows external script elements in the public index', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/public/index.html',
    '<script src="/static/application.js"></script>',
  );

  assert.deepEqual(validateSingleSourceTheme(repoRoot), []);
});

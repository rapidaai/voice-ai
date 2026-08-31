import assert from 'node:assert/strict';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

import {
  BRAND_LITERAL_ALLOWLIST,
  SHELL_CONTRACTS,
  THEME_CONFIG_PATHS,
  checkThemeContract,
  findHardcodedBrandLiterals,
  findHardcodedColorUtilities,
  validateBrandLiterals,
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

test('accepts arbitrary values and partial keys inside a theme manifest', () => {
  const manifestPath = 'ui/src/configs/config.development.json#theme';
  const manifest = {
    brand: {
      logos: { full: { light: 'any asset value' } },
    },
    links: { documentation: 'custom://documentation' },
    defaultMode: 'sepia',
    allowModeSelection: 'client-controlled',
    colors: { light: { primary: 'brand(primary)' } },
  };

  assert.deepEqual(validateThemeManifest(manifest, manifestPath), []);
  assert.deepEqual(validateThemeManifest({}, manifestPath), []);
});

test('rejects a missing or non-object theme manifest', () => {
  const manifestPath = 'ui/src/configs/config.development.json#theme';
  const expected = [`${manifestPath}: manifest must be a JSON object`];

  assert.deepEqual(validateThemeManifest(undefined, manifestPath), expected);
  assert.deepEqual(validateThemeManifest(null, manifestPath), expected);
  assert.deepEqual(validateThemeManifest([], manifestPath), expected);
  assert.deepEqual(validateThemeManifest('theme', manifestPath), expected);
});

test('returns deterministic, sorted diagnostics', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/src/configs/config.development.json',
    JSON.stringify({}),
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

test('finds hardcoded brand literals in renderable UI source', () => {
  const source = `
    const title = "Rapida AI";
    const lower = "supported by rapida";
    const docs = "https://doc.rapida.ai/assistants";
    const cdn = "https://cdn-01.rapida.ai/script.js";
    const support = "support@rapida.ai";
  `;

  assert.deepEqual(findHardcodedBrandLiterals(source), [
    'Rapida AI',
    'cdn-01.rapida.ai',
    'doc.rapida.ai',
    'rapida',
    'support@rapida.ai',
  ]);
});

test('rejects brand literals from scanned UI files', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/src/app/pages/dashboard.tsx',
    'export const title = "Rapida AI";\n',
  );
  writeFixtureFile(
    repoRoot,
    'ui/src/app/pages/dashboard.test.tsx',
    'expect("Rapida AI").toBeTruthy();\n',
  );

  assert.deepEqual(validateBrandLiterals(repoRoot), [
    'ui/src/app/pages/dashboard.tsx: hardcoded brand literal is not allowed: Rapida AI',
  ]);
});

test('allowlisted files still reject unlisted brand literals', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/src/app/pages/main/web-dashboard/index.tsx',
    `
      export const feed = "https://cdn-01.rapida.ai/web/feed.json";
      export const title = "Rapida AI";
    `,
  );

  assert.deepEqual(validateBrandLiterals(repoRoot), [
    'ui/src/app/pages/main/web-dashboard/index.tsx: hardcoded brand literal is not allowed: Rapida AI',
  ]);
});

test('documents every brand literal allowlist entry with a reason', () => {
  for (const [sourcePath, entry] of Object.entries(BRAND_LITERAL_ALLOWLIST)) {
    assert.match(sourcePath, /^ui\/src\/app\/.+\.(?:ts|tsx)$/);
    assert.ok(Array.isArray(entry.literals));
    assert.ok(entry.literals.length > 0);
    assert.equal(typeof entry.reason, 'string');
    assert.ok(entry.reason.length > 20);
  }
});

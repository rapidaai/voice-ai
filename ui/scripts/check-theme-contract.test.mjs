import assert from 'node:assert/strict';
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

import {
  BRAND_LITERAL_ALLOWLIST,
  EXTERNAL_STYLE_ADAPTERS,
  SHELL_CONTRACTS,
  TAILWIND_PROSE_MAPPINGS,
  THEME_CONTRACT,
  THEME_CONFIG_PATHS,
  THEME_STYLE_EXCEPTIONS,
  checkThemeContract,
  collectConsumedCarbonProperties,
  findHardcodedBrandLiterals,
  findHardcodedColorUtilities,
  findThemeStyleViolations,
  validateBrandLiterals,
  validateCarbonPropertyCoverage,
  validateExternalStyleAdapters,
  validateLegacyThemeRemoval,
  validateShellSource,
  validateSingleSourceTheme,
  validateThemeManifest,
  validateThemeStyles,
  validateTailwindThemeContract,
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
  writeFixtureFile(repoRoot, 'ui/index.html', '<div id="root"></div>\n');
  const adapterImports = [];
  for (const [sourcePath, adapter] of Object.entries(EXTERNAL_STYLE_ADAPTERS)) {
    writeFixtureFile(
      repoRoot,
      sourcePath,
      `${adapter.selectors.join(',\n')} { color: var(--cds-text-primary); }\n`,
    );
    adapterImports.push(
      `import './externals/${sourcePath.split('/').at(-1)}';`,
    );
  }
  writeFixtureFile(
    repoRoot,
    'ui/src/styles/global-styles.ts',
    `${adapterImports.join('\n')}\n`,
  );
  const brandMappings = Object.entries(
    THEME_CONTRACT.carbonProperties.brandOverrides,
  ).flatMap(([field, properties]) =>
    properties.map(
      property =>
        `${property}: var(${THEME_CONTRACT.colorFields[field].property});`,
    ),
  );
  const presetReferences = THEME_CONTRACT.carbonProperties.presetOwned.map(
    property => `color: var(${property});`,
  );
  writeFixtureFile(
    repoRoot,
    'ui/src/styles/carbon.scss',
    `:root {\n${[...brandMappings, ...presetReferences].join('\n')}\n}\n`,
  );
  writeFixtureFile(
    repoRoot,
    'ui/src/styles/tailwind.css',
    `@theme inline { --color-*: initial; }\n.prose {\n${Object.entries(
      TAILWIND_PROSE_MAPPINGS,
    )
      .map(([property, value]) => `  ${property}: ${value};`)
      .join('\n')}\n}\n`,
  );

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

test('rejects directional border palette and arbitrary color utilities', () => {
  const source =
    'className="border-l-red-500 border-b-[var(--cds-border-strong)]"';

  assert.deepEqual(
    findThemeStyleViolations(source).map(({ rule }) => rule),
    [
      'arbitrary-color-utility',
      'direct-carbon-color',
      'literal-palette-utility',
    ],
  );
});

test('rejects typography palettes and dark inversion', () => {
  const source = 'className="prose prose-slate dark:prose-invert"';

  assert.deepEqual(
    findThemeStyleViolations(source).map(({ rule }) => rule),
    ['dark-typography-variant', 'typography-palette'],
  );
});

test('rejects literal and dark placeholder colors', () => {
  const source =
    'className="placeholder-gray-400 dark:placeholder-gray-600"';

  assert.deepEqual(
    findThemeStyleViolations(source).map(({ rule }) => rule),
    [
      'dark-color-variant',
      'literal-palette-utility',
      'literal-palette-utility',
    ],
  );
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

test('requires every machine-owned color field and Carbon preset in version 2', () => {
  const manifestPath = 'ui/src/configs/config.development.json#theme';
  const manifest = structuredClone(validManifest);
  manifest.schemaVersion = 2;
  manifest.colors.light.carbonTheme = 'white';
  manifest.colors.dark.carbonTheme = 'g100';
  delete manifest.colors.dark.primaryActive;
  manifest.colors.light.primary = 'var(--cds-button-primary)';

  assert.deepEqual(validateThemeManifest(manifest, manifestPath), [
    `${manifestPath}.colors.dark.primaryActive: required non-empty string`,
    `${manifestPath}.colors.light.primary: internal Carbon and brand property references are not allowed`,
  ]);
});

test('classifies every consumed Carbon property and verifies brand mappings', context => {
  const repoRoot = createFixtureRepo(context);

  writeFixtureFile(
    repoRoot,
    'ui/src/app/new-carbon-consumer.tsx',
    "export const color = 'var(--cds-future-color)';\n",
  );

  assert.equal(
    collectConsumedCarbonProperties(repoRoot).includes('--cds-future-color'),
    true,
  );
  assert.deepEqual(validateCarbonPropertyCoverage(repoRoot), [
    'ui/src/theme/theme-contract.json: --cds-future-color is consumed but has no declared owner',
  ]);

  const adapterPath = resolve(repoRoot, 'ui/src/styles/carbon.scss');
  writeFileSync(
    adapterPath,
    readFileSync(adapterPath, 'utf8').replace(
      '--cds-button-primary: var(--brand-primary);',
      '--cds-button-primary: var(--brand-primary-hover);',
    ),
  );
  assert.deepEqual(validateCarbonPropertyCoverage(repoRoot), [
    'ui/src/styles/carbon.scss: --cds-button-primary must map to --brand-primary',
    'ui/src/theme/theme-contract.json: --cds-future-color is consumed but has no declared owner',
  ]);
});

test('requires Tailwind to disable its built-in color namespace', context => {
  const repoRoot = createFixtureRepo(context);
  const themePath = resolve(repoRoot, 'ui/src/styles/tailwind.css');
  writeFileSync(
    themePath,
    readFileSync(themePath, 'utf8').replace('--color-*: initial;', ''),
  );

  assert.deepEqual(validateTailwindThemeContract(repoRoot), [
    'ui/src/styles/tailwind.css: built-in color namespace must be disabled with --color-*: initial',
  ]);
});

test('requires every Typography color role to map to Carbon', context => {
  const repoRoot = createFixtureRepo(context);
  const themePath = resolve(repoRoot, 'ui/src/styles/tailwind.css');
  writeFileSync(
    themePath,
    readFileSync(themePath, 'utf8').replace(
      '--tw-prose-bullets: var(--cds-icon-secondary);',
      '--tw-prose-bullets: oklch(50% 0 0);',
    ),
  );

  assert.deepEqual(validateTailwindThemeContract(repoRoot), [
    'ui/src/styles/tailwind.css: --tw-prose-bullets must map to var(--cds-icon-secondary)',
  ]);
});

test('finds every forbidden component color construct', () => {
  const source = `
    import styles from './card.module.css';
    const classes = 'bg-red-500 dark:bg-layer text-[#fff] border-[var(--cds-border-subtle-01)]';
    const literal = 'oklch(50% 0.2 20)';
    const direct = 'var(--cds-text-primary)';
    const element = <div style={{ color: 'red' }} />;
  `;

  assert.deepEqual(
    findThemeStyleViolations(source).map(({ rule }) => rule),
    [
      'arbitrary-color-utility',
      'arbitrary-color-utility',
      'component-stylesheet',
      'dark-color-variant',
      'direct-carbon-color',
      'direct-carbon-color',
      'inline-color',
      'literal-color',
      'literal-color',
      'literal-palette-utility',
    ],
  );
});

test('ignores non-color inline values and typography variants', () => {
  const source = `
    type Swatch = { color: string };
    const className = 'dark:text-base';
    const style = { background: 'transparent' };
  `;

  assert.deepEqual(findThemeStyleViolations(source), []);
});

test('theme style exceptions are exact and do not allow an extra occurrence', () => {
  for (const [sourcePath, rules] of Object.entries(THEME_STYLE_EXCEPTIONS)) {
    const source = Object.values(rules)
      .flatMap(exception => exception.matches)
      .join('\n');
    assert.deepEqual(findThemeStyleViolations(source, sourcePath), []);

    for (const [rule, exception] of Object.entries(rules)) {
      const extra = findThemeStyleViolations(
        `${source}\n${exception.matches[0]}`,
        sourcePath,
      );
      assert.equal(
        extra.filter(violation => violation.rule === rule).length,
        1,
        `${sourcePath} ${rule}`,
      );
    }
  }
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
    'ui/src/app/components/aside.tsx',
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

test('scans renderable provider components for theme violations', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/src/app/components/domain/providers/config-renderer.tsx',
    'export const className = "border-gray-200 dark:border-gray-700";\n',
  );

  assert.deepEqual(validateThemeStyles(repoRoot), [
    'ui/src/app/components/domain/providers/config-renderer.tsx: dark-color-variant is not allowed: dark:border-gray-700',
    'ui/src/app/components/domain/providers/config-renderer.tsx: literal-palette-utility is not allowed: "border-gray-200',
    'ui/src/app/components/domain/providers/config-renderer.tsx: literal-palette-utility is not allowed: dark:border-gray-700',
  ]);
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
  writeFixtureFile(repoRoot, 'ui/index.html', '<script>boot()</script>');

  assert.deepEqual(validateSingleSourceTheme(repoRoot), [
    'ui/index.html: inline bootstrap scripts are not allowed; use CONFIG.theme',
    'ui/public/theme.json: standalone theme manifest must not exist; use CONFIG.theme',
  ]);
});

test('allows external script elements in the application index', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/index.html',
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

test('brand exceptions require exact constructs and reject extra occurrences', context => {
  const repoRoot = createFixtureRepo(context);
  writeFixtureFile(
    repoRoot,
    'ui/src/app/components/domain/providers/embedding/openai/constants.ts',
    `
      export const models = [
        { owner: 'rapida', id: 'one' },
        { owner: 'rapida', id: 'two' },
        { owner: 'rapida', id: 'three' },
      ];
      export const title = 'Rapida AI';
    `,
  );

  assert.deepEqual(validateBrandLiterals(repoRoot), [
    'ui/src/app/components/domain/providers/embedding/openai/constants.ts: hardcoded brand literal is not allowed: Rapida AI',
  ]);

  writeFixtureFile(
    repoRoot,
    'ui/src/app/components/domain/providers/embedding/openai/constants.ts',
    `
      export const models = [
        { owner: 'rapida', id: 'one' },
        { owner: 'rapida', id: 'two' },
      ];
    `,
  );
  assert.deepEqual(validateBrandLiterals(repoRoot), [
    "ui/src/app/components/domain/providers/embedding/openai/constants.ts: allowlisted brand construct is missing: owner: 'rapida',",
  ]);
});

test('documents every brand literal allowlist entry with a reason', () => {
  for (const [sourcePath, entry] of Object.entries(BRAND_LITERAL_ALLOWLIST)) {
    assert.match(sourcePath, /^ui\/src\/app\/.+\.(?:ts|tsx)$/);
    assert.ok(Array.isArray(entry.matches));
    assert.ok(entry.matches.length > 0);
    assert.equal(typeof entry.reason, 'string');
    assert.ok(entry.reason.length > 20);
  }
});

test('documents and verifies every external style adapter', context => {
  const repoRoot = createFixtureRepo(context);
  const imports = [];
  for (const [sourcePath, adapter] of Object.entries(EXTERNAL_STYLE_ADAPTERS)) {
    assert.ok(adapter.reason.length > 20);
    assert.ok(adapter.selectors.length > 0);
    writeFixtureFile(
      repoRoot,
      sourcePath,
      `${adapter.selectors.join(',\n')} { color: var(--cds-text-primary); }\n`,
    );
    imports.push(`import './externals/${sourcePath.split('/').at(-1)}';`);
  }
  writeFixtureFile(
    repoRoot,
    'ui/src/styles/global-styles.ts',
    `${imports.join('\n')}\n`,
  );

  assert.deepEqual(validateExternalStyleAdapters(repoRoot), []);

  writeFixtureFile(
    repoRoot,
    'ui/src/styles/externals/unlisted.css',
    '.vendor { color: red; }\n',
  );
  assert.deepEqual(validateExternalStyleAdapters(repoRoot), [
    'ui/src/styles/externals/unlisted.css: external style adapter is not allowlisted',
  ]);
});

test('external adapters reject unregistered selectors and named colors', context => {
  const repoRoot = createFixtureRepo(context);
  const [sourcePath, adapter] = Object.entries(EXTERNAL_STYLE_ADAPTERS)[0];
  writeFixtureFile(
    repoRoot,
    sourcePath,
    `${adapter.selectors.join(',\n')},\n.unreviewed-vendor-state { color: red; }\n`,
  );

  assert.deepEqual(validateExternalStyleAdapters(repoRoot), [
    `${sourcePath}: external selector is not allowlisted: .unreviewed-vendor-state (1/0)`,
    `${sourcePath}: literal colors are not allowed in external adapters`,
  ]);
});

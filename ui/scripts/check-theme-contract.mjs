#!/usr/bin/env node

import { readdirSync, readFileSync, statSync } from 'node:fs';
import { dirname, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

export const DEFAULT_REPO_ROOT = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '..',
  '..',
);

export const SHELL_CONTRACTS = Object.freeze({
  'ui/src/app/components/aside/index.tsx': [
    'bg-shell',
    'text-foreground',
    'border-border-subtle',
  ],
  'ui/src/app/components/navigation/actionable-header/index.tsx': [
    'bg-shell',
    'text-foreground',
    'border-border-subtle',
  ],
  'ui/src/app/components/navigation/header/index.tsx': [
    'bg-shell',
    'border-border-subtle',
  ],
  'ui/src/app/components/navigation/sidebar/index.tsx': [
    'border-border-subtle',
    'text-muted',
    'text-foreground',
    'bg-layer-hover',
  ],
});

export const FORBIDDEN_THEME_SYMBOLS = Object.freeze([
  'DarkModeContext',
  'DarkModeProvider',
  'useDarkMode',
]);

const THEME_CONTEXT_PATH = 'ui/src/context/dark-mode-context.tsx';
const LEGACY_THEME_MANIFEST_PATH = 'ui/public/theme.json';
const PUBLIC_INDEX_PATH = 'ui/public/index.html';
export const THEME_CONFIG_PATHS = Object.freeze([
  'ui/src/configs/config.development.json',
  'ui/src/configs/config.production.json',
  'docker/ui/config.community.json',
  'docker/ui/config.enterprise.json',
  'docker/ui/config.local.json',
  'docker/ui/config.local-knowledge.json',
]);
const GENERATED_STYLES_DIRECTORY = 'ui/src/styles/generated';
const BRAND_LITERAL_SCAN_DIRECTORIES = Object.freeze([
  'ui/src/app',
  'ui/src/theme',
]);
const BRAND_LITERAL_SCAN_FILES = Object.freeze(['ui/public/LLMs.txt']);
export const BRAND_LITERAL_ALLOWLIST = Object.freeze({
  'ui/src/app/components/Icon/Rapida.tsx': {
    literals: ['Rapida'],
    reason:
      'legacy icon component definition, not tenant-visible branding by itself',
  },
  'ui/src/app/components/Icon/RapidaText.tsx': {
    literals: ['Rapida'],
    reason:
      'legacy icon component definition, not tenant-visible branding by itself',
  },
  'ui/src/app/components/integration-document/endpoint-integration.tsx': {
    literals: ['Rapida', 'rapida'],
    reason:
      'SDK package and client class names are external developer API identifiers',
  },
  'ui/src/app/pages/static-pages/terms.tsx': {
    literals: ['Rapida.AI', 'support@rapida.ai'],
    reason:
      'legacy direct legal route content remains outside tenant-visible navigation',
  },
  'ui/src/app/pages/static-pages/privacy.tsx': {
    literals: ['Rapida', 'Rapida.AI', 'support@rapida.ai'],
    reason:
      'legacy direct legal route content remains outside tenant-visible navigation',
  },
  'ui/src/app/pages/main/web-dashboard/index.tsx': {
    literals: ['cdn-01.rapida.ai'],
    reason: 'dashboard content feed URL is a technical asset source',
  },
  'ui/src/app/pages/assistant/actions/create-assistant/create-agentkit.tsx': {
    literals: ['Rapida'],
    reason:
      'AgentKit and orchestration names are deployment product identifiers',
  },
  'ui/src/app/pages/assistant/actions/create-assistant-version/create-agent-kit-version.tsx':
    {
      literals: ['Rapida'],
      reason: 'AgentKit name is a deployment product identifier',
    },
  'ui/src/app/pages/assistant/view/conversations/conversation-messages.helpers.ts':
    {
      literals: ['rapida'],
      reason:
        'legacy conversation role value is mapped to a tenant-neutral display label',
    },
  'ui/src/app/components/base/modal/assistant-instruction-modal/index.tsx': {
    literals: ['cdn-01.rapida.ai'],
    reason: 'web widget script URL is a deployment snippet asset source',
  },
  'ui/src/app/components/base/modal/assistant-web-widget-deployment-modal/index.tsx':
    {
      literals: ['cdn-01.rapida.ai'],
      reason: 'web widget script URL is a deployment snippet asset source',
    },
});
const HARD_CODED_PALETTES = Object.freeze([
  'white',
  'black',
  'gray',
  'slate',
  'zinc',
  'neutral',
  'stone',
]);

const HARD_CODED_COLOR_UTILITY = new RegExp(
  String.raw`(?:^|[^A-Za-z0-9_-])((?:[A-Za-z0-9_-]+:)*!?(?:bg|text|border|ring|divide)-(?:${HARD_CODED_PALETTES.join(
    '|',
  )})(?:-[0-9]{1,3})?(?:\/[0-9]{1,3})?!?)(?=$|[^A-Za-z0-9_-])`,
  'g',
);

const BRAND_LITERAL_PATTERNS = Object.freeze([
  /\bRapida(?:\s+AI|\.AI|AI| Assistant)?\b/g,
  /(?<![-/.@])\brapida(?:\s+ai| assistant)?\b(?![.-])/g,
  /\b(?:doc|docs|www|blog)\.rapida\.ai\b/g,
  /\bcdn-01\.rapida\.ai\b/g,
  /\b[A-Za-z0-9._%+-]+@rapida\.ai\b/g,
]);

const toPosixPath = path => path.split(sep).join('/');

const isPlainObject = value =>
  value !== null && typeof value === 'object' && !Array.isArray(value);

const pushDiagnostic = (diagnostics, path, message) => {
  diagnostics.push(`${path}: ${message}`);
};

export function validateThemeManifest(manifest, manifestPath = 'CONFIG.theme') {
  return isPlainObject(manifest)
    ? []
    : [`${manifestPath}: manifest must be a JSON object`];
}

export function findHardcodedColorUtilities(source) {
  const utilities = new Set();
  HARD_CODED_COLOR_UTILITY.lastIndex = 0;

  for (const match of source.matchAll(HARD_CODED_COLOR_UTILITY)) {
    utilities.add(match[1]);
  }

  return [...utilities].sort();
}

export function validateShellSource(source, requiredTokens, sourcePath) {
  const diagnostics = [];

  for (const token of requiredTokens) {
    const tokenPattern = new RegExp(
      String.raw`(?:^|[^A-Za-z0-9_-])${token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}!?(?=$|[^A-Za-z0-9_-])`,
    );
    if (!tokenPattern.test(source)) {
      pushDiagnostic(
        diagnostics,
        sourcePath,
        `missing required semantic utility ${token}`,
      );
    }
  }

  for (const utility of findHardcodedColorUtilities(source)) {
    pushDiagnostic(
      diagnostics,
      sourcePath,
      `hardcoded structural color utility is not allowed: ${utility}`,
    );
  }

  return diagnostics.sort();
}

const walkFiles = directory => {
  const files = [];
  for (const entry of readdirSync(directory, { withFileTypes: true }).sort(
    (a, b) => a.name.localeCompare(b.name),
  )) {
    const entryPath = resolve(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...walkFiles(entryPath));
    } else if (entry.isFile()) {
      files.push(entryPath);
    }
  }
  return files;
};

const isBrandScanFile = sourcePath => {
  if (
    sourcePath.includes('/__tests__/') ||
    sourcePath.includes('/components/providers/') ||
    /\.(?:test|spec)\.[cm]?[jt]sx?$/.test(sourcePath)
  ) {
    return false;
  }

  return /\.(?:tsx?|txt)$/.test(sourcePath);
};

const collectBrandScanFiles = repoRoot => {
  const files = [];

  for (const directory of BRAND_LITERAL_SCAN_DIRECTORIES) {
    const directoryPath = resolve(repoRoot, directory);
    try {
      if (statSync(directoryPath).isDirectory()) {
        files.push(...walkFiles(directoryPath));
      }
    } catch (error) {
      if (error?.code !== 'ENOENT') throw error;
    }
  }

  for (const file of BRAND_LITERAL_SCAN_FILES) {
    const filePath = resolve(repoRoot, file);
    try {
      if (statSync(filePath).isFile()) files.push(filePath);
    } catch (error) {
      if (error?.code !== 'ENOENT') throw error;
    }
  }

  return files
    .map(filePath => ({
      filePath,
      sourcePath: toPosixPath(relative(repoRoot, filePath)),
    }))
    .filter(({ sourcePath }) => isBrandScanFile(sourcePath));
};

export function findHardcodedBrandLiterals(source) {
  const literals = new Set();

  for (const pattern of BRAND_LITERAL_PATTERNS) {
    pattern.lastIndex = 0;
    for (const match of source.matchAll(pattern)) {
      literals.add(match[0]);
    }
  }

  return [...literals].sort();
}

export function validateLegacyThemeRemoval(repoRoot = DEFAULT_REPO_ROOT) {
  const diagnostics = [];
  const contextPath = resolve(repoRoot, THEME_CONTEXT_PATH);

  try {
    if (statSync(contextPath).isFile()) {
      pushDiagnostic(
        diagnostics,
        THEME_CONTEXT_PATH,
        'legacy dark-mode context must not exist',
      );
    }
  } catch (error) {
    if (error?.code !== 'ENOENT') throw error;
  }

  const sourceRoot = resolve(repoRoot, 'ui/src');
  const generatedStylesRoot = resolve(repoRoot, GENERATED_STYLES_DIRECTORY);
  for (const filePath of walkFiles(sourceRoot)) {
    if (
      filePath === contextPath ||
      filePath === generatedStylesRoot ||
      filePath.startsWith(`${generatedStylesRoot}${sep}`)
    ) {
      continue;
    }

    const source = readFileSync(filePath, 'utf8');
    const sourcePath = toPosixPath(relative(repoRoot, filePath));
    if (source.includes('dark-mode-context')) {
      pushDiagnostic(
        diagnostics,
        sourcePath,
        'reference to legacy dark-mode-context is not allowed',
      );
    }
    for (const symbol of FORBIDDEN_THEME_SYMBOLS) {
      if (new RegExp(String.raw`\b${symbol}\b`).test(source)) {
        pushDiagnostic(
          diagnostics,
          sourcePath,
          `reference to legacy ${symbol} is not allowed`,
        );
      }
    }
  }

  return diagnostics.sort();
}

export function validateShellFiles(repoRoot = DEFAULT_REPO_ROOT) {
  const diagnostics = [];

  for (const [sourcePath, requiredTokens] of Object.entries(SHELL_CONTRACTS)) {
    try {
      const source = readFileSync(resolve(repoRoot, sourcePath), 'utf8');
      diagnostics.push(
        ...validateShellSource(source, requiredTokens, sourcePath),
      );
    } catch (error) {
      pushDiagnostic(
        diagnostics,
        sourcePath,
        error?.code === 'ENOENT'
          ? 'required shell file is missing'
          : `could not read shell file: ${error.message}`,
      );
    }
  }

  return diagnostics.sort();
}

export function validateThemeConfigFiles(repoRoot = DEFAULT_REPO_ROOT) {
  const diagnostics = [];

  for (const configPath of THEME_CONFIG_PATHS) {
    try {
      const source = readFileSync(resolve(repoRoot, configPath), 'utf8');
      const config = JSON.parse(source);
      diagnostics.push(
        ...validateThemeManifest(config.theme, `${configPath}#theme`),
      );
    } catch (error) {
      pushDiagnostic(
        diagnostics,
        configPath,
        error?.code === 'ENOENT'
          ? 'deployable UI config is missing'
          : `could not parse UI config: ${error.message}`,
      );
    }
  }

  return diagnostics.sort();
}

export function validateSingleSourceTheme(repoRoot = DEFAULT_REPO_ROOT) {
  const diagnostics = [];

  try {
    if (statSync(resolve(repoRoot, LEGACY_THEME_MANIFEST_PATH)).isFile()) {
      pushDiagnostic(
        diagnostics,
        LEGACY_THEME_MANIFEST_PATH,
        'standalone theme manifest must not exist; use CONFIG.theme',
      );
    }
  } catch (error) {
    if (error?.code !== 'ENOENT') throw error;
  }

  try {
    const source = readFileSync(resolve(repoRoot, PUBLIC_INDEX_PATH), 'utf8');
    const scriptTags = source.match(/<script\b[^>]*>/gi) ?? [];
    if (scriptTags.some(scriptTag => !/\ssrc\s*=/i.test(scriptTag))) {
      pushDiagnostic(
        diagnostics,
        PUBLIC_INDEX_PATH,
        'inline bootstrap scripts are not allowed; use CONFIG.theme',
      );
    }
  } catch (error) {
    pushDiagnostic(
      diagnostics,
      PUBLIC_INDEX_PATH,
      error?.code === 'ENOENT'
        ? 'public index is missing'
        : `could not read public index: ${error.message}`,
    );
  }

  return diagnostics.sort();
}

export function validateBrandLiterals(repoRoot = DEFAULT_REPO_ROOT) {
  const diagnostics = [];

  for (const { filePath, sourcePath } of collectBrandScanFiles(repoRoot)) {
    const allowedLiterals = new Set(
      BRAND_LITERAL_ALLOWLIST[sourcePath]?.literals ?? [],
    );

    const source = readFileSync(filePath, 'utf8');
    for (const literal of findHardcodedBrandLiterals(source)) {
      if (allowedLiterals.has(literal)) continue;

      pushDiagnostic(
        diagnostics,
        sourcePath,
        `hardcoded brand literal is not allowed: ${literal}`,
      );
    }
  }

  return diagnostics.sort();
}

export function checkThemeContract(repoRoot = DEFAULT_REPO_ROOT) {
  return [
    ...validateLegacyThemeRemoval(repoRoot),
    ...validateShellFiles(repoRoot),
    ...validateThemeConfigFiles(repoRoot),
    ...validateSingleSourceTheme(repoRoot),
    ...validateBrandLiterals(repoRoot),
  ].sort();
}

export function runCli(argv = process.argv.slice(2)) {
  const repoRoot = argv[0] ? resolve(argv[0]) : DEFAULT_REPO_ROOT;
  const diagnostics = checkThemeContract(repoRoot);

  if (diagnostics.length > 0) {
    console.error('Theme contract check failed:');
    for (const diagnostic of diagnostics) {
      console.error(`- ${diagnostic}`);
    }
    return 1;
  }

  console.log('Theme contract check passed.');
  return 0;
}

const isDirectExecution =
  process.argv[1] !== undefined &&
  resolve(process.argv[1]) === fileURLToPath(import.meta.url);

if (isDirectExecution) {
  process.exitCode = runCli();
}

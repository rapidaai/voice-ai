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
const REQUIRED_LINKS = Object.freeze([
  'documentation',
  'source',
  'support',
  'terms',
  'privacy',
]);
const REQUIRED_COLORS = Object.freeze([
  'primary',
  'primaryHover',
  'primaryActive',
  'onPrimary',
]);
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

const toPosixPath = path => path.split(sep).join('/');

const isNonEmptyString = value =>
  typeof value === 'string' && value.trim().length > 0;

const HEX_COLOR_PATTERN = /^#[0-9a-f]{6}$/i;

const isThemeColor = value =>
  isNonEmptyString(value) && HEX_COLOR_PATTERN.test(value);

const getRelativeLuminance = hexColor => {
  const channels = [1, 3, 5].map(offset =>
    Number.parseInt(hexColor.slice(offset, offset + 2), 16),
  );
  const [red, green, blue] = channels.map(channel => {
    const normalized = channel / 255;
    return normalized <= 0.04045
      ? normalized / 12.92
      : ((normalized + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue;
};

const getContrastRatio = (first, second) => {
  const firstLuminance = getRelativeLuminance(first);
  const secondLuminance = getRelativeLuminance(second);
  return (
    (Math.max(firstLuminance, secondLuminance) + 0.05) /
    (Math.min(firstLuminance, secondLuminance) + 0.05)
  );
};

const hasAllowedProtocol = (value, protocols) => {
  if (!isNonEmptyString(value)) return false;
  if (value.startsWith('/') && !value.startsWith('//')) return true;

  try {
    return protocols.includes(new URL(value).protocol);
  } catch {
    return false;
  }
};

const isAppLink = value => hasAllowedProtocol(value, ['https:', 'mailto:']);

const isAssetLink = value => hasAllowedProtocol(value, ['https:']);

const isPlainObject = value =>
  value !== null && typeof value === 'object' && !Array.isArray(value);

const pushDiagnostic = (diagnostics, path, message) => {
  diagnostics.push(`${path}: ${message}`);
};

export function validateThemeManifest(manifest, manifestPath = 'CONFIG.theme') {
  const diagnostics = [];

  if (!isPlainObject(manifest)) {
    pushDiagnostic(diagnostics, manifestPath, 'manifest must be a JSON object');
    return diagnostics;
  }

  if (manifest.schemaVersion !== 1) {
    pushDiagnostic(diagnostics, manifestPath, 'schemaVersion must equal 1');
  }
  if (!isNonEmptyString(manifest.id)) {
    pushDiagnostic(diagnostics, manifestPath, 'id must be a nonempty string');
  }
  if (!isPlainObject(manifest.brand)) {
    pushDiagnostic(diagnostics, manifestPath, 'brand must be an object');
  } else {
    if (!isNonEmptyString(manifest.brand.name)) {
      pushDiagnostic(
        diagnostics,
        manifestPath,
        'brand.name must be a nonempty string',
      );
    }

    if (manifest.brand.logos !== undefined) {
      for (const logo of ['full', 'compact']) {
        for (const mode of ['light', 'dark']) {
          const value = manifest.brand.logos?.[logo]?.[mode];
          if (!isAssetLink(value)) {
            pushDiagnostic(
              diagnostics,
              manifestPath,
              `brand.logos.${logo}.${mode} must be root-relative or use https`,
            );
          }
        }
      }
    }

    if (
      manifest.brand.favicon !== undefined &&
      !isAssetLink(manifest.brand.favicon)
    ) {
      pushDiagnostic(
        diagnostics,
        manifestPath,
        'brand.favicon must be root-relative or use https',
      );
    }
  }

  for (const link of REQUIRED_LINKS) {
    if (!isAppLink(manifest.links?.[link])) {
      pushDiagnostic(
        diagnostics,
        manifestPath,
        `links.${link} must be root-relative or use https/mailto`,
      );
    }
  }

  if (!['light', 'dark', 'system'].includes(manifest.defaultMode)) {
    pushDiagnostic(
      diagnostics,
      manifestPath,
      'defaultMode must be light, dark, or system',
    );
  }
  if (typeof manifest.allowModeSelection !== 'boolean') {
    pushDiagnostic(
      diagnostics,
      manifestPath,
      'allowModeSelection must be a boolean',
    );
  }

  for (const mode of ['light', 'dark']) {
    for (const color of REQUIRED_COLORS) {
      if (!isThemeColor(manifest.colors?.[mode]?.[color])) {
        pushDiagnostic(
          diagnostics,
          manifestPath,
          `colors.${mode}.${color} must be a six-digit hexadecimal color`,
        );
      }
    }

    const colors = manifest.colors?.[mode];
    if (
      REQUIRED_COLORS.every(color => isThemeColor(colors?.[color])) &&
      [colors.primary, colors.primaryHover, colors.primaryActive].some(
        color => getContrastRatio(color, colors.onPrimary) < 4.5,
      )
    ) {
      pushDiagnostic(
        diagnostics,
        manifestPath,
        `colors.${mode}.onPrimary must maintain WCAG AA contrast against all primary interaction states`,
      );
    }
  }

  return diagnostics.sort();
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
    if (/<script\b/i.test(source)) {
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

export function checkThemeContract(repoRoot = DEFAULT_REPO_ROOT) {
  return [
    ...validateLegacyThemeRemoval(repoRoot),
    ...validateShellFiles(repoRoot),
    ...validateThemeConfigFiles(repoRoot),
    ...validateSingleSourceTheme(repoRoot),
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

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
  'ui/src/app/components/layout/aside.tsx': [
    'bg-shell',
    'text-foreground',
    'border-border-subtle',
  ],
  'ui/src/app/components/layout/navigation/actionable-header.tsx': [
    'bg-shell',
    'text-foreground',
    'border-border-subtle',
  ],
  'ui/src/app/components/layout/navigation/header.tsx': [
    'bg-shell',
    'border-border-subtle',
  ],
  'ui/src/app/components/layout/navigation/sidebar.tsx': [
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
const APP_INDEX_PATH = 'ui/index.html';
const THEME_CONTRACT_PATH = 'ui/src/theme/theme-contract.json';
const TAILWIND_THEME_PATH = 'ui/src/styles/tailwind.css';
export const THEME_CONFIG_PATHS = Object.freeze([
  'ui/src/configs/config.development.json',
  'ui/src/configs/config.production.json',
  'docker/ui/config.community.json',
  'docker/ui/config.enterprise.json',
  'docker/ui/config.local.json',
  'docker/ui/config.local-knowledge.json',
]);
const GENERATED_STYLES_DIRECTORY = 'ui/src/styles/generated';
const CARBON_PROPERTY_SCAN_DIRECTORIES = Object.freeze([
  'ui/src/app',
  'ui/src/styles',
  'ui/src/theme',
]);
const CARBON_PROPERTY_PATTERN = /--cds-[a-z0-9-]+/gi;
export const THEME_CONTRACT = JSON.parse(
  readFileSync(resolve(DEFAULT_REPO_ROOT, THEME_CONTRACT_PATH), 'utf8'),
);
export const TAILWIND_PROSE_MAPPINGS = Object.freeze({
  '--tw-prose-body': 'var(--cds-text-primary)',
  '--tw-prose-headings': 'var(--cds-text-primary)',
  '--tw-prose-lead': 'var(--cds-text-secondary)',
  '--tw-prose-links': 'var(--cds-link-primary)',
  '--tw-prose-bold': 'var(--cds-text-primary)',
  '--tw-prose-counters': 'var(--cds-text-secondary)',
  '--tw-prose-bullets': 'var(--cds-icon-secondary)',
  '--tw-prose-hr': 'var(--cds-border-subtle-01)',
  '--tw-prose-quotes': 'var(--cds-text-secondary)',
  '--tw-prose-quote-borders': 'var(--cds-border-strong-01)',
  '--tw-prose-captions': 'var(--cds-text-helper)',
  '--tw-prose-kbd': 'var(--cds-text-primary)',
  '--tw-prose-kbd-shadows': 'initial',
  '--tw-prose-code': 'var(--cds-text-primary)',
  '--tw-prose-pre-code': 'var(--cds-text-inverse)',
  '--tw-prose-pre-bg': 'var(--cds-background-inverse)',
  '--tw-prose-th-borders': 'var(--cds-border-strong-01)',
  '--tw-prose-td-borders': 'var(--cds-border-subtle-01)',
  '--tw-prose-invert-body': 'var(--cds-text-primary)',
  '--tw-prose-invert-headings': 'var(--cds-text-primary)',
  '--tw-prose-invert-lead': 'var(--cds-text-secondary)',
  '--tw-prose-invert-links': 'var(--cds-link-primary)',
  '--tw-prose-invert-bold': 'var(--cds-text-primary)',
  '--tw-prose-invert-counters': 'var(--cds-text-secondary)',
  '--tw-prose-invert-bullets': 'var(--cds-icon-secondary)',
  '--tw-prose-invert-hr': 'var(--cds-border-subtle-01)',
  '--tw-prose-invert-quotes': 'var(--cds-text-secondary)',
  '--tw-prose-invert-quote-borders': 'var(--cds-border-strong-01)',
  '--tw-prose-invert-captions': 'var(--cds-text-helper)',
  '--tw-prose-invert-kbd': 'var(--cds-text-primary)',
  '--tw-prose-invert-kbd-shadows': 'initial',
  '--tw-prose-invert-code': 'var(--cds-text-primary)',
  '--tw-prose-invert-pre-code': 'var(--cds-text-inverse)',
  '--tw-prose-invert-pre-bg': 'var(--cds-background-inverse)',
  '--tw-prose-invert-th-borders': 'var(--cds-border-strong-01)',
  '--tw-prose-invert-td-borders': 'var(--cds-border-subtle-01)',
});
const BRAND_LITERAL_SCAN_DIRECTORIES = Object.freeze([
  'ui/src/app',
  'ui/src/theme',
]);
const BRAND_LITERAL_SCAN_FILES = Object.freeze(['ui/public/LLMs.txt']);
export const BRAND_LITERAL_ALLOWLIST = Object.freeze({
  'ui/src/app/components/domain/integration-document/endpoint-integration.tsx':
    {
      matches: Object.freeze([
        'description="Install the Rapida Python SDK using pip."',
        'description="Import the required classes and create a Rapida client."',
        'code={`from rapida import RapidaClient, RapidaClientOptions, RapidaEnvironment',
        'description="Install the Rapida Node.js SDK using npm or yarn."',
        'description="Import the required classes and create a Rapida client."',
        'description="Add the Rapida Go module to your project."',
        'description="Import the required packages and create a Rapida client."',
      ]),
      reason:
        'SDK package and client class names are external developer API identifiers',
    },
  'ui/src/app/pages/static-pages/terms.tsx': {
    matches: Object.freeze([
      'the Rapida.AI platform ("Platform"). By accessing or using the',
      '"Service" means the Rapida.AI platform, including all features',
      'Rapida.AI owns all intellectual property rights in and to',
      'You grant Rapida.AI a non-exclusive, worldwide, royalty-free',
      'agree that Rapida.AI may use your Content for marketing or',
      'you or Rapida.AI.',
      'Rapida.AI may terminate these Terms at any time for any',
      'Rapida.AI with respect to the subject matter hereof and',
      'Rapida.AI may amend these Terms at any time by posting the',
      'href="mailto:support@rapida.ai"',
      'support@rapida.ai',
    ]),
    reason: 'terms of service must identify the Rapida legal entity',
  },
  'ui/src/app/pages/assistant/view/conversations/conversation-messages.helpers.ts':
    {
      matches: Object.freeze(["if (roleKey === 'rapida') {"]),
      reason:
        'legacy conversation role value is mapped to a tenant-neutral display label',
    },
  'ui/src/app/components/domain/providers/embedding/cohere/constants.ts': {
    matches: Object.freeze(Array(7).fill("owner: 'rapida',")),
    reason:
      'provider model metadata retains the established technical catalog owner value',
  },
  'ui/src/app/components/domain/providers/embedding/openai/constants.ts': {
    matches: Object.freeze(Array(3).fill("owner: 'rapida',")),
    reason:
      'provider model metadata retains the established technical catalog owner value',
  },
  'ui/src/app/components/domain/providers/embedding/voyageai/constants.ts': {
    matches: Object.freeze(Array(7).fill("owner: 'rapida',")),
    reason:
      'provider model metadata retains the established technical catalog owner value',
  },
  'ui/src/app/components/domain/providers/reranker/cohere/constants.ts': {
    matches: Object.freeze(Array(4).fill("owner: 'rapida',")),
    reason:
      'provider model metadata retains the established technical catalog owner value',
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

const BUILT_IN_COLOR_PALETTES = Object.freeze([
  ...HARD_CODED_PALETTES,
  'red',
  'orange',
  'amber',
  'yellow',
  'lime',
  'green',
  'emerald',
  'teal',
  'cyan',
  'sky',
  'blue',
  'indigo',
  'violet',
  'purple',
  'fuchsia',
  'pink',
  'rose',
]);

const COLOR_UTILITY_PREFIXES =
  '(?:bg|text|placeholder|border(?:-[trblxy])?|ring|outline|divide|fill|stroke|decoration|from|via|to|shadow|accent|caret)';
const UTILITY_BOUNDARY_START = '(?:^|[^A-Za-z0-9_-])';
const UTILITY_BOUNDARY_END = '(?=$|[^A-Za-z0-9_-])';
const SEMANTIC_COLOR_NAMES = [
  'primary',
  'primary-hover',
  'primary-active',
  'primary-dark',
  'secondary',
  'light-background',
  'surface',
  'surface-hover',
  'surface-inverse',
  'surface-inverse-hover',
  'shell',
  'layer',
  'layer-elevated',
  'layer-hover',
  'layer-selected',
  'layer-accent',
  'field',
  'foreground',
  'muted',
  'helper',
  'disabled',
  'placeholder',
  'inverse',
  'on-primary',
  'icon-primary',
  'icon-secondary',
  'icon-disabled',
  'border-subtle',
  'border-strong',
  'link',
  'overlay',
  'notification-warning',
  'chat-prompt',
  'support-error',
  'support-success',
  'support-warning',
  'support-info',
  'data-1',
  'data-2',
  'data-3',
  'data-4',
  'data-5',
];

const THEME_STYLE_PATTERNS = Object.freeze({
  'typography-palette': new RegExp(
    String.raw`${UTILITY_BOUNDARY_START}(?:[A-Za-z0-9_-]+:)*!?prose-(?:${BUILT_IN_COLOR_PALETTES.join(
      '|',
    )})!?${UTILITY_BOUNDARY_END}`,
    'g',
  ),
  'dark-typography-variant': new RegExp(
    String.raw`${UTILITY_BOUNDARY_START}(?:[A-Za-z0-9_-]+:)*dark:(?:[A-Za-z0-9_-]+:)*!?prose-invert!?${UTILITY_BOUNDARY_END}`,
    'g',
  ),
  'literal-palette-utility': new RegExp(
    String.raw`${UTILITY_BOUNDARY_START}(?:[A-Za-z0-9_-]+:)*!?${COLOR_UTILITY_PREFIXES}-(?:${BUILT_IN_COLOR_PALETTES.join(
      '|',
    )})(?:-[0-9]{1,3})?(?:\/[0-9]{1,3})?!?${UTILITY_BOUNDARY_END}`,
    'g',
  ),
  'arbitrary-color-utility': new RegExp(
    String.raw`${UTILITY_BOUNDARY_START}(?:[A-Za-z0-9_-]+:)*!?${COLOR_UTILITY_PREFIXES}-\[[^\]]*(?:#|rgba?\(|hsla?\(|oklch\(|color\(|var\()[^\]]*\]!?${UTILITY_BOUNDARY_END}`,
    'gi',
  ),
  'dark-color-variant': new RegExp(
    String.raw`${UTILITY_BOUNDARY_START}(?:[A-Za-z0-9_-]+:)*dark:(?:[A-Za-z0-9_-]+:)*!?${COLOR_UTILITY_PREFIXES}-(?:(?:${BUILT_IN_COLOR_PALETTES.join(
      '|',
    )})(?:-[0-9]{1,3})?|\[[^\]]+\]|(?:${SEMANTIC_COLOR_NAMES.join(
      '|',
    )}))(?:\/[0-9]{1,3})?!?${UTILITY_BOUNDARY_END}`,
    'g',
  ),
  'literal-color': /#[0-9a-f]{3,8}\b|\b(?:rgba?|hsla?|oklch)\([^)]*\)/gi,
  'direct-carbon-color': /var\(\s*--cds-[^)]+\)/gi,
  'component-stylesheet':
    /(?:import\s+(?:[^'";]+?\s+from\s+)?|require\(\s*)['"][^'"]+\.(?:css|scss)['"]/g,
  'inline-color':
    /(?<![\w\[])\b(?:color|background|backgroundColor|borderColor|fill|stroke)\s*:(?!\s*(?:undefined\b|null\b|string\b|number\b|boolean\b|['"]?(?:transparent|currentColor|inherit)['"]?\b))\s*[^,}\n]+/g,
});

export const THEME_STYLE_EXCEPTIONS = Object.freeze({
  'ui/src/theme/theme-config.ts': Object.freeze({
    'literal-color': Object.freeze({
      matches: Object.freeze([
        '#525252',
        '#393939',
        '#262626',
        '#ffffff',
        '#c6c6c6',
        '#e0e0e0',
        '#f4f4f4',
        '#161616',
      ]),
      reason: 'version 1 compatibility defaults are frozen public behavior',
    }),
  }),
  'ui/src/app/components/ui/primitives/buttons/social-button-group.tsx':
    Object.freeze({
      'literal-color': Object.freeze({
        matches: Object.freeze(['#EA4335', '#4285F4', '#FBBC05', '#34A853']),
        reason:
          'Google owns the fixed multicolor mark rendered by this sign-in control',
      }),
    }),
  'ui/src/app/index.tsx': Object.freeze({
    'component-stylesheet': Object.freeze({
      matches: Object.freeze(["import 'reactflow/dist/style.css'"]),
      reason:
        'React Flow requires its package stylesheet for graph layout and controls',
    }),
  }),
  'ui/src/app/components/dialogs/conversation/conversation-telemetry-latency-stack-chart.tsx':
    Object.freeze({
      'inline-color': Object.freeze({
        matches: Object.freeze([
          'backgroundColor: item.color || meta?.seriesColor',
          'backgroundColor: LATENCY_METRIC_META[metricName].seriesColor',
          "fill: 'var(--color-muted)'",
        ]),
        reason:
          'Recharts requires semantic colors through SVG and tooltip style props',
      }),
    }),
  'ui/src/app/pages/activities/conversation-activity-v2/components/conversation-waterfall.tsx':
    Object.freeze({
      'inline-color': Object.freeze({
        matches: Object.freeze([
          'backgroundColor: getDocumentColor(item)',
          'backgroundColor: getDocumentColor(item)',
        ]),
        reason:
          'Timeline widths and semantic event colors are data-driven inline values',
      }),
    }),
  'ui/src/app/pages/activities/conversation-activity-v2/index.tsx':
    Object.freeze({
      'inline-color': Object.freeze({
        matches: Object.freeze([
          "fill: 'var(--color-muted)'",
          "fill: 'var(--color-muted)'",
        ]),
        reason: 'Recharts axis ticks require semantic colors through SVG props',
      }),
    }),
  'ui/src/app/pages/assistant/actions/agentflow/index.tsx': Object.freeze({
    'inline-color': Object.freeze({
      matches: Object.freeze([
        'background: colors[0]',
        "color: 'yellow'",
        "stroke: 'var(--color-muted)'",
      ]),
      reason:
        'React Flow owns connection styles and the node color field is persisted domain data',
    }),
  }),
  'ui/src/app/pages/assistant/actions/agentflow/nodes/sticky-note-node/card.tsx':
    Object.freeze({
      'inline-color': Object.freeze({
        matches: Object.freeze([
          'background: foldBackground',
          'background: stickyNoteHighlight',
          "backgroundColor: 'var(--color-data-1)'",
          'backgroundColor: colors.surface',
          "borderColor: 'var(--color-muted)'",
        ]),
        reason:
          'React Flow resize handles and data-driven note surfaces require style props',
      }),
    }),
  'ui/src/app/pages/assistant/actions/create-assistant-version/agentflow-version-builder.tsx':
    Object.freeze({
      'inline-color': Object.freeze({
        matches: Object.freeze([
          'background: colors[0]',
          "color: 'yellow'",
          "stroke: 'var(--color-muted)'",
        ]),
        reason:
          'React Flow owns connection styles and the node color field is persisted domain data',
      }),
    }),
  'ui/src/app/pages/assistant/view/overview/assistant-analytics.tsx':
    Object.freeze({
      'inline-color': Object.freeze({
        matches: Object.freeze([
          'backgroundColor: CHART_COLORS[i % CHART_COLORS.length]',
          'backgroundColor: CHART_COLORS[i % CHART_COLORS.length]',
          'backgroundColor: color',
          'backgroundColor: itemColor',
          'backgroundColor: p.stroke',
          "fill: 'var(--color-muted)'",
          "fill: 'var(--color-muted)'",
          "fill: 'var(--color-primary)'",
        ]),
        reason:
          'Recharts and data-driven legends require semantic colors through style props',
      }),
    }),
  'ui/src/app/pages/endpoint/view/pages/endpoint-activity-dashboard.tsx':
    Object.freeze({
      'inline-color': Object.freeze({
        matches: Object.freeze([
          "background: 'var(--color-layer)'",
          "color: 'var(--color-foreground)'",
          "fill: 'var(--color-muted)'",
          "fill: 'var(--color-muted)'",
        ]),
        reason:
          'Recharts requires semantic tooltip and axis colors through style props',
      }),
    }),
  'ui/src/app/pages/main/web-dashboard/index.tsx': Object.freeze({
    'component-stylesheet': Object.freeze({
      matches: Object.freeze([
        "import '@carbon/charts-react/styles.css'",
      ]),
      reason:
        'Carbon Charts requires its package stylesheet, scoped to the dashboard route to preserve the initial CSS budget',
    }),
    'inline-color': Object.freeze({
      matches: Object.freeze([
        "background: 'var(--color-layer)'",
        "color: 'var(--color-foreground)'",
        "fill: 'var(--color-muted)'",
        "fill: 'var(--color-muted)'",
      ]),
      reason:
        'Recharts requires semantic tooltip and axis colors through style props',
    }),
  }),
});

export const EXTERNAL_STYLE_ADAPTERS = Object.freeze({
  'ui/src/styles/externals/editors.css': Object.freeze({
    selectors: Object.freeze([
      '.markdown-editor .cm-line',
      '.prompt-editor .suggest-widget',
      '.prompt-editor .suggest-details',
      '.json-editor .suggest-widget',
      '.json-editor .suggest-details',
      '.monaco-editor',
      '.monaco-editor',
      '.monaco-editor-background',
      '.prompt-editor .margin',
      '.json-editor .margin',
      '.code-highlighting-editor .margin',
      '.ͼ42',
      '.ͼo',
      '.monaco-editor .margin-view-overlays .current-line',
      '.monaco-editor .view-overlays .current-line',
      '.cm-content',
      '.monaco-editor-overlaymessage',
      '.prompt-editor .monaco-editor',
      '.prompt-editor.monaco-editor',
      '.json-editor .monaco-editor',
      '.json-editor.monaco-editor',
      '.code-highlighting-editor .monaco-editor',
      '.code-highlighting-editor.monaco-editor',
      '.cm-editor',
      '.prompt-editor-placeholder',
      '.json-editor-placeholder',
      '.cm-line .cm-placeholder',
      '.json-editor-surface::after',
      '.json-editor-surface:focus-within::after',
      '.cm-content',
      '.cm-scroller',
      '.cm-gutters',
      '.ͼ1 .cm-cursor',
      '.ͼ1 .cm-dropCursor',
    ]),
    reason:
      'CodeMirror, Monaco, and markdown packages expose runtime selectors instead of component class APIs',
  }),
  'ui/src/styles/externals/voice-visualizer.css': Object.freeze({
    selectors: Object.freeze([
      '.listening',
      '.listening::before',
      '.listening::after',
      '.listening::after',
    ]),
    reason:
      'the voice visualizer package owns its runtime state class and pseudo-elements',
  }),
  'ui/src/styles/externals/flatpickr.css': Object.freeze({
    selectors: Object.freeze([
      '.flatpickr-calendar',
      '.flatpickr-months',
      '.flatpickr-month',
      '.flatpickr-weekdays',
      '.flatpickr-weekday',
      '.flatpickr-day',
      '.flatpickr-day.prevMonthDay',
      '.flatpickr-day.nextMonthDay',
      '.flatpickr-current-month .flatpickr-monthDropdown-months',
      '.flatpickr-current-month input.cur-year',
      '.flatpickr-months .flatpickr-prev-month svg',
      '.flatpickr-months .flatpickr-next-month svg',
    ]),
    reason:
      'flatpickr renders its calendar outside the Carbon date-picker component tree',
  }),
});

const extractExternalSelectors = source =>
  [...source.matchAll(/([^{}]+)\{/g)].flatMap(match =>
    match[1]
      .split(',')
      .map(selector => selector.trim().replace(/\s+/g, ' '))
      .filter(Boolean),
  );

const countValues = values => {
  const counts = new Map();
  for (const value of values) counts.set(value, (counts.get(value) ?? 0) + 1);
  return counts;
};

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
  if (!isPlainObject(manifest)) {
    return [`${manifestPath}: manifest must be a JSON object`];
  }

  if (manifest.schemaVersion === undefined || manifest.schemaVersion === 1) {
    return [];
  }
  if (manifest.schemaVersion !== THEME_CONTRACT.schemaVersion) {
    return [`${manifestPath}.schemaVersion: unsupported schema version`];
  }

  const diagnostics = [];
  const requireObject = (value, path) => {
    if (!isPlainObject(value)) {
      pushDiagnostic(diagnostics, path, 'required JSON object');
      return {};
    }
    return value;
  };
  const requireString = (value, path) => {
    if (typeof value !== 'string' || value.trim().length === 0) {
      pushDiagnostic(diagnostics, path, 'required non-empty string');
      return;
    }
    if (/var\(\s*--(?:cds|brand)-/i.test(value)) {
      pushDiagnostic(
        diagnostics,
        path,
        'internal Carbon and brand property references are not allowed',
      );
    }
  };

  requireString(manifest.id, `${manifestPath}.id`);
  const brand = requireObject(manifest.brand, `${manifestPath}.brand`);
  requireString(brand.name, `${manifestPath}.brand.name`);
  if (brand.environmentLabel !== undefined) {
    requireString(
      brand.environmentLabel,
      `${manifestPath}.brand.environmentLabel`,
    );
  }
  if (brand.favicon !== undefined) {
    requireString(brand.favicon, `${manifestPath}.brand.favicon`);
  }
  if (brand.logos !== undefined) {
    const logos = requireObject(brand.logos, `${manifestPath}.brand.logos`);
    for (const shape of ['full', 'compact']) {
      const logo = requireObject(
        logos[shape],
        `${manifestPath}.brand.logos.${shape}`,
      );
      for (const mode of ['light', 'dark']) {
        requireString(
          logo[mode],
          `${manifestPath}.brand.logos.${shape}.${mode}`,
        );
      }
    }
  }

  const links = requireObject(manifest.links, `${manifestPath}.links`);
  for (const field of [
    'documentation',
    'source',
    'support',
    'terms',
    'privacy',
  ]) {
    requireString(links[field], `${manifestPath}.links.${field}`);
  }
  if (!['light', 'dark', 'system'].includes(manifest.defaultMode)) {
    pushDiagnostic(
      diagnostics,
      `${manifestPath}.defaultMode`,
      'must be light, dark, or system',
    );
  }
  if (typeof manifest.allowModeSelection !== 'boolean') {
    pushDiagnostic(
      diagnostics,
      `${manifestPath}.allowModeSelection`,
      'required boolean',
    );
  }

  const colors = requireObject(manifest.colors, `${manifestPath}.colors`);
  for (const mode of ['light', 'dark']) {
    const modeColors = requireObject(
      colors[mode],
      `${manifestPath}.colors.${mode}`,
    );
    const allowedThemes = THEME_CONTRACT.carbonThemes[mode];
    if (
      typeof modeColors.carbonTheme !== 'string' ||
      !Object.hasOwn(allowedThemes, modeColors.carbonTheme)
    ) {
      pushDiagnostic(
        diagnostics,
        `${manifestPath}.colors.${mode}.carbonTheme`,
        `must be one of ${Object.keys(allowedThemes).join(', ')}`,
      );
    }
    for (const field of Object.keys(THEME_CONTRACT.colorFields)) {
      requireString(
        modeColors[field],
        `${manifestPath}.colors.${mode}.${field}`,
      );
    }
  }

  return diagnostics.sort();
}

const countPatternMatches = (source, pattern) => {
  pattern.lastIndex = 0;
  return [...source.matchAll(pattern)].map(match => match[0].trim());
};

export function findThemeStyleViolations(source, sourcePath = '') {
  const violations = [];

  for (const [rule, pattern] of Object.entries(THEME_STYLE_PATTERNS)) {
    const exception = THEME_STYLE_EXCEPTIONS[sourcePath]?.[rule];
    const remainingExceptions = new Map();
    for (const match of exception?.matches ?? []) {
      remainingExceptions.set(match, (remainingExceptions.get(match) ?? 0) + 1);
    }

    for (const match of countPatternMatches(source, pattern)) {
      const remaining = remainingExceptions.get(match) ?? 0;
      if (remaining > 0) {
        remainingExceptions.set(match, remaining - 1);
        continue;
      }
      violations.push({ rule, match });
    }
  }

  return violations.sort(
    (left, right) =>
      left.rule.localeCompare(right.rule) ||
      left.match.localeCompare(right.match),
  );
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

const isThemeStyleScanFile = sourcePath => {
  if (
    sourcePath.includes('/__tests__/') ||
    /\.(?:test|spec|stories)\.[cm]?[jt]sx?$/.test(sourcePath)
  ) {
    return false;
  }

  return /\.(?:ts|tsx)$/.test(sourcePath);
};

const isCarbonPropertyScanFile = sourcePath => {
  if (
    sourcePath.includes('/__tests__/') ||
    sourcePath.startsWith(`${GENERATED_STYLES_DIRECTORY}/`) ||
    /\.(?:test|spec|stories)\.[cm]?[jt]sx?$/.test(sourcePath)
  ) {
    return false;
  }

  return /\.(?:css|scss|ts|tsx)$/.test(sourcePath);
};

export function collectConsumedCarbonProperties(repoRoot = DEFAULT_REPO_ROOT) {
  const properties = new Set();

  for (const directory of CARBON_PROPERTY_SCAN_DIRECTORIES) {
    const directoryPath = resolve(repoRoot, directory);
    let files = [];
    try {
      if (statSync(directoryPath).isDirectory())
        files = walkFiles(directoryPath);
    } catch (error) {
      if (error?.code !== 'ENOENT') throw error;
    }

    for (const filePath of files) {
      const sourcePath = toPosixPath(relative(repoRoot, filePath));
      if (!isCarbonPropertyScanFile(sourcePath)) continue;

      const source = readFileSync(filePath, 'utf8');
      CARBON_PROPERTY_PATTERN.lastIndex = 0;
      for (const match of source.matchAll(CARBON_PROPERTY_PATTERN)) {
        properties.add(match[0].toLowerCase());
      }
    }
  }

  return [...properties].sort();
}

export function validateCarbonPropertyCoverage(repoRoot = DEFAULT_REPO_ROOT) {
  const diagnostics = [];
  const brandOverrides = THEME_CONTRACT.carbonProperties?.brandOverrides;
  const presetOwned = THEME_CONTRACT.carbonProperties?.presetOwned;
  if (!isPlainObject(brandOverrides) || !Array.isArray(presetOwned)) {
    return [
      `${THEME_CONTRACT_PATH}: carbonProperties must classify brand overrides and preset-owned properties`,
    ];
  }

  const classified = new Map();
  const classify = (property, owner) => {
    if (typeof property !== 'string' || !/^--cds-[a-z0-9-]+$/.test(property)) {
      pushDiagnostic(
        diagnostics,
        THEME_CONTRACT_PATH,
        `invalid Carbon property classification: ${String(property)}`,
      );
      return;
    }
    if (classified.has(property)) {
      pushDiagnostic(
        diagnostics,
        THEME_CONTRACT_PATH,
        `${property} has more than one Carbon property owner`,
      );
      return;
    }
    classified.set(property, owner);
  };

  for (const field of Object.keys(THEME_CONTRACT.colorFields)) {
    const properties = brandOverrides[field];
    if (!Array.isArray(properties) || properties.length === 0) {
      pushDiagnostic(
        diagnostics,
        THEME_CONTRACT_PATH,
        `${field} must own at least one Carbon property`,
      );
      continue;
    }
    for (const property of properties) classify(property, `brand:${field}`);
  }
  for (const field of Object.keys(brandOverrides)) {
    if (!Object.hasOwn(THEME_CONTRACT.colorFields, field)) {
      pushDiagnostic(
        diagnostics,
        THEME_CONTRACT_PATH,
        `unknown brand override field: ${field}`,
      );
    }
  }
  for (const property of presetOwned) classify(property, 'carbon-preset');

  let carbonAdapter = '';
  try {
    carbonAdapter = readFileSync(
      resolve(repoRoot, 'ui/src/styles/carbon.scss'),
      'utf8',
    );
  } catch (error) {
    pushDiagnostic(
      diagnostics,
      'ui/src/styles/carbon.scss',
      error?.code === 'ENOENT'
        ? 'Carbon theme adapter is missing'
        : `could not read Carbon theme adapter: ${error.message}`,
    );
  }
  for (const [field, properties] of Object.entries(brandOverrides)) {
    if (!Array.isArray(properties)) continue;
    const brandProperty = THEME_CONTRACT.colorFields[field]?.property;
    if (typeof brandProperty !== 'string') continue;
    for (const property of properties) {
      if (!carbonAdapter.includes(`${property}: var(${brandProperty})`)) {
        pushDiagnostic(
          diagnostics,
          'ui/src/styles/carbon.scss',
          `${property} must map to ${brandProperty}`,
        );
      }
    }
  }

  const consumed = new Set(collectConsumedCarbonProperties(repoRoot));
  for (const property of consumed) {
    if (!classified.has(property)) {
      pushDiagnostic(
        diagnostics,
        THEME_CONTRACT_PATH,
        `${property} is consumed but has no declared owner`,
      );
    }
  }
  for (const property of classified.keys()) {
    if (!consumed.has(property)) {
      pushDiagnostic(
        diagnostics,
        THEME_CONTRACT_PATH,
        `${property} is classified but not consumed`,
      );
    }
  }

  return diagnostics.sort();
}

export function validateTailwindThemeContract(repoRoot = DEFAULT_REPO_ROOT) {
  let source;
  try {
    source = readFileSync(resolve(repoRoot, TAILWIND_THEME_PATH), 'utf8');
  } catch (error) {
    return [
      `${TAILWIND_THEME_PATH}: ${
        error?.code === 'ENOENT'
          ? 'Tailwind theme adapter is missing'
          : `could not read Tailwind theme adapter: ${error.message}`
      }`,
    ];
  }

  const diagnostics = [];
  if (!source.includes('--color-*: initial;')) {
    diagnostics.push(
      `${TAILWIND_THEME_PATH}: built-in color namespace must be disabled with --color-*: initial`,
    );
  }
  for (const [property, value] of Object.entries(TAILWIND_PROSE_MAPPINGS)) {
    if (!source.includes(`${property}: ${value};`)) {
      diagnostics.push(
        `${TAILWIND_THEME_PATH}: ${property} must map to ${value}`,
      );
    }
  }
  return diagnostics.sort();
}

export function validateThemeStyles(repoRoot = DEFAULT_REPO_ROOT) {
  const diagnostics = [];
  for (const directory of BRAND_LITERAL_SCAN_DIRECTORIES) {
    const directoryPath = resolve(repoRoot, directory);
    let files = [];
    try {
      if (statSync(directoryPath).isDirectory())
        files = walkFiles(directoryPath);
    } catch (error) {
      if (error?.code !== 'ENOENT') throw error;
    }

    for (const filePath of files) {
      const sourcePath = toPosixPath(relative(repoRoot, filePath));
      if (!isThemeStyleScanFile(sourcePath)) continue;

      const violations = findThemeStyleViolations(
        readFileSync(filePath, 'utf8'),
        sourcePath,
      );
      for (const { rule, match } of violations) {
        pushDiagnostic(
          diagnostics,
          sourcePath,
          `${rule} is not allowed: ${match}`,
        );
      }
    }
  }

  return diagnostics.sort();
}

export function validateExternalStyleAdapters(repoRoot = DEFAULT_REPO_ROOT) {
  const diagnostics = [];
  const adapterDirectory = resolve(repoRoot, 'ui/src/styles/externals');
  const expectedPaths = new Set(Object.keys(EXTERNAL_STYLE_ADAPTERS));
  let actualPaths = [];
  try {
    actualPaths = walkFiles(adapterDirectory)
      .filter(filePath => filePath.endsWith('.css'))
      .map(filePath => toPosixPath(relative(repoRoot, filePath)));
  } catch (error) {
    if (error?.code !== 'ENOENT') throw error;
  }

  for (const sourcePath of actualPaths) {
    if (!expectedPaths.has(sourcePath)) {
      pushDiagnostic(
        diagnostics,
        sourcePath,
        'external style adapter is not allowlisted',
      );
    }
  }

  let globalStyles = '';
  try {
    globalStyles = readFileSync(
      resolve(repoRoot, 'ui/src/styles/global-styles.ts'),
      'utf8',
    );
  } catch (error) {
    pushDiagnostic(
      diagnostics,
      'ui/src/styles/global-styles.ts',
      'global style owner is missing',
    );
  }

  for (const [sourcePath, adapter] of Object.entries(EXTERNAL_STYLE_ADAPTERS)) {
    let source;
    try {
      source = readFileSync(resolve(repoRoot, sourcePath), 'utf8');
    } catch (error) {
      pushDiagnostic(diagnostics, sourcePath, 'allowlisted adapter is missing');
      continue;
    }

    const importPath = `./externals/${sourcePath.split('/').at(-1)}`;
    if (!globalStyles.includes(importPath)) {
      pushDiagnostic(
        diagnostics,
        sourcePath,
        'allowlisted adapter must be imported by global-styles.ts',
      );
    }
    const expectedSelectors = countValues(adapter.selectors);
    const actualSelectors = countValues(extractExternalSelectors(source));
    for (const [selector, expectedCount] of expectedSelectors) {
      const actualCount = actualSelectors.get(selector) ?? 0;
      if (actualCount < expectedCount) {
        pushDiagnostic(
          diagnostics,
          sourcePath,
          `required external selector is missing: ${selector} (${actualCount}/${expectedCount})`,
        );
      }
    }
    for (const [selector, actualCount] of actualSelectors) {
      const expectedCount = expectedSelectors.get(selector) ?? 0;
      if (actualCount > expectedCount) {
        pushDiagnostic(
          diagnostics,
          sourcePath,
          `external selector is not allowlisted: ${selector} (${actualCount}/${expectedCount})`,
        );
      }
    }
    if (
      /#[0-9a-f]{3,8}\b|\b(?:rgba?|hsla?|oklch)\(|(?<![-\w])(?:aliceblue|antiquewhite|aqua|aquamarine|azure|beige|bisque|black|blue|brown|coral|crimson|cyan|fuchsia|gold|goldenrod|gray|green|grey|indigo|ivory|khaki|lavender|lime|magenta|maroon|navy|olive|orange|orchid|pink|plum|purple|red|salmon|silver|tan|teal|tomato|turquoise|violet|white|yellow)(?![-\w])/i.test(
        source,
      )
    ) {
      pushDiagnostic(
        diagnostics,
        sourcePath,
        'literal colors are not allowed in external adapters',
      );
    }
  }

  return diagnostics.sort();
}

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
    const source = readFileSync(resolve(repoRoot, APP_INDEX_PATH), 'utf8');
    const scriptTags = source.match(/<script\b[^>]*>/gi) ?? [];
    if (scriptTags.some(scriptTag => !/\ssrc\s*=/i.test(scriptTag))) {
      pushDiagnostic(
        diagnostics,
        APP_INDEX_PATH,
        'inline bootstrap scripts are not allowed; use CONFIG.theme',
      );
    }
  } catch (error) {
    pushDiagnostic(
      diagnostics,
      APP_INDEX_PATH,
      error?.code === 'ENOENT'
        ? 'application index is missing'
        : `could not read application index: ${error.message}`,
    );
  }

  return diagnostics.sort();
}

export function validateBrandLiterals(repoRoot = DEFAULT_REPO_ROOT) {
  const diagnostics = [];

  for (const { filePath, sourcePath } of collectBrandScanFiles(repoRoot)) {
    let source = readFileSync(filePath, 'utf8');
    for (const allowedMatch of
      BRAND_LITERAL_ALLOWLIST[sourcePath]?.matches ?? []) {
      const matchIndex = source.indexOf(allowedMatch);
      if (matchIndex === -1) {
        pushDiagnostic(
          diagnostics,
          sourcePath,
          `allowlisted brand construct is missing: ${allowedMatch}`,
        );
        continue;
      }
      source = `${source.slice(0, matchIndex)}${' '.repeat(
        allowedMatch.length,
      )}${source.slice(matchIndex + allowedMatch.length)}`;
    }

    for (const literal of findHardcodedBrandLiterals(source)) {

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
    ...validateCarbonPropertyCoverage(repoRoot),
    ...validateTailwindThemeContract(repoRoot),
    ...validateThemeStyles(repoRoot),
    ...validateExternalStyleAdapters(repoRoot),
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

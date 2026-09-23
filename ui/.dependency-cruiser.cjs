// UI import gate. `yarn check:dependencies` runs it, and CI can gate on it.
// The graph covers production source. Tests and generated Tailwind output are
// excluded so the rules stay focused on code that ships in the app bundle.
// dependency-cruiser is pinned in package.json because newer releases currently
// pull a transitive dependency that requires Node 22.13 or later.
const testAndGeneratedSources = [
  '^src/styles/generated/',
  '(^|/)__tests__/',
  '\\.(test|spec)\\.[cm]?[jt]sx?$',
];

module.exports = {
  forbidden: [
    {
      name: 'not-to-unresolvable',
      comment:
        'Every production import must resolve through TypeScript, package.json, ' +
        'or node_modules. A missing target is usually a broken path alias, ' +
        'deleted file, or undeclared dependency.',
      severity: 'error',
      from: {},
      to: {
        couldNotResolve: true,
      },
    },
    {
      name: 'no-circular',
      comment:
        'Circular imports make module initialization order part of behavior. ' +
        'Keep ownership one-way so components, stores, and provider config can ' +
        'be changed independently.',
      severity: 'error',
      from: {},
      to: {
        circular: true,
      },
    },
    {
      name: 'no-non-package-json',
      comment:
        'An npm import must be declared in package.json. Otherwise local installs ' +
        'can pass while clean installs or production builds fail.',
      severity: 'error',
      from: {},
      to: {
        dependencyTypes: ['npm-no-pkg', 'npm-unknown'],
      },
    },
    {
      name: 'not-to-dev-dep-from-production',
      comment:
        'Renderable source may not import dev-only packages. Tests and setup ' +
        'files are excluded above because they are the intended consumers of ' +
        'test tooling.',
      severity: 'error',
      from: {
        path: '^src/',
        pathNot: ['^src/setup-tests\\.ts$', ...testAndGeneratedSources],
      },
      to: {
        dependencyTypes: ['npm-dev'],
      },
    },
    {
      name: 'not-to-deprecated',
      comment:
        'Deprecated npm dependencies stay visible as warnings so they can be ' +
        'removed deliberately without blocking unrelated UI work.',
      severity: 'warn',
      from: {},
      to: {
        dependencyTypes: ['deprecated'],
      },
    },
  ],
  options: {
    tsConfig: {
      fileName: 'tsconfig.json',
    },
    includeOnly: {
      path: '^src/',
    },
    exclude: {
      path: testAndGeneratedSources,
    },
    doNotFollow: {
      path: 'node_modules',
      dependencyTypes: [
        'npm',
        'npm-dev',
        'npm-optional',
        'npm-peer',
        'npm-bundled',
        'npm-no-pkg',
      ],
    },
    enhancedResolveOptions: {
      extensions: ['.js', '.jsx', '.ts', '.tsx', '.json', '.d.ts'],
    },
    reporterOptions: {
      dot: {
        // Keep generated dependency graphs readable by collapsing packages.
        collapsePattern: 'node_modules/[^/]+',
      },
    },
  },
};

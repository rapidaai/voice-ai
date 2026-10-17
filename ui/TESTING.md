# UI Testing

The UI uses Jest and Testing Library for component behavior, Playwright for browser behavior, axe for
accessibility, and Storybook for isolated component development. See `DEVELOPMENT.md` for the test
selection and implementation workflow.

## Setup and development

Install the locked dependencies and generate the CSS bundle:

```bash
yarn install --frozen-lockfile
yarn build:css
```

Start the Vite application or Storybook:

```bash
yarn start:dev
yarn storybook
```

Install the Playwright browser once:

```bash
yarn e2e:install
```

## Verification

Run the UI gate without browser tests:

```bash
yarn verify:ui
```

Run the complete gate, including Playwright:

```bash
yarn verify:ui:e2e
```

The verifier runs every stage and reports all failures at the end. It does not modify source files
or apply automatic fixes.

## Jest

Run all Jest tests:

```bash
yarn test --watchAll=false --passWithNoTests
```

Run the nearest suite while developing:

```bash
yarn test --watchAll=false src/path/to/changed.test.tsx
```

Run Jest with coverage:

```bash
yarn test:coverage
```

Coverage is written to `coverage/lcov-report/index.html`.

## Allure reports

Allure is optional and requires Java. Generate fresh results, coverage, and the HTML report with:

```bash
yarn allure:report
```

Open or serve an existing report with:

```bash
yarn allure:open
yarn allure:serve
```

Generated output is written to `allure-results/` and `allure-report/` and is ignored by Git.

## Playwright

Playwright covers route inventory, browser smoke tests, accessibility, and screenshot comparison.
The default run builds the current source and starts a fresh production server. It does not attach to
an existing local server. Set `PLAYWRIGHT_SKIP_WEB_SERVER=1` with `PLAYWRIGHT_BASE_URL` only when an
external server is intentionally under test.

Run individual lanes:

```bash
yarn check:e2e-routes
yarn e2e:smoke
yarn e2e:a11y
yarn e2e:screenshots
```

Run the complete browser suite:

```bash
yarn e2e
```

`yarn test:e2e` runs the same canonical suite and is used by CI and `just ui-browser-test`.

The route inventory lives in `e2e/route-manifest.js`. Every enabled route must have an explicit
smoke, accessibility, and screenshot decision. A route can defer a check only with a concrete reason.

Keep route-wide contract specs at the `e2e/` root. Put page interaction journeys in
`e2e/journeys/<route-area>/` and reusable client doubles in `e2e/mocks/clients/`, with one module per
client domain. Shared gRPC-Web framing stays in `e2e/mocks/grpc-web.js`. Install default client
behavior through `e2e/fixtures.js`, then override only the operation needed for an error or edge-path
assertion. Route-wide smoke, accessibility, and screenshot coverage stays manifest-driven.

Accessibility checks cover WCAG 2.1 A and AA in light and dark modes for every journey that is not
explicitly deferred. Existing debt is recorded in `e2e/a11y-baseline.js` with a maximum node count
for each journey, rule, and selector. A new finding or an increase above that limit fails the suite.

Playwright writes failure screenshots, diffs, traces, and videos to `test-results/e2e/`. Its HTML
report is generated under `e2e/reports/` in the `playwright` subdirectory. Reviewed visual baselines use the
`e2e/screenshots/<route-area>/<journey>-<theme>.png` structure.

Update visual baselines only after reviewing the rendered change:

```bash
yarn e2e:screenshots --update-snapshots
```

`yarn test:e2e:update` is an alias for the same reviewed screenshot update.

## Documentation and tooling checks

The UI documentation checker validates documented Yarn scripts and referenced UI paths:

```bash
yarn check:docs
```

The checker and verification runner have focused Node tests under `scripts/*.test.mjs`:

```bash
yarn test:tooling
```

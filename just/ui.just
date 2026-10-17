# UI local defaults. Override with environment variables or recipe args.
ui_storybook_port := env_var_or_default("STORYBOOK_PORT", "6006")
ui_storybook_host := env_var_or_default("STORYBOOK_HOST", "127.0.0.1")
ui_test_path := env_var_or_default("UI_TEST_PATH", "")

# Set up the complete UI developer workspace, test report, and Storybook build.
ui:
    @just ui-setup
    @just ui-doctor
    @just ui-report
    @just ui-storybook-build
    @just ui-help

# Show the primary UI development commands.
ui-help:
    @printf '%s\n' \
      'UI workspace is ready.' \
      'Development server: just ui-dev' \
      'Storybook:          just ui-storybook' \
      'Allure report:      just ui-report-open' \
      'Coverage report:    ui/coverage/lcov-report/index.html' \
      'Focused report:     just ui-report src/app/components/app-shell' \
      'Quality checks:     just ui-quality'

# Install UI dependencies and prepare generated local assets.
ui-setup:
    #!/usr/bin/env bash
    cd ui
    yarn install --frozen-lockfile
    mkdir -p src/styles/generated
    yarn build:css
    echo "UI setup complete."

# Check that UI, Storybook, and Allure local tooling is available.
ui-doctor:
    #!/usr/bin/env bash
    command -v node >/dev/null
    command -v yarn >/dev/null
    command -v java >/dev/null
    cd ui
    node - <<'NODE'
    const pkg = require('./package.json');
    const deps = { ...pkg.dependencies, ...pkg.devDependencies };
    const required = [
      '@chromatic-com/storybook',
      '@storybook/addon-a11y',
      '@storybook/addon-docs',
      '@storybook/react-vite',
      '@vitejs/plugin-react',
      'allure-commandline',
      'allure-jest',
      'allure-js-commons',
      'jest',
      'storybook',
      'vite',
    ];
    const missing = required.filter(name => !deps[name]);
    if (missing.length) {
      console.error(`Missing UI tooling dependencies: ${missing.join(', ')}`);
      process.exit(1);
    }
    NODE
    test -f .storybook/main.ts
    test -f .storybook/preview.tsx
    test -f TESTING.md
    echo "UI local tooling looks ready."

# Run the UI development server.
ui-dev: ui-setup
    cd ui && yarn start:dev

# Run Storybook locally. Override with STORYBOOK_HOST and STORYBOOK_PORT.
ui-storybook host=ui_storybook_host port=ui_storybook_port: ui-setup
    cd ui && STORYBOOK_DISABLE_TELEMETRY=1 ./node_modules/.bin/storybook dev --host "{{host}}" --port "{{port}}"

# Build static Storybook docs locally.
ui-storybook-build: ui-setup
    cd ui && STORYBOOK_DISABLE_TELEMETRY=1 ./node_modules/.bin/storybook build

# Run UI Jest tests. Set UI_TEST_PATH or pass a path to focus the run.
ui-test path=ui_test_path: ui-setup
    #!/usr/bin/env bash
    cd ui
    if [[ -n "{{path}}" ]]; then
      yarn test --watchAll=false "{{path}}"
    else
      yarn test --watchAll=false --passWithNoTests
    fi

# Run UI Jest coverage.
ui-test-coverage: ui-setup
    cd ui && yarn test:coverage

# Install browser binaries for Playwright.
ui-e2e-install: ui-setup
    cd ui && yarn e2e:install

# Check that the Playwright route manifest matches app route roots.
ui-e2e-routes: ui-setup
    cd ui && yarn check:e2e-routes

# Run Playwright route smoke tests. Run ui-e2e-install once first.
ui-e2e-smoke: ui-setup
    cd ui && yarn e2e:smoke

# Run the Playwright accessibility gate. Run ui-e2e-install once first.
ui-e2e-a11y: ui-setup
    cd ui && yarn e2e:a11y

# Capture the Playwright screenshot journey. Run ui-e2e-install once first.
ui-e2e-screenshots: ui-setup
    cd ui && yarn e2e:screenshots

# Run all Playwright e2e checks. Run ui-e2e-install once first.
ui-e2e: ui-setup
    cd ui && yarn check:e2e-routes
    cd ui && yarn e2e

# Generate an Allure report. Set UI_TEST_PATH or pass a path to focus the run.
ui-report path=ui_test_path: ui-setup
    #!/usr/bin/env bash
    cd ui
    if [[ -n "{{path}}" ]]; then
      yarn test:allure "{{path}}"
      yarn allure:generate
    else
      yarn allure:report
      echo "Coverage report generated at ui/coverage/lcov-report/index.html."
    fi
    echo "Allure report generated at ui/allure-report/index.html."

# Open the generated Allure report in a browser.
ui-report-open:
    cd ui && yarn allure:open

# Serve a temporary Allure report from current test results.
ui-report-serve:
    cd ui && yarn allure:serve

# Run UI lint, type, test, report, and Storybook validation.
ui-quality:
    @just ui-setup
    cd ui && yarn check:theme-contract
    cd ui && yarn test:theme-contract
    cd ui && yarn test:bundle-size
    cd ui && yarn test:typescript-check
    cd ui && yarn check:public-assets
    cd ui && yarn test:public-assets
    cd ui && yarn lint
    cd ui && yarn lint:css
    cd ui && yarn checkTs
    @just ui-test
    @just ui-report
    @just ui-storybook-build

# Remove generated UI docs, reports, and generated CSS.
ui-clean:
    rm -rf ui/storybook-static ui/allure-results ui/allure-report ui/src/styles/generated/tailwindcss.css

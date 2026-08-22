import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import {
  formatBundleSizeReport,
  inspectBundleSize,
} from './check-bundle-size.mjs';

const createBuild = ({ javascript, styles }) => {
  const directory = mkdtempSync(join(tmpdir(), 'ui-bundle-budget-'));
  mkdirSync(join(directory, 'static', 'js'), { recursive: true });
  mkdirSync(join(directory, 'static', 'css'), { recursive: true });
  writeFileSync(join(directory, 'static', 'js', 'main.js'), javascript);
  writeFileSync(join(directory, 'static', 'css', 'main.css'), styles);
  writeFileSync(
    join(directory, 'asset-manifest.json'),
    JSON.stringify({
      entrypoints: ['static/css/main.css', 'static/js/main.js'],
    }),
  );
  return directory;
};

test('accepts an entrypoint within every gzip budget', t => {
  const directory = createBuild({
    javascript: 'const value = 1;',
    styles: 'a{}',
  });
  t.after(() => rmSync(directory, { recursive: true, force: true }));

  const report = inspectBundleSize(directory, {
    initialJavaScriptGzip: 100,
    initialStylesGzip: 100,
    totalInitialGzip: 200,
  });

  assert.deepEqual(report.violations, []);
  assert.match(formatBundleSizeReport(report), /budgets passed/i);
});

test('reports the exact metric that exceeds its budget', t => {
  const directory = createBuild({
    javascript: Array.from({ length: 200 }, (_, index) => `${index},`).join(''),
    styles: 'a{}',
  });
  t.after(() => rmSync(directory, { recursive: true, force: true }));

  const report = inspectBundleSize(directory, {
    initialJavaScriptGzip: 20,
    initialStylesGzip: 100,
    totalInitialGzip: 1_000,
  });

  assert.deepEqual(
    report.violations.map(violation => violation.metric),
    ['initialJavaScriptGzip'],
  );
  assert.match(formatBundleSizeReport(report), /exceeds its budget/i);
});

#!/usr/bin/env node

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { gzipSync } from 'node:zlib';

export const DEFAULT_BUNDLE_BUDGETS = Object.freeze({
  initialJavaScriptGzip: 1_450_000,
  initialStylesGzip: 110_000,
  totalInitialGzip: 1_560_000,
});

const readBudget = (name, fallback) => {
  const configured = Number(process.env[name]);
  return Number.isFinite(configured) && configured > 0 ? configured : fallback;
};

export const getConfiguredBundleBudgets = () => ({
  initialJavaScriptGzip: readBudget(
    'UI_BUDGET_INITIAL_JS_GZIP',
    DEFAULT_BUNDLE_BUDGETS.initialJavaScriptGzip,
  ),
  initialStylesGzip: readBudget(
    'UI_BUDGET_INITIAL_CSS_GZIP',
    DEFAULT_BUNDLE_BUDGETS.initialStylesGzip,
  ),
  totalInitialGzip: readBudget(
    'UI_BUDGET_TOTAL_INITIAL_GZIP',
    DEFAULT_BUNDLE_BUDGETS.totalInitialGzip,
  ),
});

const getGzipSize = filePath =>
  gzipSync(readFileSync(filePath), { level: 9 }).length;

export const inspectBundleSize = (
  buildDirectory,
  budgets = DEFAULT_BUNDLE_BUDGETS,
) => {
  const manifestPath = resolve(buildDirectory, 'asset-manifest.json');
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  const entrypoints = Array.isArray(manifest.entrypoints)
    ? manifest.entrypoints
    : [];

  const assets = entrypoints.map(relativePath => ({
    path: relativePath,
    gzipBytes: getGzipSize(resolve(buildDirectory, relativePath)),
  }));
  const initialJavaScriptGzip = assets
    .filter(asset => asset.path.endsWith('.js'))
    .reduce((total, asset) => total + asset.gzipBytes, 0);
  const initialStylesGzip = assets
    .filter(asset => asset.path.endsWith('.css'))
    .reduce((total, asset) => total + asset.gzipBytes, 0);
  const totalInitialGzip = initialJavaScriptGzip + initialStylesGzip;
  const metrics = {
    initialJavaScriptGzip,
    initialStylesGzip,
    totalInitialGzip,
  };
  const violations = Object.entries(metrics)
    .filter(([metric, bytes]) => bytes > budgets[metric])
    .map(([metric, bytes]) => ({ metric, bytes, budget: budgets[metric] }));

  return { assets, budgets, metrics, violations };
};

const formatBytes = bytes => `${(bytes / 1024).toFixed(1)} KiB`;

export const formatBundleSizeReport = report => {
  const rows = Object.entries(report.metrics).map(([metric, bytes]) => {
    const budget = report.budgets[metric];
    return `${metric}: ${formatBytes(bytes)} / ${formatBytes(budget)}`;
  });

  if (report.violations.length === 0) {
    rows.push('Bundle-size budgets passed.');
  } else {
    rows.push(
      ...report.violations.map(
        violation =>
          `${violation.metric} exceeds its budget by ${formatBytes(
            violation.bytes - violation.budget,
          )}.`,
      ),
    );
  }

  return rows.join('\n');
};

if (
  process.argv[1] &&
  resolve(process.argv[1]) === resolve(import.meta.filename)
) {
  try {
    const report = inspectBundleSize(
      resolve(process.cwd(), 'build'),
      getConfiguredBundleBudgets(),
    );
    console.log(formatBundleSizeReport(report));
    if (report.violations.length > 0) process.exitCode = 1;
  } catch (error) {
    console.error(`Unable to inspect UI bundle: ${error.message}`);
    process.exitCode = 1;
  }
}

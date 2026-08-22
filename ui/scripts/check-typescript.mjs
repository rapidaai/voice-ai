#!/usr/bin/env node

import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const positionedDiagnosticPattern = /^(.+)\(\d+,\d+\): error (TS\d+): (.+)$/;
const typeScriptErrorPattern = /(?:^|:\s)error TS\d+:/;

export const parseTypeScriptDiagnostics = output => {
  const diagnostics = new Map();

  for (const line of output.split('\n')) {
    const match = line.match(positionedDiagnosticPattern);
    if (!match) continue;
    const [, file, code, message] = match;
    const fingerprint = `${file}|${code}|${message}`;
    diagnostics.set(fingerprint, (diagnostics.get(fingerprint) ?? 0) + 1);
  }

  return diagnostics;
};

export const findUnparsedTypeScriptErrors = output =>
  output
    .split('\n')
    .filter(
      line =>
        typeScriptErrorPattern.test(line) &&
        !positionedDiagnosticPattern.test(line),
    );

export const findUnexpectedDiagnostics = (current, baseline) =>
  [...current.entries()]
    .filter(([fingerprint, count]) => count > (baseline.get(fingerprint) ?? 0))
    .map(([fingerprint, count]) => ({
      fingerprint,
      count,
      baseline: baseline.get(fingerprint) ?? 0,
    }));

export const isCompleteTypeScriptRun = (result, diagnostics) =>
  result.signal === null &&
  ((result.status === 0 && diagnostics.size === 0) ||
    (result.status === 2 && diagnostics.size > 0));

const mapToSortedObject = diagnostics =>
  Object.fromEntries(
    [...diagnostics.entries()].sort(([left], [right]) =>
      left.localeCompare(right),
    ),
  );

const objectToMap = value =>
  new Map(
    Object.entries(value).filter(
      ([fingerprint, count]) =>
        typeof fingerprint === 'string' &&
        typeof count === 'number' &&
        Number.isInteger(count) &&
        count > 0,
    ),
  );

const getDiagnosticCount = diagnostics =>
  [...diagnostics.values()].reduce((total, count) => total + count, 0);

export const runTypeScriptCheck = ({ update = false } = {}) => {
  const baselinePath = resolve(
    process.cwd(),
    'scripts/typescript-baseline.json',
  );
  const compilerPath = resolve(
    process.cwd(),
    'node_modules/typescript/bin/tsc',
  );
  const result = spawnSync(
    process.execPath,
    [compilerPath, '--noEmit', '--pretty', 'false'],
    {
      cwd: process.cwd(),
      encoding: 'utf8',
      maxBuffer: 20 * 1024 * 1024,
    },
  );
  const output = `${result.stdout ?? ''}${result.stderr ?? ''}`;
  const diagnostics = parseTypeScriptDiagnostics(output);
  const unparsedErrors = findUnparsedTypeScriptErrors(output);

  if (result.error) {
    console.error(output || result.error.message);
    return 1;
  }

  if (unparsedErrors.length > 0) {
    console.error('Unbaselined TypeScript compiler errors detected:');
    unparsedErrors.slice(0, 25).forEach(error => console.error(`- ${error}`));
    return 1;
  }

  if (!isCompleteTypeScriptRun(result, diagnostics)) {
    console.error(output || 'TypeScript did not complete.');
    return 1;
  }

  if (update) {
    writeFileSync(
      baselinePath,
      `${JSON.stringify(mapToSortedObject(diagnostics), null, 2)}\n`,
    );
    console.log(
      `Updated TypeScript baseline with ${getDiagnosticCount(diagnostics)} diagnostics.`,
    );
    return 0;
  }

  const baseline = objectToMap(JSON.parse(readFileSync(baselinePath, 'utf8')));
  const unexpected = findUnexpectedDiagnostics(diagnostics, baseline);
  const currentCount = getDiagnosticCount(diagnostics);
  const baselineCount = getDiagnosticCount(baseline);

  console.log(
    `TypeScript diagnostics: ${currentCount} current / ${baselineCount} baseline.`,
  );

  if (unexpected.length > 0) {
    console.error('New TypeScript diagnostics detected:');
    unexpected.slice(0, 25).forEach(item => {
      console.error(
        `- ${item.fingerprint} (${item.count} current / ${item.baseline} baseline)`,
      );
    });
    return 1;
  }

  console.log('No new TypeScript diagnostics.');
  return 0;
};

if (
  process.argv[1] &&
  resolve(process.argv[1]) === resolve(import.meta.filename)
) {
  process.exitCode = runTypeScriptCheck({
    update: process.argv.includes('--update'),
  });
}

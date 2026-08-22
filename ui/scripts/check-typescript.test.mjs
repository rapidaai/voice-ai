import assert from 'node:assert/strict';
import test from 'node:test';

import {
  findUnparsedTypeScriptErrors,
  findUnexpectedDiagnostics,
  isCompleteTypeScriptRun,
  parseTypeScriptDiagnostics,
} from './check-typescript.mjs';

test('normalizes line changes into stable diagnostic fingerprints', () => {
  const diagnostics = parseTypeScriptDiagnostics(
    [
      "src/example.ts(10,2): error TS2322: Type 'string' is not assignable to type 'number'.",
      "src/example.ts(20,5): error TS2322: Type 'string' is not assignable to type 'number'.",
    ].join('\n'),
  );

  assert.deepEqual(
    [...diagnostics.entries()],
    [
      [
        "src/example.ts|TS2322|Type 'string' is not assignable to type 'number'.",
        2,
      ],
    ],
  );
});

test('rejects new diagnostics while allowing baseline reductions', () => {
  const baseline = new Map([['src/existing.ts|TS2322|Existing error.', 2]]);
  const current = new Map([
    ['src/existing.ts|TS2322|Existing error.', 1],
    ['src/new.ts|TS2304|New error.', 1],
  ]);

  assert.deepEqual(findUnexpectedDiagnostics(current, baseline), [
    {
      fingerprint: 'src/new.ts|TS2304|New error.',
      count: 1,
      baseline: 0,
    },
  ]);
});

test('rejects positionless compiler errors mixed with baselined diagnostics', () => {
  const output = [
    'src/existing.ts(10,2): error TS2322: Existing error.',
    "error TS5023: Unknown compiler option 'unsupportedOption'.",
  ].join('\n');

  assert.equal(parseTypeScriptDiagnostics(output).size, 1);
  assert.deepEqual(findUnparsedTypeScriptErrors(output), [
    "error TS5023: Unknown compiler option 'unsupportedOption'.",
  ]);
});

test('accepts complete compiler runs with the expected exit status', () => {
  assert.equal(
    isCompleteTypeScriptRun(
      { status: 2, signal: null },
      new Map([['src/existing.ts|TS2322|Existing error.', 1]]),
    ),
    true,
  );
  assert.equal(
    isCompleteTypeScriptRun({ status: 0, signal: null }, new Map()),
    true,
  );
});

test('rejects partial compiler output from crashes or termination', () => {
  const diagnostics = new Map([['src/existing.ts|TS2322|Existing error.', 1]]);

  assert.equal(
    isCompleteTypeScriptRun({ status: 1, signal: null }, diagnostics),
    false,
  );
  assert.equal(
    isCompleteTypeScriptRun({ status: null, signal: 'SIGTERM' }, diagnostics),
    false,
  );
});

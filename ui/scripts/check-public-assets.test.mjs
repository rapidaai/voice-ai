import assert from 'node:assert/strict';
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

import {
  checkPublicAssets,
  duplicateContentKey,
} from './check-public-assets.mjs';

const checkerPath = path.resolve(
  import.meta.dirname,
  'check-public-assets.mjs',
);

function createFixtureRepo(context, files = {}) {
  const repoRoot = path.join(
    tmpdir(),
    `public-assets-${process.pid}-${Date.now()}-${Math.random()}`,
  );
  mkdirSync(path.join(repoRoot, 'ui', 'public'), { recursive: true });
  context.after(() => rmSync(repoRoot, { recursive: true }));

  for (const [relativePath, content] of Object.entries(files)) {
    const filePath = path.join(repoRoot, 'ui', 'public', relativePath);
    mkdirSync(path.dirname(filePath), { recursive: true });
    writeFileSync(filePath, content);
  }

  return repoRoot;
}

test('accepts assets within budgets with unique content', context => {
  const repoRoot = createFixtureRepo(context, {
    'images/logo.svg': '<svg>logo</svg>',
    'robots.txt': 'User-agent: *',
  });

  assert.deepEqual(
    checkPublicAssets(repoRoot, { maxTotalBytes: 100, maxFileBytes: 50 }),
    [],
  );
});

test('rejects total and per-file budget violations', context => {
  const repoRoot = createFixtureRepo(context, {
    'large.bin': Buffer.alloc(6, 1),
    'small.bin': Buffer.alloc(5, 2),
  });

  assert.deepEqual(
    checkPublicAssets(repoRoot, { maxTotalBytes: 10, maxFileBytes: 5 }),
    [
      'ui/public/large.bin: 6 bytes exceeds per-file budget of 5 bytes',
      'ui/public: 11 bytes exceeds total budget of 10 bytes',
    ],
  );
});

test('rejects duplicate content with deterministic diagnostics', context => {
  const repoRoot = createFixtureRepo(context, {
    'z/copy.png': 'duplicate',
    'a/original.png': 'duplicate',
    'unique.png': 'unique',
  });

  const firstResult = checkPublicAssets(repoRoot);
  const secondResult = checkPublicAssets(repoRoot);

  assert.deepEqual(firstResult, [
    'duplicate public asset content: ui/public/a/original.png, ui/public/z/copy.png',
  ]);
  assert.deepEqual(secondResult, firstResult);
});

test('allows an explicitly listed duplicate-content group', context => {
  const repoRoot = createFixtureRepo(context, {
    'a.png': 'shared',
    'b.png': 'shared',
  });
  const duplicateAllowlist = [['ui/public/b.png', 'ui/public/a.png']];

  assert.equal(
    duplicateContentKey(duplicateAllowlist[0]),
    'ui/public/a.png\nui/public/b.png',
  );
  assert.deepEqual(checkPublicAssets(repoRoot, { duplicateAllowlist }), []);
});

test('reports a missing public directory', context => {
  const repoRoot = createFixtureRepo(context);
  const missingRoot = path.join(repoRoot, 'missing');

  assert.deepEqual(checkPublicAssets(missingRoot), [
    'ui/public: directory does not exist',
  ]);
});

test('CLI exits nonzero and reports violations', context => {
  const repoRoot = createFixtureRepo(context, {
    'a.png': 'duplicate',
    'b.png': 'duplicate',
  });
  const result = spawnSync(process.execPath, [checkerPath, repoRoot], {
    encoding: 'utf8',
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Public asset check failed:/);
  assert.match(result.stderr, /duplicate public asset content/);
});

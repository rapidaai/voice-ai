import { createHash } from 'node:crypto';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const MAX_TOTAL_BYTES = 4 * 1024 * 1024;
export const MAX_FILE_BYTES = 1024 * 1024;
export const DUPLICATE_CONTENT_ALLOWLIST = [];

function toPosixPath(filePath) {
  return filePath.split(path.sep).join('/');
}

function listFiles(directoryPath) {
  return readdirSync(directoryPath, { withFileTypes: true })
    .sort((left, right) => left.name.localeCompare(right.name))
    .flatMap(entry => {
      const entryPath = path.join(directoryPath, entry.name);
      return entry.isDirectory() ? listFiles(entryPath) : [entryPath];
    });
}

export function duplicateContentKey(assetPaths) {
  return [...assetPaths].sort().join('\n');
}

export function checkPublicAssets(
  repoRoot,
  {
    maxTotalBytes = MAX_TOTAL_BYTES,
    maxFileBytes = MAX_FILE_BYTES,
    duplicateAllowlist = DUPLICATE_CONTENT_ALLOWLIST,
  } = {},
) {
  const publicDirectory = path.join(repoRoot, 'ui', 'public');
  if (!existsSync(publicDirectory)) {
    return ['ui/public: directory does not exist'];
  }

  const diagnostics = [];
  const filesByHash = new Map();
  let totalBytes = 0;

  for (const filePath of listFiles(publicDirectory)) {
    const relativePath = toPosixPath(path.relative(repoRoot, filePath));
    const fileSize = statSync(filePath).size;
    totalBytes += fileSize;

    if (fileSize > maxFileBytes) {
      diagnostics.push(
        `${relativePath}: ${fileSize} bytes exceeds per-file budget of ${maxFileBytes} bytes`,
      );
    }

    const digest = createHash('sha256')
      .update(readFileSync(filePath))
      .digest('hex');
    const duplicatePaths = filesByHash.get(digest) ?? [];
    duplicatePaths.push(relativePath);
    filesByHash.set(digest, duplicatePaths);
  }

  if (totalBytes > maxTotalBytes) {
    diagnostics.push(
      `ui/public: ${totalBytes} bytes exceeds total budget of ${maxTotalBytes} bytes`,
    );
  }

  const allowedDuplicates = new Set(
    duplicateAllowlist.map(duplicateContentKey),
  );
  for (const duplicatePaths of filesByHash.values()) {
    if (
      duplicatePaths.length > 1 &&
      !allowedDuplicates.has(duplicateContentKey(duplicatePaths))
    ) {
      diagnostics.push(
        `duplicate public asset content: ${duplicatePaths.sort().join(', ')}`,
      );
    }
  }

  return diagnostics.sort();
}

const isMainModule =
  process.argv[1] &&
  path.resolve(process.argv[1]) ===
    path.resolve(fileURLToPath(import.meta.url));

if (isMainModule) {
  const repoRoot = path.resolve(
    process.argv[2] ?? path.join(import.meta.dirname, '..', '..'),
  );
  const diagnostics = checkPublicAssets(repoRoot);

  if (diagnostics.length > 0) {
    console.error('Public asset check failed:');
    for (const diagnostic of diagnostics) {
      console.error(`- ${diagnostic}`);
    }
    process.exitCode = 1;
  } else {
    console.log('Public asset check passed.');
  }
}

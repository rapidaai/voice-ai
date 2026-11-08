import fs from 'fs';
import path from 'path';

const sourceRoot = path.resolve(__dirname, '../../../..');
const legacyHeaderNames = [
  ['page', 'header', 'block'].join('-'),
  ['Page', 'Header', 'Block'].join(''),
];

function findLegacyHeaderReferences(directory: string): string[] {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) return findLegacyHeaderReferences(entryPath);
    if (!/\.(ts|tsx)$/.test(entry.name)) return [];

    const source = fs.readFileSync(entryPath, 'utf8');
    return legacyHeaderNames.some(name => source.includes(name))
      ? [path.relative(sourceRoot, entryPath)]
      : [];
  });
}

describe('page header component contract', () => {
  it('does not allow the removed legacy page header to return', () => {
    expect(findLegacyHeaderReferences(sourceRoot)).toEqual([]);
  });
});

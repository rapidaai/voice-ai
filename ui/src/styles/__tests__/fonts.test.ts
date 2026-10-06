import { readFileSync } from 'fs';
import { resolve } from 'path';

import packageJson from '../../../package.json';

describe('application fonts', () => {
  it('bundles deterministic IBM Plex font files', () => {
    expect(packageJson.dependencies['@ibm/plex-sans']).toBe('1.1.0');
    expect(packageJson.dependencies['@ibm/plex-mono']).toBe('1.1.0');

    const globalStyles = readFileSync(
      resolve(process.cwd(), 'src/styles/global-styles.ts'),
      'utf8',
    );
    expect(globalStyles).toMatch(/import ['"]\.\/fonts\.scss['"]/);

    const fontStyles = readFileSync(
      resolve(process.cwd(), 'src/styles/fonts.scss'),
      'utf8',
    );
    expect(fontStyles).toMatch(
      /@ibm\/plex-sans\/fonts\/complete\/woff2/,
    );
    expect(fontStyles).toMatch(
      /@ibm\/plex-mono\/fonts\/complete\/woff2/,
    );
    expect(fontStyles).not.toMatch(/\blocal\(/);
  });
});

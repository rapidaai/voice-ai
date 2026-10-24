import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

describe('Carbon theme overrides', () => {
  it('keeps the native Carbon primary-button ramp in dark mode', () => {
    const styles = readFileSync(
      resolve(process.cwd(), 'src/styles/carbon.scss'),
      'utf8',
    );
    const darkTheme = styles.match(
      /:root\[data-color-mode='dark'\]\s*\{([\s\S]*?)\n\}/,
    )?.[1];

    expect(darkTheme).toContain('@include carbon-theme.theme(themes.$g100)');
    expect(darkTheme).not.toMatch(/--cds-button-primary(?:-hover|-active)?:/);
  });
});

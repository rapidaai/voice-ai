import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const stylePath = (file: string) => resolve(process.cwd(), 'src/styles', file);
const readStyles = (file: string) => readFileSync(stylePath(file), 'utf8');

describe('external style adapters', () => {
  it('loads only the registered third-party adapters', () => {
    const imports = readStyles('global-styles.ts');

    expect(imports).toContain("import '@carbon/charts-react/styles.css'");
    expect(imports).toContain("import './externals/editors.css'");
    expect(imports).toContain("import './externals/flatpickr.css'");
    expect(imports).toContain("import './externals/voice-visualizer.css'");
    expect(imports).not.toContain('custom.css');
    expect(existsSync(stylePath('custom.css'))).toBe(false);
  });

  it('uses Carbon roles instead of independent adapter colors', () => {
    for (const file of [
      'externals/editors.css',
      'externals/flatpickr.css',
      'externals/voice-visualizer.css',
    ]) {
      const styles = readStyles(file);

      expect(styles).toContain('var(--cds-');
      expect(styles).not.toMatch(/#[0-9a-f]{3,8}\b|\b(?:rgba?|hsla?|oklch)\(/i);
    }
  });

  it('has no remaining component CSS modules', () => {
    expect(
      existsSync(
        resolve(
          process.cwd(),
          'src/app/components/domain/configuration/style.module.css',
        ),
      ),
    ).toBe(false);
    expect(
      existsSync(
        resolve(
          process.cwd(),
          'src/app/components/domain/configuration/config-var/style.module.css',
        ),
      ),
    ).toBe(false);
  });
});

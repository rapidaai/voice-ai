import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const readStyles = (file: string) =>
  readFileSync(resolve(process.cwd(), `src/styles/${file}`), 'utf8');

describe('Carbon theme adapters', () => {
  it('installs every configured Carbon preset', () => {
    const styles = readStyles('carbon.scss');

    for (const preset of ['white', 'g10', 'g90', 'g100']) {
      expect(styles).toContain(`:root[data-carbon-theme='${preset}']`);
      expect(styles).toContain(
        `@include carbon-theme.theme(themes.$${preset})`,
      );
    }
  });

  it('preserves the version 1 dark primary ramp and maps version 2 brand actions', () => {
    const styles = readStyles('carbon.scss');
    const version1Dark = styles.match(
      /:root\[data-theme-schema='1'\]\[data-color-mode='dark'\]\s*\{([\s\S]*?)\n\}/,
    )?.[1];
    const version2 = styles.match(
      /:root\[data-theme-schema='2'\]\s*\{([\s\S]*?)\n\}/,
    )?.[1];

    expect(version1Dark).toContain('--cds-background: #0a0a0a');
    expect(version1Dark).toContain('--cds-layer-01: #161616');
    expect(version1Dark).toContain('--cds-button-primary: #0f62fe');
    expect(version1Dark).toContain('--cds-button-primary-hover: #0353e9');
    expect(version1Dark).toContain('--cds-button-primary-active: #002d9c');
    expect(version2).toContain('--cds-button-primary: var(--brand-primary)');
    expect(version2).toContain(
      '--cds-button-primary-hover: var(--brand-primary-hover)',
    );
    expect(version2).toContain(
      '--cds-button-primary-active: var(--brand-primary-active)',
    );
  });

  it('scopes configured primary foregrounds away from danger controls', () => {
    const styles = readStyles('carbon.scss');
    const version2 = styles.match(
      /:root\[data-theme-schema='2'\]\s*\{([\s\S]*?)\n\}/,
    )?.[1];

    expect(version2).not.toContain('--cds-text-on-color:');
    expect(version2).not.toContain('--cds-icon-on-color:');
    expect(styles).toContain(
      '.cds--btn--primary:not(:disabled):not(.cds--modal-close)',
    );
    expect(styles).toMatch(
      /:root\[data-theme-schema='2'\]\s+\.cds--btn--primary[^{]*{[\s\S]*?--cds-text-on-color: var\(--brand-on-primary\)/,
    );
    expect(styles).not.toMatch(
      /\.cds--btn--danger[^{]*{[\s\S]*?--brand-on-primary/,
    );
  });

  it('maps Tailwind action utilities to the resolved Carbon action ramp', () => {
    const styles = readStyles('tailwind.css');

    expect(styles).toContain('--color-*: initial');
    expect(styles).toContain('--color-primary: var(--cds-button-primary)');
    expect(styles).toContain(
      '--color-primary-hover: var(--cds-button-primary-hover)',
    );
    expect(styles).toContain(
      '--color-primary-active: var(--cds-button-primary-active)',
    );
    expect(styles).toMatch(
      /:root\[data-theme-schema='1'\][^{]*{[\s\S]*?--color-primary: var\(--brand-primary\)/,
    );
    expect(styles).toContain('@utility text-on-primary');
  });

  it('loads Carbon user avatar styles', () => {
    expect(readStyles('carbon.scss')).toContain(
      "@use '@carbon/styles/scss/components/user-avatar'",
    );
  });

  it('uses the configured brand contrast for onboarding checklists', () => {
    const styles = readStyles('carbon.scss');

    expect(styles).toMatch(
      /:root\[data-theme-schema='1'\]\s+\.app-onboarding-checklist\s*{[^}]*--app-checklist-foreground: var\(--brand-on-primary\)/,
    );
    expect(styles).toMatch(
      /:root\[data-theme-schema='2'\]\s+\.app-onboarding-checklist\s*{[^}]*--app-checklist-foreground: var\(--brand-on-primary\)/,
    );
  });
});

import { readFileSync } from 'fs';
import { resolve } from 'path';

describe('Carbon component styles', () => {
  it('loads styles for rendered Carbon components', () => {
    const carbonStyles = readFileSync(
      resolve(process.cwd(), 'src/styles/carbon.scss'),
      'utf8',
    );

    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/inline-loading';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/contained-list';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/accordion';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/structured-list';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/fluid-text-input';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/fluid-text-area';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/fluid-number-input';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/data-table/expandable';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/data-table/skeleton';",
    );
    expect(carbonStyles).toContain('.debugger-configuration-accordion');
    expect(carbonStyles).toContain('border-block-start: 0;');
    expect(carbonStyles).toContain('margin-inline-start: 0;');
    expect(carbonStyles).toContain('padding-inline: 0;');
    expect(carbonStyles).toContain('padding-block: 0;');
  });

  it('loads only the next coachmark styles', () => {
    const coachmarkStyles = readFileSync(
      resolve(process.cwd(), 'src/styles/carbon.scss'),
      'utf8',
    );

    expect(coachmarkStyles).toContain(
      "@use '@carbon/ibm-products/scss/components/Coachmark/coachmark_next';",
    );
    expect(coachmarkStyles).toContain('inline-size: 290px;');
  });

  it('keeps connected provider model controls on one bottom border', () => {
    const carbonStyles = readFileSync(
      resolve(process.cwd(), 'src/styles/carbon.scss'),
      'utf8',
    );
    const connectedProviderStyles = carbonStyles.match(
      /\.text-provider-combo-row,\s*\.domain-connected-dropdown-row\s*\{([\s\S]*?)\n\}/,
    )?.[1];

    expect(connectedProviderStyles).toMatch(
      /\.cds--dropdown,\s*\.cds--list-box,\s*\.cds--list-box__field\s*\{[\s\S]*?border: none;/,
    );
    expect(carbonStyles).toMatch(
      /\.text-provider-combo-row\s*\{\s*\.cds--combo-box \.cds--text-input\s*\{\s*border-block-end: none;/,
    );
  });

  it('loads the Carbon tearsheet without role-specific geometry', () => {
    const carbonStyles = readFileSync(
      resolve(process.cwd(), 'src/styles/carbon.scss'),
      'utf8',
    );

    expect(carbonStyles).toContain(
      "@use '@carbon/ibm-products/scss/components/Tearsheet';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/ibm-products/scss/components/AddSelect';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/ibm-products/scss/components/EmptyStates';",
    );
    expect(carbonStyles).toContain(
      "@use '@carbon/ibm-products/scss/components/FullPageError';",
    );
    expect(carbonStyles).not.toContain('.role-create-modal');
  });

  it('inherits shared contextual navigation typography for every item state', () => {
    const navigationStyles = readFileSync(
      resolve(process.cwd(), 'src/styles/carbon.scss'),
      'utf8',
    );

    expect(navigationStyles).toMatch(
      /\.app-side-nav :is\(a, button, span\)\s*{\s*font: inherit !important;\s*letter-spacing: inherit !important;/,
    );
    expect(navigationStyles).not.toContain(
      '.cds--side-nav__menu-item .cds--side-nav__link-text',
    );
  });
});

import { readFileSync } from 'fs';
import { resolve } from 'path';

describe('Carbon component styles', () => {
  it('loads styles for components rendered by the voice debugger', () => {
    const carbonStyles = readFileSync(
      resolve(process.cwd(), 'src/styles/carbon.scss'),
      'utf8',
    );

    expect(carbonStyles).toContain(
      "@use '@carbon/react/scss/components/inline-loading';",
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
    expect(carbonStyles).toContain(
      "@use '@carbon/ibm-products/scss/components/SidePanel';",
    );
    expect(carbonStyles).toContain('.debugger-configuration-accordion');
    expect(carbonStyles).toContain('border-block-start: 0;');
    expect(carbonStyles).toContain('margin-inline-start: 0;');
    expect(carbonStyles).toContain('padding-inline: 0;');
    expect(carbonStyles).toContain('padding-block: 0;');
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
    expect(carbonStyles).not.toContain('.role-create-modal');
  });
});

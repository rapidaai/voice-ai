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
    expect(carbonStyles).toContain('.debugger-configuration-accordion');
    expect(carbonStyles).toContain('border-block-start: 0;');
    expect(carbonStyles).toContain('margin-inline-start: 0;');
    expect(carbonStyles).toContain('padding-inline: 0;');
    expect(carbonStyles).toContain('padding-block: 0;');
  });
});

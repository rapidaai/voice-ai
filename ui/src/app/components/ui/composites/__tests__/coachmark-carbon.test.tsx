import React, { createRef } from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { FeatureFlags } from '@carbon/react';

import { Coachmark } from '../coachmark';

describe('Coachmark Carbon v12 integration', () => {
  it('keeps the launcher ref and expanded state with dynamic floating styles', async () => {
    const launcherButtonRef = createRef<HTMLButtonElement>();

    render(
      <FeatureFlags
        enableV12Autoalign
        enableV12DynamicFloatingStyles
        enableV12Overflowmenu
      >
        <Coachmark
          actionLabel="Done"
          align="bottom"
          description="Description"
          launcherButtonRef={launcherButtonRef}
          onAction={jest.fn()}
          onClose={jest.fn()}
          open
          title="Tour step"
        >
          <button ref={launcherButtonRef} type="button">
            Target
          </button>
        </Coachmark>
      </FeatureFlags>,
    );

    await screen.findByRole('region', { name: 'Tour step coachmark' });

    expect(screen.getByRole('button', { name: 'Target' })).toHaveAttribute(
      'aria-expanded',
      'true',
    );
  });
});

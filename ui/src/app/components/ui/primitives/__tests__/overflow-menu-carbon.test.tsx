import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { FeatureFlags } from '@carbon/react';

import { OverflowMenu, OverflowMenuItem } from '../overflow-menu';

describe('OverflowMenu Carbon v12 integration', () => {
  it('opens, runs an action, and closes without targeted migration notices', async () => {
    const user = userEvent.setup();
    const onClick = jest.fn();
    const consoleInfo = jest.spyOn(console, 'info').mockImplementation();

    render(
      <FeatureFlags
        enableV12Overflowmenu
        enableV12Autoalign
        enableV12DynamicFloatingStyles
      >
        <OverflowMenu
          className="app-overflow-menu-test [&_.cds--overflow-menu]:h-8"
          size="sm"
          iconDescription="Actions"
        >
          <OverflowMenuItem itemText="Edit" onClick={onClick} />
        </OverflowMenu>
      </FeatureFlags>,
    );

    const trigger = screen.getByRole('button', { name: 'Actions' });
    const container = trigger.closest('.app-overflow-menu-test');
    expect(container).toHaveClass(
      'app-overflow-menu-test',
      '[&_.cds--overflow-menu]:h-8',
    );
    expect(trigger).toHaveClass('cds--overflow-menu');

    await user.click(trigger);
    await user.click(await screen.findByRole('menuitem', { name: 'Edit' }));

    expect(onClick).toHaveBeenCalledTimes(1);
    await waitFor(() =>
      expect(
        screen.queryByRole('menuitem', { name: 'Edit' }),
      ).not.toBeInTheDocument(),
    );

    const notices = consoleInfo.mock.calls.flat().join('\n');
    expect(notices).not.toContain('enable-v12-overflowmenu');
    expect(notices).not.toContain('enable-v12-autoalign');
    expect(notices).not.toContain('enable-v12-dynamic-floating-styles');
  });
});

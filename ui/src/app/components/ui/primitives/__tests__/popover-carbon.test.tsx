import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { Popover, PopoverContent } from '../popover';

describe('Popover Carbon integration', () => {
  it('renders caretless content without passing Carbon the deprecated caret prop', () => {
    const consoleError = jest.spyOn(console, 'error').mockImplementation();

    render(
      <Popover align="bottom-end" open>
        <button type="button">Open context</button>
        <PopoverContent>Context content</PopoverContent>
      </Popover>,
    );

    expect(screen.getByText('Context content')).toBeVisible();
    expect(
      screen
        .getByRole('button', { name: 'Open context' })
        .closest('.cds--popover-container'),
    ).toHaveClass(
      'app-popover--without-caret',
      '[--cds-popover-offset:0.25rem]!',
      '[&>.cds--popover>.cds--popover-caret]:!hidden',
      '[&>.cds--popover>.cds--popover-content>.cds--popover-caret]:!hidden',
    );
    expect(consoleError.mock.calls.flat().join('\n')).not.toContain(
      'The `caret` prop has been deprecated',
    );
    consoleError.mockRestore();
  });
});

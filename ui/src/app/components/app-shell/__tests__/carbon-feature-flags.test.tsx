import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { CarbonFeatureFlags } from '../carbon-feature-flags';

jest.mock('@carbon/react', () => ({
  FeatureFlags: ({ children, ...flags }: any) => (
    <div data-testid="carbon-feature-flags" data-flags={JSON.stringify(flags)}>
      {children}
    </div>
  ),
}));

describe('CarbonFeatureFlags', () => {
  it('enables the targeted v12 floating component behavior', () => {
    render(
      <CarbonFeatureFlags>
        <span>Application</span>
      </CarbonFeatureFlags>,
    );

    const flags = JSON.parse(
      screen.getByTestId('carbon-feature-flags').getAttribute('data-flags')!,
    );

    expect(flags).toEqual({
      enableV12Autoalign: true,
      enableV12DynamicFloatingStyles: true,
      enableV12Overflowmenu: true,
    });
    expect(screen.getByText('Application')).toBeInTheDocument();
  });

  it('does not enable the complete v12 release early', () => {
    render(
      <CarbonFeatureFlags>
        <span>Application</span>
      </CarbonFeatureFlags>,
    );

    const flags = JSON.parse(
      screen.getByTestId('carbon-feature-flags').getAttribute('data-flags')!,
    );

    expect(flags).not.toHaveProperty('enableV12Release');
  });
});

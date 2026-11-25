import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { typographyClassNames } from '@/app/components/ui/primitives/text';
import {
  SideNavigationCollapseControl,
  SideNavigationPanel,
  SideNavigationSectionLabel,
} from '../side-navigation-patterns';

describe('side navigation patterns', () => {
  it('uses the shared expanded width, section treatment, and collapse control', async () => {
    const user = userEvent.setup();
    const onClick = jest.fn();

    render(
      <SideNavigationPanel data-testid="panel" expanded>
        <SideNavigationSectionLabel expanded>
          Settings
        </SideNavigationSectionLabel>
        <SideNavigationCollapseControl
          aria-label="Collapse navigation"
          expanded
          onClick={onClick}
        />
      </SideNavigationPanel>,
    );

    expect(screen.getByTestId('panel')).toHaveStyle({ width: '13.5rem' });
    expect(screen.getByText('Settings')).toHaveClass(
      ...typographyClassNames['label-01'].split(' '),
      'text-helper',
    );
    expect(screen.getByText('Collapse')).toHaveClass(
      ...typographyClassNames['label-01'].split(' '),
    );
    expect(screen.getByRole('button')).toHaveClass(
      'app-side-navigation-collapse-control',
    );
    expect(screen.getByRole('button').querySelector('svg')).toHaveAttribute(
      'width',
      '16',
    );

    await user.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('keeps the rail width and hides section and control labels when collapsed', () => {
    render(
      <SideNavigationPanel data-testid="panel" expanded={false}>
        <SideNavigationSectionLabel expanded={false}>
          Settings
        </SideNavigationSectionLabel>
        <SideNavigationCollapseControl
          aria-label="Expand navigation"
          expanded={false}
          onClick={jest.fn()}
        />
      </SideNavigationPanel>,
    );

    expect(screen.getByTestId('panel')).toHaveStyle({ width: '3rem' });
    expect(screen.getByText('Settings')).toHaveClass('h-0', 'opacity-0');
    expect(screen.getByText('Expand')).toHaveClass('w-0', 'opacity-0');
  });
});

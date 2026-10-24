import '@testing-library/jest-dom';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { DeploymentEditTabs } from '../deployment-edit-tabs';
import { DeploymentEditHeader } from '../deployment-edit-header';
import { typographyClassNames } from '@/app/components/ui/primitives/text';

const tabs = [
  { code: 'general', name: 'General' },
  { code: 'voice', name: 'Voice' },
] as const;

function TestEditor() {
  const [activeTab, setActiveTab] =
    useState<(typeof tabs)[number]['code']>('general');

  return (
    <DeploymentEditTabs
      tabs={tabs}
      activeTab={activeTab}
      onChange={setActiveTab}
      aria-label="Deployment edit tabs"
      panels={{
        general: <div>General settings</div>,
        voice: <div>Voice settings</div>,
      }}
    />
  );
}

describe('DeploymentEditTabs', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: true,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
  });

  it('renders the selected editor inside an unpadded Carbon tab panel', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<TestEditor />);

    const generalPanel = screen
      .getByText('General settings')
      .closest('[role="tabpanel"]');
    expect(generalPanel).toBe(screen.getByRole('tabpanel'));
    expect(generalPanel).toHaveClass('!w-full', '!p-0');
    const tabList = screen.getByRole('tablist', {
      name: 'Deployment edit tabs',
    });
    expect(tabList.parentElement).toHaveClass(
      'w-full',
      'border-y',
      'border-border-subtle',
    );
    expect(tabList.parentElement).not.toHaveClass('cds--tabs--full-width');
    expect(
      screen.getByText('General settings').closest('[data-deployment-form]'),
    ).toHaveClass('w-full', 'max-w-3xl');

    await user.click(screen.getByRole('tab', { name: 'Voice' }));

    expect(
      screen.getByText('Voice settings').closest('[role="tabpanel"]'),
    ).toBe(screen.getByRole('tabpanel'));
  });

  it('uses the shared Carbon typography for deployment edit headers', () => {
    render(
      <DeploymentEditHeader
        eyebrow="Debugger deployment"
        title="Edit debugger deployment"
        description="Update experience and voice settings."
      />,
    );

    expect(screen.getByText('Debugger deployment')).toHaveClass(
      ...typographyClassNames['label-01'].split(' '),
    );
    expect(
      screen.getByRole('heading', { name: 'Edit debugger deployment' }),
    ).toHaveClass(...typographyClassNames['heading-03'].split(' '));
    expect(
      screen.getByText('Update experience and voice settings.'),
    ).toHaveClass(...typographyClassNames['body-01'].split(' '));
  });
});

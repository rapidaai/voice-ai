import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { AssistantApiDeploymentDialog } from '../assistant-api-deployment-modal';

jest.mock('@/app/components/dialogs/shared', () => ({
  RightSideModal: ({ children, className, label }: any) => (
    <section className={className} aria-label={label}>
      {children}
    </section>
  ),
  DeploymentRow: ({ children, label }: any) => (
    <div>
      {label}
      {children}
    </div>
  ),
  DeploymentSectionHeader: ({ label }: any) => <h3>{label}</h3>,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  CopyButton: ({ children }: any) => <button>{children}</button>,
  Tabs: (props: any) => {
    const React = require('react');
    return (
      <div
        data-contained={String(Boolean(props.contained))}
        data-full-width={String(Boolean(props.fullWidth))}
      >
        <div role="tablist" aria-label={props['aria-label']}>
          {props.tabs.map((tab: string, index: number) => (
            <button
              key={tab}
              role="tab"
              aria-selected={props.selectedIndex === index}
              onClick={() => props.onChange(index)}
            >
              {tab}
            </button>
          ))}
        </div>
        {React.Children.map(props.children, (child, index) => (
          <section hidden={props.selectedIndex !== index}>{child}</section>
        ))}
      </div>
    );
  },
}));

jest.mock('@/app/components/layout/container/message/notice-block', () => ({
  YellowNoticeBlock: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/domain/pills/provider-model-pill', () => ({
  ProviderPill: ({ provider }: any) => <span>{provider}</span>,
}));

jest.mock('@/app/components/dialogs/shared/deployment-audio-panel', () => ({
  DeploymentAudioPanel: () => <div>Shared deployment audio</div>,
}));

jest.mock(
  '@/app/components/dialogs/shared/deployment-configuration-panel',
  () => ({
    DeploymentConfigurationPanel: () => (
      <div>Saved deployment configuration</div>
    ),
  }),
);

jest.mock(
  '@/app/components/domain/integration-document/assistant-deployment-setup',
  () => ({
    ApiDeploymentSetup: () => <div>API setup guidance</div>,
  }),
);

const deployment = {
  getId: () => 'deployment-1',
  getAssistantid: () => 'assistant-1',
  getInputaudio: () => undefined,
  getOutputaudio: () => undefined,
};

describe('assistant deployment detail dialogs', () => {
  it('organizes SDK / API details into setup, configuration, and audio', async () => {
    const user = userEvent.setup();
    render(
      <AssistantApiDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        deployment={deployment as any}
      />,
    );

    const tabList = screen.getByRole('tablist', {
      name: 'SDK / API deployment tabs',
    });
    expect(tabList.parentElement).toHaveAttribute('data-contained', 'true');
    expect(tabList.parentElement).toHaveAttribute('data-full-width', 'true');
    expect(screen.getAllByRole('tab').map(tab => tab.textContent)).toEqual([
      'Setup',
      'Configuration',
      'Audio',
    ]);

    await user.click(screen.getByRole('tab', { name: 'Configuration' }));
    expect(screen.getByText('Saved deployment configuration')).toBeVisible();
    await user.click(screen.getByRole('tab', { name: 'Audio' }));
    expect(screen.getByText('Shared deployment audio')).toBeVisible();
  });
});

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { AssistantWebWidgetlDeploymentDialog } from '../assistant-web-widget-deployment-modal';

jest.mock('@rapidaai/react', () => {
  class AssistantWebpluginDeployment {
    getId() {
      return 'deployment-1';
    }
    getAssistantid() {
      return 'assistant-123';
    }
    getInputaudio() {
      return undefined;
    }
    getOutputaudio() {
      return undefined;
    }
    getName() {
      return 'Website assistant';
    }
    getSuggestionList() {
      return ['Track my order'];
    }
    getHelpcenterenabled() {
      return true;
    }
    getProductcatalogenabled() {
      return false;
    }
    getArticlecatalogenabled() {
      return false;
    }
    getUploadfileenabled() {
      return true;
    }
  }

  return { AssistantWebpluginDeployment };
});

jest.mock('@/app/components/dialogs/shared', () => ({
  RightSideModal: ({ children, className, label, title }: any) => (
    <section className={className} data-testid="widget-details-drawer">
      <div>{label}</div>
      <h2>{title}</h2>
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

jest.mock(
  '@/app/components/domain/integration-document/widget-integration',
  () => ({
    WidgetIntegration: ({ assistantId }: any) => (
      <div>Widget integration for {assistantId}</div>
    ),
  }),
);

jest.mock('@/app/components/dialogs/shared/deployment-audio-panel', () => ({
  DeploymentAudioPanel: () => <div>Deployment audio panel</div>,
}));

jest.mock(
  '@/app/components/dialogs/shared/deployment-configuration-panel',
  () => ({
    DeploymentConfigurationPanel: ({ channelDetails }: any) => (
      <div>
        Saved deployment configuration
        {channelDetails}
      </div>
    ),
  }),
);

describe('AssistantWebWidgetlDeploymentDialog', () => {
  it('uses full-width Carbon tabs for setup, configuration, and audio', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const { AssistantWebpluginDeployment } =
      jest.requireMock('@rapidaai/react');

    render(
      <AssistantWebWidgetlDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        deployment={new AssistantWebpluginDeployment()}
      />,
    );

    expect(screen.getByTestId('widget-details-drawer')).toHaveClass(
      'w-[580px]',
      'max-w-[calc(100vw-2rem)]',
    );
    const tabList = screen.getByRole('tablist', {
      name: 'Web widget deployment tabs',
    });
    expect(tabList.parentElement).toHaveAttribute('data-contained', 'true');
    expect(tabList.parentElement).toHaveAttribute('data-full-width', 'true');
    expect(
      screen.getByText('Widget integration for assistant-123'),
    ).toBeVisible();

    expect(screen.getAllByRole('tab').map(tab => tab.textContent)).toEqual([
      'Setup',
      'Configuration',
      'Audio',
    ]);

    await user.click(screen.getByRole('tab', { name: 'Configuration' }));
    expect(screen.getByText('Saved deployment configuration')).toBeVisible();
    expect(screen.getByText('Website assistant')).toBeVisible();

    await user.click(screen.getByRole('tab', { name: 'Audio' }));
    expect(screen.getByText('Deployment audio panel')).toBeVisible();
  });
});

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { AssistantPhoneCallDeploymentDialog } from '../assistant-phone-call-deployment-modal';

jest.mock('@/configs', () => ({
  CONFIG: {
    connection: {
      media: 'https://voice.client.example',
      sip: undefined,
      socket: undefined,
    },
  },
}));

jest.mock('@/app/components/dialogs/shared', () => ({
  RightSideModal: ({ children, label, title }: any) => (
    <section>
      <div>{label}</div>
      <h2>{title}</h2>
      {children}
    </section>
  ),
  DeploymentRow: ({ children, label }: any) => (
    <div>
      <span>{label}</span>
      {children}
    </div>
  ),
  DeploymentSectionHeader: ({ label }: any) => <h3>{label}</h3>,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  CopyButton: ({ children }: any) => <button>{children}</button>,
  InputHelper: ({ children }: any) => <p>{children}</p>,
  Tabs: ({ tabs, children, selectedIndex = 0, onChange }: any) => {
    const React = require('react');
    return (
      <div>
        <div role="tablist">
          {tabs.map((tab: string, index: number) => (
            <button role="tab" key={tab} onClick={() => onChange(index)}>
              {tab}
            </button>
          ))}
        </div>
        {React.Children.map(children, (child, index) => (
          <section hidden={selectedIndex !== index}>{child}</section>
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

jest.mock('@/app/components/domain/cards/rapida-credential-card', () => ({
  RapidaCredentialCard: () => <div>Project credentials</div>,
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

const metadata = (key: string, value: string) => ({
  getKey: () => key,
  getValue: () => value,
});

const phoneDeployment = (
  provider: string,
  phoneOptions = [metadata('phone', '+15551234567')],
) => ({
  getId: () => `${provider}-deployment`,
  getAssistantid: () => 'assistant-123',
  getPhoneprovidername: () => provider,
  getPhoneoptionsList: () => phoneOptions,
  getInputaudio: () => undefined,
  getOutputaudio: () => undefined,
  getStatus: () => 'ACTIVE',
});

describe('AssistantPhoneCallDeploymentDialog', () => {
  it('keeps SIP deployment instructions tenant neutral', () => {
    render(
      <AssistantPhoneCallDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        deployment={
          phoneDeployment('sip', [
            metadata('phone', '+15551234567'),
            metadata('rapida.sip_inbound', 'true'),
          ]) as any
        }
      />,
    );

    expect(screen.queryByText('rapida.sip_inbound')).not.toBeInTheDocument();
    expect(document.body.textContent).toContain('<your-phone-host>:5060');
    expect(document.body.textContent).toContain('[voice-ai-trunk]');
    expect(document.body.textContent).not.toMatch(/rapida/i);
  });

  it('keeps Asterisk deployment instructions tenant neutral', () => {
    render(
      <AssistantPhoneCallDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        deployment={phoneDeployment('asterisk') as any}
      />,
    );

    expect(document.body.textContent).toContain('[voice-ai-inbound-ws]');
    expect(document.body.textContent).toContain(
      'wss://voice.client.example/v1/talk/asterisk/ctx/{contextId}',
    );
    expect(document.body.textContent).not.toMatch(/rapida/i);
  });

  it('separates setup, telephony configuration, and audio', async () => {
    const user = userEvent.setup();
    render(
      <AssistantPhoneCallDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        deployment={phoneDeployment('twilio') as any}
      />,
    );

    expect(screen.getAllByRole('tab').map(tab => tab.textContent)).toEqual([
      'Setup',
      'Configuration',
      'Audio',
    ]);
    expect(screen.getByText('Step 1')).toBeVisible();
    expect(screen.getByText('Project credentials')).toBeVisible();

    await user.click(screen.getByRole('tab', { name: 'Configuration' }));
    expect(screen.getByText('Saved deployment configuration')).toBeVisible();
    expect(screen.getByText('Telephony')).toBeVisible();
  });
});

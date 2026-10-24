import React from 'react';
import { render, screen } from '@testing-library/react';
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
  Tabs: ({ tabs, children, selectedIndex = 0 }: any) => (
    <div>
      {tabs.map((tab: string) => (
        <button key={tab}>{tab}</button>
      ))}
      {Array.isArray(children) ? children[selectedIndex] : children}
    </div>
  ),
}));

jest.mock('@/app/components/layout/container/message/notice-block', () => ({
  YellowNoticeBlock: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/domain/pills/provider-model-pill', () => ({
  ProviderPill: ({ provider }: any) => <span>{provider}</span>,
}));

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
});

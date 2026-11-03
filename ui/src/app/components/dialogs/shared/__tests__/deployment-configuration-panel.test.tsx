import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { DeploymentConfigurationPanel } from '../deployment-configuration-panel';

jest.mock('@/app/components/ui/feedback', () => ({
  RecordStatusIndicator: ({ state }: any) => <span>Status {state}</span>,
}));

jest.mock('../deployment-modal-primitives', () => ({
  DeploymentRow: ({ children, label }: any) => (
    <div>
      <span>{label}</span>
      {children}
    </div>
  ),
  DeploymentSectionHeader: ({ label }: any) => <h3>{label}</h3>,
}));

const makeDeployment = (overrides = {}) => ({
  getAssistantid: () => 'assistant-123',
  getStatus: () => 'ACTIVE',
  getMaxsessionduration: () => '300',
  getIdealtimeout: () => '30',
  getIdealtimeoutmessage: () => 'Are you still there?',
  getIdealtimeoutbackoff: () => '2',
  hasGreetinginterruptible: () => true,
  getGreetinginterruptible: () => true,
  hasUnclearinputtimeout: () => true,
  getUnclearinputtimeout: () => 8,
  getUnclearinputmessage: () => 'Could you repeat that?',
  getGreeting: () => 'Hello!',
  getMistake: () => 'Something went wrong.',
  ...overrides,
});

describe('DeploymentConfigurationPanel', () => {
  it('presents the saved deployment behavior in grouped Carbon rows', () => {
    render(
      <DeploymentConfigurationPanel
        deployment={makeDeployment()}
        channelDetails={<div>Channel-specific settings</div>}
      />,
    );

    expect(
      screen.queryByRole('heading', { name: 'General' }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Session settings' }),
    ).toBeVisible();
    expect(
      screen.getByRole('heading', { name: 'Conversation messages' }),
    ).toBeVisible();
    expect(screen.getByText('Channel-specific settings')).toBeVisible();
    expect(screen.getByText('300 seconds')).toBeVisible();
    expect(screen.getByText('Status ACTIVE')).toBeVisible();
    expect(screen.getByText('Interruptible')).toBeVisible();
    expect(screen.getByText('Are you still there?')).toBeVisible();
  });

  it('shows effective editor defaults for absent legacy configuration', () => {
    render(
      <DeploymentConfigurationPanel
        deployment={makeDeployment({
          getStatus: () => 'INACTIVE',
          getMaxsessionduration: () => '0',
          getIdealtimeout: () => '0',
          getIdealtimeoutmessage: () => '',
          getIdealtimeoutbackoff: () => '',
          hasGreetinginterruptible: () => false,
          getGreetinginterruptible: () => false,
          hasUnclearinputtimeout: () => false,
          getUnclearinputtimeout: () => 0,
          getUnclearinputmessage: () => '',
          getGreeting: () => '',
          getMistake: () => '',
        })}
      />,
    );

    expect(screen.getByText('Status INACTIVE')).toBeVisible();
    expect(screen.getByText('300 seconds')).toBeVisible();
    expect(screen.getByText('10 seconds')).toBeVisible();
    expect(screen.getByText('2')).toBeVisible();
    expect(screen.getByText('Interruptible')).toBeVisible();
    expect(screen.getByText('2 seconds')).toBeVisible();
    expect(screen.getAllByText('Not configured').length).toBeGreaterThan(1);
  });

  it('describes an explicit zero idle reminder count as unlimited', () => {
    render(
      <DeploymentConfigurationPanel
        deployment={makeDeployment({
          getIdealtimeoutbackoff: () => '0',
        })}
      />,
    );

    expect(screen.getByText('Unlimited')).toBeVisible();
  });
});

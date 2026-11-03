import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { AssistantDebugDeploymentDialog } from '../assistant-debug-deployment-modal';

const mockGoToEditDebugger = jest.fn();
const mockGoToAssistantPreview = jest.fn();

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToEditDebugger: mockGoToEditDebugger,
    goToAssistantPreview: mockGoToAssistantPreview,
  }),
}));

jest.mock('@/app/components/dialogs/shared', () => ({
  RightSideModal: ({ children, className, resizable, title }: any) => (
    <section
      className={className}
      aria-label={title}
      data-resizable={String(resizable)}
    >
      {children}
    </section>
  ),
}));

jest.mock('@carbon/react', () => ({
  Accordion: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  AccordionItem: ({ children, open, title, ...props }: any) => (
    <section {...props}>
      <button aria-expanded={open}>{title}</button>
      <div>{children}</div>
    </section>
  ),
  ButtonSet: ({ children }: any) => <div>{children}</div>,
  StructuredListBody: ({ children }: any) => <div>{children}</div>,
  StructuredListCell: ({ children }: any) => <div>{children}</div>,
  StructuredListRow: ({ children }: any) => <div>{children}</div>,
  StructuredListWrapper: ({
    children,
    isCondensed: _isCondensed,
    selection: _selection,
    ...props
  }: any) => <div {...props}>{children}</div>,
}));

jest.mock('@carbon/icons-react', () => ({
  Chat: () => <svg data-testid="general-icon" />,
  Edit: () => <svg data-testid="edit-icon" />,
  Launch: () => <svg data-testid="launch-icon" />,
  VoiceActivate: () => <svg data-testid="input-icon" />,
  VolumeUp: () => <svg data-testid="output-icon" />,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  PrimaryButton: ({ children, onClick }: any) => (
    <button onClick={onClick}>{children}</button>
  ),
  SecondaryButton: ({ children, onClick }: any) => (
    <button onClick={onClick}>{children}</button>
  ),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  RecordStatusIndicator: ({ state }: any) => <span>Status {state}</span>,
}));

const option = (key: string, value: string) => ({
  getKey: () => key,
  getValue: () => value,
});

const audio = (provider: string, options: any[]) => ({
  getAudioprovider: () => provider,
  getAudiooptionsList: () => options,
});

const makeDeployment = (overrides = {}) => ({
  getId: () => 'debugger-1',
  getAssistantid: () => 'assistant-1',
  getStatus: () => 'ACTIVE',
  getGreeting: () => 'Hello, how can I help?',
  getMistake: () => 'Sorry, something went wrong.',
  getMaxsessionduration: () => '300',
  getIdealtimeout: () => '30',
  getIdealtimeoutbackoff: () => '2',
  hasGreetinginterruptible: () => true,
  getGreetinginterruptible: () => true,
  hasUnclearinputtimeout: () => true,
  getUnclearinputtimeout: () => 8,
  getInputaudio: () => audio('deepgram', [option('listen.model', 'nova-3')]),
  getOutputaudio: () => audio('cartesia', [option('speak.voice', 'sonic-2')]),
  ...overrides,
});

describe('AssistantDebugDeploymentDialog', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('uses one Carbon detail surface and routes its two actions', async () => {
    const user = userEvent.setup();
    const setModalOpen = jest.fn();

    render(
      <AssistantDebugDeploymentDialog
        modalOpen
        setModalOpen={setModalOpen}
        deployment={makeDeployment() as any}
      />,
    );

    expect(screen.queryByRole('tablist')).not.toBeInTheDocument();
    expect(screen.getByLabelText('Debugger')).toHaveAttribute(
      'data-resizable',
      'false',
    );
    expect(screen.getByLabelText('Debugger configuration')).toHaveClass(
      'debugger-configuration-accordion',
    );
    expect(
      screen.getByRole('button', { name: /General experience/ }),
    ).toBeVisible();
    expect(screen.getByRole('button', { name: /Voice input/ })).toBeVisible();
    expect(screen.getByRole('button', { name: /Voice output/ })).toBeVisible();
    expect(screen.getByText('Status ACTIVE')).toBeVisible();
    expect(screen.getByText('Hello, how can I help?')).toBeVisible();
    expect(screen.getByText('300 seconds')).toBeVisible();
    expect(screen.getByText('30 seconds · 2 reminders')).toBeVisible();
    expect(screen.getByText('Deepgram · nova-3')).toBeVisible();
    expect(screen.getByText('Cartesia · sonic-2')).toBeVisible();

    await user.click(
      screen.getByRole('button', { name: 'Edit configuration' }),
    );
    expect(setModalOpen).toHaveBeenCalledWith(false);
    expect(mockGoToEditDebugger).toHaveBeenCalledWith(
      'assistant-1',
      'debugger-1',
    );

    await user.click(screen.getByRole('button', { name: 'Open debugger' }));
    expect(mockGoToAssistantPreview).toHaveBeenCalledWith('assistant-1');
  });

  it('uses effective debugger defaults for fields absent from legacy deployments', () => {
    render(
      <AssistantDebugDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        deployment={
          makeDeployment({
            getGreeting: () => '',
            getMistake: () => '',
            getMaxsessionduration: () => '0',
            getIdealtimeout: () => '0',
            getIdealtimeoutbackoff: () => '',
            hasGreetinginterruptible: () => false,
            getGreetinginterruptible: () => false,
            hasUnclearinputtimeout: () => false,
            getUnclearinputtimeout: () => 0,
            getInputaudio: () => undefined,
            getOutputaudio: () => undefined,
          }) as any
        }
      />,
    );

    expect(screen.getAllByText('—')).toHaveLength(2);
    expect(screen.getByText('300 seconds')).toBeVisible();
    expect(screen.getByText('Interruptible')).toBeVisible();
    expect(screen.getByText('10 seconds · 2 reminders')).toBeVisible();
    expect(screen.getByText('2 seconds')).toBeVisible();
    expect(screen.getAllByText('Off')).toHaveLength(4);
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument();
    expect(screen.queryByText('Setup')).not.toBeInTheDocument();
  });

  it('describes an explicit zero idle prompt count as unlimited', () => {
    render(
      <AssistantDebugDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        deployment={
          makeDeployment({
            getIdealtimeoutbackoff: () => '0',
          }) as any
        }
      />,
    );

    expect(screen.getByText('30 seconds · Unlimited')).toBeVisible();
  });
});

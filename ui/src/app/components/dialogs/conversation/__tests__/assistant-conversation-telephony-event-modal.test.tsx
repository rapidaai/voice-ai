import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import {
  AssistantConversationTelephonyEventDialog,
  AssistantConversationTelephonyEventPanel,
} from '../assistant-conversation-telephony-event-modal';

jest.mock('@/app/components/dialogs/shared', () => ({
  ModalBody: ({ children }: any) => <main>{children}</main>,
  RightSideModal: ({ modalOpen, children }: any) =>
    modalOpen ? <div>{children}</div> : null,
}));

jest.mock('@carbon/react', () => ({
  Button: ({ children, kind, ...props }: any) => (
    <button data-carbon-button-kind={kind} {...props}>
      {children}
    </button>
  ),
  Table: ({ children }: any) => <table>{children}</table>,
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableCell: ({ children, colSpan }: any) => (
    <td colSpan={colSpan}>{children}</td>
  ),
  TableHead: ({ children }: any) => <thead>{children}</thead>,
  TableHeader: ({ children }: any) => <th>{children}</th>,
  TableRow: ({ children }: any) => <tr>{children}</tr>,
}));

jest.mock('@carbon/icons-react', () => ({
  ChevronDown: () => <svg data-testid="chevron-down" />,
  ChevronRight: () => <svg data-testid="chevron-right" />,
}));

jest.mock('@/app/components/ui/editor/code-highlighting', () => ({
  CodeHighlighting: ({ language, code }: any) => (
    <div data-testid="payload" data-language={language}>
      {code}
    </div>
  ),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  EmptyState: ({ title, subtitle }: any) => (
    <section>
      <h3>{title}</h3>
      <p>{subtitle}</p>
    </section>
  ),
}));

jest.mock('@/utils/date', () => ({
  toHumanReadableDateTime: jest.fn(() => 'formatted-date'),
}));

type MockEvent = {
  getId: () => string;
  getProvider: () => string;
  getEventtype: () => string;
  getCreateddate: () => any;
  getPayload: () => { toJavaScript: () => unknown };
};

const makeEvent = (overrides: Partial<MockEvent> = {}): MockEvent => ({
  getId: () => 'evt_1',
  getProvider: () => 'twilio',
  getEventtype: () => 'ringing',
  getCreateddate: () => ({}),
  getPayload: () => ({ toJavaScript: () => ({ ok: true }) }),
  ...overrides,
});

describe('AssistantConversationTelephonyEventDialog', () => {
  it('renders JSON payload using json language when row is expanded', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <AssistantConversationTelephonyEventDialog
        modalOpen
        setModalOpen={jest.fn()}
        events={[makeEvent() as any]}
      />,
    );

    await user.click(screen.getByText('evt_1').closest('button')!);

    expect(screen.getByText('evt_1').closest('button')).toHaveAttribute(
      'data-carbon-button-kind',
      'ghost',
    );
    expect(screen.getByTestId('payload')).toHaveAttribute(
      'data-language',
      'json',
    );

    await user.click(screen.getByText('evt_1').closest('button')!);
    expect(screen.queryByTestId('payload')).not.toBeInTheDocument();
  });

  it('shows fallback created date when event created date is missing', () => {
    render(
      <AssistantConversationTelephonyEventDialog
        modalOpen
        setModalOpen={jest.fn()}
        events={[makeEvent({ getCreateddate: () => undefined }) as any]}
      />,
    );

    expect(screen.getByText('N/A')).toBeInTheDocument();
  });

  it('renders all event rows without filtering controls', () => {
    render(
      <AssistantConversationTelephonyEventDialog
        modalOpen
        setModalOpen={jest.fn()}
        events={[
          makeEvent({
            getId: () => 'evt_session',
            getEventtype: () => 'session',
          }) as any,
          makeEvent({
            getId: () => 'evt_llm',
            getEventtype: () => 'llm',
          }) as any,
        ]}
      />,
    );

    expect(screen.getByText('evt_session')).toBeInTheDocument();
    expect(screen.getByText('evt_llm')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Apply Session Filter' }),
    ).not.toBeInTheDocument();
  });

  it('renders an empty state when no events are available', () => {
    render(<AssistantConversationTelephonyEventPanel events={[]} />);

    expect(screen.getByText('No telephony events')).toBeInTheDocument();
    expect(screen.getByText('0')).toBeInTheDocument();
    expect(screen.getAllByText('N/A')).toHaveLength(2);
  });
});

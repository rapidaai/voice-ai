import React from 'react';
import '@testing-library/jest-dom';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { getAssistantConversationDetail } from '@/clients/assistant.client';
import { typographyClassNames } from '@/app/components/ui/primitives/text';
import { ConversationDetailPage } from './conversation-detail';

const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

jest.mock('react-router-dom', () => ({
  useParams: () => ({ assistantId: 'assistant-1', sessionId: 'session-1' }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    token: 'token-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/stores/assistant/assistant-chat.store', () => {
  const React = require('react');
  return {
    AssistantChatContext: React.createContext({}),
    useAssistantChat: () => ({}),
  };
});

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({ goToAssistantSessionList: jest.fn() }),
}));

jest.mock('@/clients/assistant.client', () => ({
  getAssistantConversationDetail: jest.fn(),
}));

jest.mock(
  '@/app/pages/assistant/view/conversations/conversation-messages',
  () => ({ ConversationMessages: () => <div>Message content</div> }),
);

jest.mock('@/app/components/layout/blocks/page-header-block', () => ({
  PageHeaderBlock: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

jest.mock('@/app/components/ui/feedback', () => {
  const actual = jest.requireActual('@/app/components/ui/feedback');
  return {
    ...actual,
    CarbonStatusIndicator: () => <span>Status</span>,
    EmptyState: ({ title }: { title: string }) => <div>{title}</div>,
    PageLoader: () => <div>Loading</div>,
  };
});

const conversation = {
  getArgumentsList: () => [],
  getAssistantid: () => 'assistant-1',
  getContextsList: () => [],
  getId: () => 'session-1',
  getMetadataList: () => [],
  getMetricsList: () => [],
  getOptionsList: () => [],
};

describe('ConversationDetailPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
  });

  it('renders session content in an unpadded Carbon tab panel', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (window.matchMedia as jest.Mock).mockReturnValue({
      matches: true,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
    });
    (getAssistantConversationDetail as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => conversation,
    });

    render(<ConversationDetailPage />);

    const messageContent = await screen.findByText('Message content');
    expect(
      screen.getByRole('tablist', { name: 'Session detail tabs' }),
    ).toBeVisible();
    const tabList = screen.getByRole('tablist', {
      name: 'Session detail tabs',
    });
    expect(tabList.parentElement).toHaveClass(
      'w-full',
      'border-y',
      'border-border-subtle',
    );
    expect(tabList.parentElement).not.toHaveClass('cds--tabs--full-width');
    expect(messageContent.closest('[role="tabpanel"]')).toBe(
      screen.getByRole('tabpanel'),
    );
    expect(screen.getByRole('tabpanel')).toHaveClass(
      '!block',
      '!w-full',
      '!overflow-auto',
      '!p-0',
    );

    await user.click(screen.getByRole('tab', { name: 'Context' }));

    const emptyHeading = screen.getByRole('heading', {
      name: 'No context available',
    });
    expect(emptyHeading).toHaveClass(
      ...typographyClassNames['heading-03'].split(' '),
    );
    expect(emptyHeading.closest('[role="tabpanel"]')).toBe(
      screen.getByRole('tabpanel'),
    );
    expect(
      screen.getByText('No knowledge context was retrieved for this session.'),
    ).toHaveClass(...typographyClassNames['body-01'].split(' '));

    for (const state of [
      {
        tab: 'Arguments',
        title: 'No arguments available',
        description:
          'This session does not contain arguments, options, or metadata.',
      },
      {
        tab: 'Analysis',
        title: 'No analysis available',
        description: 'This session does not contain analysis data.',
      },
      {
        tab: 'Metrics',
        title: 'No metrics available',
        description: 'This session does not contain metrics.',
      },
    ]) {
      await user.click(screen.getByRole('tab', { name: state.tab }));
      expect(screen.getByRole('heading', { name: state.title })).toBeVisible();
      expect(screen.getByText(state.description)).toBeVisible();
    }
  });

  it('shows a recoverable Carbon error when the session cannot load', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    (getAssistantConversationDetail as jest.Mock)
      .mockResolvedValueOnce({
        getSuccess: () => false,
        getData: () => null,
      })
      .mockResolvedValueOnce({
        getSuccess: () => true,
        getData: () => conversation,
      });

    render(<ConversationDetailPage />);

    const errorHeading = await screen.findByRole('heading', {
      name: 'Session unavailable',
    });
    expect(errorHeading).toHaveClass(
      ...typographyClassNames['heading-04'].split(' '),
    );
    expect(screen.getByText('503')).toHaveClass(
      ...typographyClassNames['heading-06'].split(' '),
    );
    expect(
      screen.getByText("We couldn't load this session. Try again in a moment."),
    ).toHaveClass(...typographyClassNames['body-01'].split(' '));

    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('Message content')).toBeVisible();
    expect(getAssistantConversationDetail).toHaveBeenCalledTimes(2);
  });

  it('keeps argument and metric columns aligned when values are long', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const populatedConversation = {
      ...conversation,
      getArgumentsList: () => [
        {
          getName: () => 'customer_identifier',
          getValue: () => 'customer-1234567890-abcdefghijklmnopqrstuvwxyz',
        },
      ],
      getMetricsList: () => [
        {
          getName: () => 'conversation.duration_ms',
          getValue: () => '12345',
          getDescription: () =>
            'Total duration of the assistant conversation in milliseconds.',
        },
      ],
    };
    (getAssistantConversationDetail as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => populatedConversation,
    });

    render(<ConversationDetailPage />);
    await screen.findByText('Message content');

    await user.click(screen.getByRole('tab', { name: 'Arguments' }));
    const argumentsTable = screen.getByRole('table', {
      name: 'Session arguments',
    });
    expect(argumentsTable).toHaveClass(
      'cds--data-table--lg',
      'w-full',
      'table-fixed',
    );
    const argumentHeaders = screen.getAllByRole('columnheader');
    expect(argumentHeaders).toHaveLength(3);
    expect(argumentHeaders[0]).toHaveClass('w-1/5');
    expect(argumentHeaders[1]).toHaveClass('w-[30%]');
    expect(argumentHeaders[2]).toHaveClass('w-1/2');

    await user.click(screen.getByRole('tab', { name: 'Metrics' }));
    const metricsTable = screen.getByRole('table', {
      name: 'Session metrics',
    });
    expect(metricsTable).toHaveClass(
      'cds--data-table--lg',
      'w-full',
      'table-fixed',
    );
    const metricHeaders = screen.getAllByRole('columnheader');
    expect(metricHeaders).toHaveLength(3);
    expect(metricHeaders[0]).toHaveClass('w-[30%]');
    expect(metricHeaders[1]).toHaveClass('w-1/5');
    expect(metricHeaders[2]).toHaveClass('w-1/2');
  });
});

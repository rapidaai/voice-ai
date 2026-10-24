import React from 'react';
import '@testing-library/jest-dom';
import { render, screen } from '@testing-library/react';
import { typographyClassNames } from '@/app/components/ui/primitives/text';
import { AssistantChatContext } from '@/stores/assistant/assistant-chat.store';
import { ConversationMessages } from './conversation-messages';

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/app/components/ui/feedback', () => ({
  CarbonStatusIndicator: () => <span>Status</span>,
  SectionLoader: () => <div>Loading messages</div>,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  GhostButton: ({ iconDescription }: { iconDescription: string }) => (
    <button type="button">{iconDescription}</button>
  ),
}));

jest.mock('@/app/components/ui/composites/audio-player', () => ({
  AudioPlayer: () => <div>Audio player</div>,
}));

describe('ConversationMessages', () => {
  it('uses the session empty state when no messages were captured', async () => {
    const onGetConversationMessages = jest.fn(
      (_assistantId, _conversationId, _projectId, _token, _userId, onDone) =>
        onDone(),
    );
    const value = {
      conversations: [],
      onGetConversationMessages,
      onChangeConversationMessages: jest.fn(),
    };
    const conversation = {
      getMetricsList: () => [],
      getRecordingsList: () => [],
    };

    render(
      <AssistantChatContext.Provider value={value as any}>
        <ConversationMessages
          conversation={conversation as any}
          assistantId="assistant-1"
          conversationId="session-1"
        />
      </AssistantChatContext.Provider>,
    );

    const heading = await screen.findByRole('heading', {
      name: 'No messages available',
    });
    expect(heading).toHaveClass(
      ...typographyClassNames['heading-03'].split(' '),
    );
    expect(
      screen.getByText('This session does not contain any messages.'),
    ).toHaveClass(...typographyClassNames['body-01'].split(' '));
    expect(heading.closest('section')).toHaveClass('min-h-0', 'flex-1');
    expect(onGetConversationMessages).toHaveBeenCalledTimes(1);
  });
});

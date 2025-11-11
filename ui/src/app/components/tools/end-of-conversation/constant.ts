import { Metadata } from '@rapidaai/react';

export const GetEndOfConversationDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  return [];
};

export const ValidateEndOfConversationDefaultOptions = (
  options: Metadata[],
): boolean => {
  return true;
};

export const EndOfConverstaionToolDefintion = {
  name: 'end_conversation',
  description:
    'Gracefully ends the current conversation when the user indicates that they are done, expresses gratitude, or the assistant determines the session is complete.',
  parameters: JSON.stringify(
    {
      properties: {
        reason: {
          description:
            "Brief reason for ending the conversation, such as 'user said goodbye', 'conversation completed', or 'timeout'.",
          type: 'string',
        },
      },
      required: ['reason'],
      type: 'object',
    },
    null,
    2,
  ),
};

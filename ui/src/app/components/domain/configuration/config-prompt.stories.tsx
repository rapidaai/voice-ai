import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { ConfigPrompt } from './config-prompt';

const meta = {
  title: 'Domain/Configuration/Config Prompt',
  component: ConfigPrompt as any,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Prompt editor wrapper for instruction messages, template variables, and runtime reserved variables.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const RuntimeHints: Story = {
  render: function Render() {
    const [prompt, setPrompt] = useState({
      prompt: [
        {
          role: 'system',
          content:
            'You are {{assistant.name}} helping {{customer_name}} on {{client.phone}}.',
        },
        {
          role: 'user',
          content: 'Customer asks about {{topic}}.',
        },
      ],
      variables: [
        { name: 'assistant.name', type: 'text', defaultvalue: '' },
        { name: 'customer_name', type: 'text', defaultvalue: 'Priyanka' },
        { name: 'client.phone', type: 'text', defaultvalue: '' },
        { name: 'topic', type: 'paragraph', defaultvalue: 'Billing' },
      ],
    });

    return (
      <ConfigPrompt
        existingPrompt={prompt}
        showRuntimeReplacementHint
        enableReservedVariableSuggestions
        onChange={setPrompt}
      />
    );
  },
};

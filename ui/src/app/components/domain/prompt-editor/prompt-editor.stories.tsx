import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import PromptEditor from './prompt-editor';

const meta = {
  title: 'Domain/Prompt Editor',
  component: PromptEditor,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Monaco-backed prompt editor with Twig syntax and reserved runtime variable suggestions.',
      },
    },
  },
} satisfies Meta<typeof PromptEditor>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ReservedVariables: Story = {
  render: function Render() {
    const [value, setValue] = useState(
      'You are {{assistant.name}} helping {{client.phone}}.',
    );

    return (
      <div className="max-w-4xl">
        <PromptEditor
          className="border border-gray-200 dark:border-gray-800"
          enableReservedVariableSuggestions
          height="280px"
          onChange={setValue}
          placeholder="Write your prompt. Type {{ to insert a runtime variable."
          value={value}
        />
      </div>
    );
  },
};

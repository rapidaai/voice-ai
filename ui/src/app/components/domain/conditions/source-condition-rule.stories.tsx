import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { SourceConditionRule } from './source-condition-rule';

const conditionOptions = [
  { label: 'Equals', value: '=' },
  { label: 'Not equals', value: '!=' },
];

const sourceOptions = [
  { label: 'Phone', value: 'phone' },
  { label: 'Web', value: 'web' },
];

const keyOptions = [
  { label: 'Source', value: 'source' },
  { label: 'Mode', value: 'conversation_mode' },
];

const valueOptionsByKey = {
  source: sourceOptions,
  conversation_mode: [
    { label: 'Voice', value: 'voice' },
    { label: 'Text', value: 'text' },
  ],
};

const meta = {
  title: 'Domain/Conditions/Source Condition Rule',
  component: SourceConditionRule as any,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Carbon table rule editor for conditional tool and assistant behavior.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Default: Story = {
  render: function Render() {
    const [conditions, setConditions] = useState([
      { key: 'source', condition: '=', value: 'phone' },
    ]);

    return (
      <SourceConditionRule
        conditions={conditions}
        onChangeConditions={setConditions}
        conditionOptions={conditionOptions}
        sourceOptions={sourceOptions}
        keyOptions={keyOptions}
        valueOptionsByKey={valueOptionsByKey}
      />
    );
  },
};

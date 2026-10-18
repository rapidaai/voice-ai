import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { useState } from 'react';
import { CodeEditor } from './code-editor';
import { CodeHighlighting } from './code-highlighting';
import { JsonEditor } from './json-editor';
import { MarkdownViewer } from './markdown-viewer';

const meta = {
  title: 'UI/Editor',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Editor surfaces for JSON configuration, code previews, and rendered markdown.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Editors: Story = {
  render: function Render() {
    const [json, setJson] = useState('{\n  "enabled": true\n}');

    return (
      <Stack gap={7} className="max-w-4xl">
        <CodeEditor
          labelText="Tool parameters"
          helperText="JSON passed to the runtime."
          placeholder="Enter JSON"
          value={json}
          onChange={setJson}
        />
        <JsonEditor
          value={json}
          onChange={setJson}
          placeholder="Enter JSON"
          height="160px"
        />
        <CodeHighlighting
          language="json"
          code={'{\n  "status": "ready"\n}'}
          className="min-h-40"
        />
        <MarkdownViewer
          text={'# Release notes\n\n- Carbon aligned editor UI'}
        />
      </Stack>
    );
  },
};

import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Metadata } from '@rapidaai/react';
import { useState } from 'react';
import {
  CloudStorageProvider,
  defaultStorageFiles,
  StorageFileSelector,
} from './storage';

const meta = {
  title: 'Domain/Providers/Storage',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Storage provider selector and recording file selection controls used by assistant storage configuration.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const StorageSetup: Story = {
  render: function Render() {
    const [provider, setProvider] = useState('');
    const [parameters, setParameters] = useState<Metadata[]>([]);
    const [selectedFiles, setSelectedFiles] =
      useState<string[]>(defaultStorageFiles);

    return (
      <Stack gap={7} className="max-w-4xl">
        <StorageFileSelector
          group="Recording"
          selectedFiles={selectedFiles}
          onChange={setSelectedFiles}
        />
        <CloudStorageProvider
          provider={provider}
          onChangeProvider={setProvider}
          parameters={parameters}
          onChangeParameter={setParameters}
        />
      </Stack>
    );
  },
};

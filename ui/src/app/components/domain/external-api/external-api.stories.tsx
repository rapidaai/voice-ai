import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { useState } from 'react';
import { ApiHeader, ApiStringHeader } from './api-header';
import { ApiParameter } from './api-parameter';

const meta = {
  title: 'Domain/External API',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Carbon table editors for external API headers and key value parameter mappings.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Editors: Story = {
  render: function Render() {
    const [headers, setHeaders] = useState([
      { key: 'Authorization', value: 'Bearer token' },
      { key: 'X-Trace', value: 'trace-id' },
    ]);
    const [headerJson, setHeaderJson] = useState(
      '{"Content-Type":"application/json"}',
    );
    const [parameters, setParameters] = useState([
      { key: 'body.customerId', value: 'customer.id' },
    ]);

    return (
      <Stack gap={6}>
        <ApiHeader headers={headers} setHeaders={setHeaders} />
        <ApiStringHeader
          headerValue={headerJson}
          setHeaderValue={setHeaderJson}
        />
        <ApiParameter
          initialValues={parameters}
          setParameterValue={setParameters}
          actionButtonLabel="Add parameter"
        />
      </Stack>
    );
  },
};

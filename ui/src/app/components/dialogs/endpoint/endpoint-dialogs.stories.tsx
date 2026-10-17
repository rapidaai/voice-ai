import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  ConfigureEndpointPromptDialog,
  EndpointArguments,
  EndpointInstructionDialog,
  EndpointMetadatas,
  EndpointMetrics,
  EndpointOptions,
  EndpointTraceModal,
} from '.';

const makeArgument = (name: string, value: string) =>
  ({
    getName: () => name,
    getValue: () => value,
  }) as never;

const makeMetadata = (key: string, value: string) =>
  ({
    getKey: () => key,
    getValue: () => value,
  }) as never;

const makeMetric = (name: string, description: string, value: string) =>
  ({
    getDescription: () => description,
    getName: () => name,
    getValue: () => value,
  }) as never;

const trace = {
  getArgumentsList: () => [
    makeArgument('customer_id', 'cust_123'),
    makeArgument('transcript', 'Customer asked about plan limits.'),
  ],
  getCreateddate: () => null,
  getEndpointprovidermodelid: () => 'model-1',
  getId: () => 'trace-1',
  getMetadataList: () => [
    makeMetadata('region', 'us-east'),
    makeMetadata('channel', 'voice'),
  ],
  getMetricsList: () => [
    makeMetric('confidence', 'Classifier confidence', '0.86'),
    makeMetric('latency', 'Total request latency', '245'),
  ],
  getOptionsList: () => [
    makeMetadata('temperature', '0.7'),
    makeMetadata('response_format', '{"type":"json_schema"}'),
  ],
  getSource: () => 'api',
  getStatus: () => 'success',
  getTimetaken: () => 245000000,
};

const meta = {
  title: 'Dialogs/Endpoint',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Endpoint dialogs and trace panels for integration guidance, template selection, and runtime inspection.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const ConfigurePromptTemplate: Story = {
  render: () => (
    <ConfigureEndpointPromptDialog
      modalOpen
      setModalOpen={() => undefined}
      onSelectTemplate={() => undefined}
    />
  ),
};

export const TraceDetails: Story = {
  render: () => (
    <EndpointTraceModal
      modalOpen
      setModalOpen={() => undefined}
      currentTrace={trace as never}
    />
  ),
};

export const InstructionPlaceholder: Story = {
  render: () => (
    <EndpointInstructionDialog
      modalOpen
      setModalOpen={() => undefined}
      currentEndpoint={null}
    />
  ),
};

export const TracePanels: Story = {
  render: () => (
    <div className="grid gap-6">
      <EndpointMetrics metrics={trace.getMetricsList()} />
      <EndpointMetadatas metadata={trace.getMetadataList()} />
      <EndpointOptions options={trace.getOptionsList()} />
      <EndpointArguments args={trace.getArgumentsList()} />
    </div>
  ),
};

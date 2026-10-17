import type { Meta, StoryObj } from '@storybook/react-vite';
import { Tag } from '@carbon/react';
import {
  ErrorWrapper,
  InfoWrapper,
  PlainWrapper,
  SuccessWrapper,
  WarnWrapper,
} from './alert-wrapper';

const meta = {
  title: 'Layout/Wrapper/AlertWrapper',
  component: InfoWrapper,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Status wrapper surfaces for compact inline messages and execution rows.',
      },
    },
  },
} satisfies Meta<typeof InfoWrapper>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Variants: Story = {
  render: () => (
    <div className="grid max-w-3xl gap-3 bg-background p-4">
      <InfoWrapper>
        <Tag type="blue" size="sm">
          Info
        </Tag>
        <span>Configuration is ready for review.</span>
      </InfoWrapper>
      <SuccessWrapper>
        <Tag type="green" size="sm">
          Success
        </Tag>
        <span>Deployment completed successfully.</span>
      </SuccessWrapper>
      <WarnWrapper>
        <Tag type="warm-gray" size="sm">
          Warning
        </Tag>
        <span>Credentials expire soon.</span>
      </WarnWrapper>
      <ErrorWrapper>
        <Tag type="red" size="sm">
          Error
        </Tag>
        <span>Provider connection failed.</span>
      </ErrorWrapper>
      <PlainWrapper>
        <Tag type="gray" size="sm">
          Plain
        </Tag>
        <span>Execution message without status emphasis.</span>
      </PlainWrapper>
    </div>
  ),
};

import type { Meta, StoryObj } from '@storybook/react-vite';
import { ConversationDirectionIndicator } from './conversation-direction';
import { HttpStatusSpanIndicator } from './http-status';
import { OrganizationRoleIndicator } from './organization-role';
import { RoleIndicator } from './role';
import { SourceIndicator } from './source';
import { StatusIndicator } from './status';
import { VersionIndicator } from './version';

const meta = {
  title: 'Domain/Indicators',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Domain indicators for status, roles, channels, HTTP responses, and deployment versions.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Statuses: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <StatusIndicator state="INVITED" />
      <StatusIndicator state="ACTIVE" />
      <StatusIndicator state="IN_PROGRESS" />
      <StatusIndicator state="SUCCESS" />
      <StatusIndicator state="FAILED" />
      <StatusIndicator state="UNKNOWN" />
    </div>
  ),
};

export const Roles: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <RoleIndicator role="SUPER_ADMIN" />
      <RoleIndicator role="admin" />
      <RoleIndicator role="writer" />
      <RoleIndicator role="reader" />
      <OrganizationRoleIndicator role="OWNER" />
      <OrganizationRoleIndicator role="member" />
    </div>
  ),
};

export const Sources: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <ConversationDirectionIndicator direction="inbound" />
      <ConversationDirectionIndicator direction="outbound" />
      <SourceIndicator source="phone-call" />
      <SourceIndicator source="react-sdk" />
      <SourceIndicator source="python-sdk" />
      <SourceIndicator source="unknown-source" />
      <SourceIndicator source="sdk" withLabel={false} />
    </div>
  ),
};

export const HttpStatuses: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <HttpStatusSpanIndicator status={204} />
      <HttpStatusSpanIndicator status={302} />
      <HttpStatusSpanIndicator status={404} />
      <HttpStatusSpanIndicator status={503} />
      <HttpStatusSpanIndicator status={102} />
    </div>
  ),
};

export const Version: Story = {
  render: () => <VersionIndicator id="deployment-1" />,
};

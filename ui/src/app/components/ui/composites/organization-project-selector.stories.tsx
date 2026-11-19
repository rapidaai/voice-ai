import { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@carbon/react';
import { Text } from '@/app/components/ui/primitives';
import {
  OrganizationProjectSelector,
  type OrganizationProjectSelectorOrganization,
  type OrganizationProjectSelectorProject,
} from './organization-project-selector';

const organizations: OrganizationProjectSelectorOrganization[] = [
  {
    id: 'acme',
    name: 'Acme Corporation',
    projects: [
      { id: 'customer-support', name: 'Customer Support' },
      { id: 'sales-automation', name: 'Sales Automation' },
      { id: 'voice-platform', name: 'Voice Platform' },
    ],
  },
  {
    id: 'northwind',
    name: 'Northwind Group',
    projects: [
      { id: 'internal-tools', name: 'Internal Tools' },
      { id: 'order-assistant', name: 'Order Assistant' },
    ],
  },
  {
    id: 'contoso',
    name: 'Contoso Research',
    projects: [],
  },
];

function ReviewSurface({
  items = organizations,
  isLoading = false,
}: {
  items?: OrganizationProjectSelectorOrganization[];
  isLoading?: boolean;
}) {
  const [organization, setOrganization] = useState(items[0]);
  const [project, setProject] = useState<
    OrganizationProjectSelectorProject | undefined
  >(items[0]?.projects[0]);

  return (
    <Stack gap={6} className="min-h-96 w-full">
      <div className="flex h-12 justify-end border-b border-border-subtle bg-layer">
        <OrganizationProjectSelector
          organizations={items}
          selectedOrganizationId={organization?.id}
          selectedProjectId={project?.id}
          isLoading={isLoading}
          onChange={(
            selectedOrganization: OrganizationProjectSelectorOrganization,
            selectedProject?: OrganizationProjectSelectorProject,
          ) => {
            setOrganization(selectedOrganization);
            setProject(selectedProject);
          }}
        />
      </div>
      <div>
        <Text as="h2" variant="heading-compact-01">
          Selected context
        </Text>
        <Text as="p" variant="body-compact-01" className="mt-2 text-muted">
          {organization?.name || 'No organization'}
          {project ? ` / ${project.name}` : ''}
        </Text>
      </div>
    </Stack>
  );
}

const meta = {
  title: 'UI/Composites/Organization project selector',
  component: OrganizationProjectSelector,
  tags: ['autodocs'],
  args: {
    organizations,
    onChange: () => {},
  },
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          'A Carbon-composed context selector that groups projects under their owning organization. It owns presentation and interaction only.',
      },
    },
  },
} satisfies Meta<typeof OrganizationProjectSelector>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => <ReviewSurface />,
};

export const LongNames: Story = {
  render: () => (
    <ReviewSurface
      items={[
        {
          id: 'enterprise',
          name: 'International Customer Experience Organization',
          projects: [
            {
              id: 'project',
              name: 'Enterprise Conversational Automation Platform',
            },
          ],
        },
        ...organizations,
      ]}
    />
  ),
};

export const OrganizationWithoutProjects: Story = {
  render: () => <ReviewSurface items={[organizations[2]]} />,
};

export const Empty: Story = {
  render: () => <ReviewSurface items={[]} />,
};

export const Loading: Story = {
  render: () => <ReviewSurface isLoading />,
};

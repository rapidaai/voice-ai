import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack, Tag } from '@carbon/react';
import { AssistantTag } from './assistant-tags';
import { EndpointTag } from './endpoint-tags';
import { KnowledgeTags } from './knowledge-tags';

interface TagCatalogProps {
  title: string;
  tags: string[];
}

function TagCatalog({ title, tags }: TagCatalogProps) {
  return (
    <section className="space-y-3">
      <div>
        <h3 className="text-base font-semibold text-gray-900 dark:text-gray-100">
          {title}
        </h3>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {tags.length} suggested tags
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        {tags.map(tag => (
          <Tag key={tag} type="gray">
            {tag}
          </Tag>
        ))}
      </div>
    </section>
  );
}

const meta = {
  title: 'Domain/Tags',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Curated tag catalogs used by assistant, endpoint, and knowledge selectors.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Catalogs: Story = {
  render: () => (
    <Stack gap={7}>
      <TagCatalog title="Assistant" tags={AssistantTag} />
      <TagCatalog title="Endpoint" tags={EndpointTag} />
      <TagCatalog title="Knowledge" tags={KnowledgeTags} />
    </Stack>
  ),
};

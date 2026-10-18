import type { Meta, StoryObj } from '@storybook/react-vite';
import { Button, Stack } from '@carbon/react';
import { useState } from 'react';
import { DateFilter } from './date-filter';
import { SearchIconInput } from './icon-input';
import { CardOptionMenu, OptionMenu, OptionMenuItem } from './menu';
import { QuerySearch, type QuerySearchField } from './query-search';
import { TabForm } from './tab-form';
import { TableToolbarFilter } from './table-toolbar-filter';
import { TagInput } from './tag-input';

const meta = {
  title: 'UI/Composites',
  component: SearchIconInput,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Composed UI controls built from Carbon primitives and shared application wrappers.',
      },
    },
  },
} satisfies Meta<typeof SearchIconInput>;

export default meta;

type Story = StoryObj<typeof meta>;

const queryFields: QuerySearchField[] = [
  {
    category: 'runtime',
    queryKey: 'status',
    text: 'Status',
    type: 'multi-select',
    items: [
      { id: 'active', text: 'Active' },
      { id: 'paused', text: 'Paused' },
      { id: 'failed', text: 'Failed' },
    ],
  },
  {
    category: 'runtime',
    logicLabel: 'after',
    queryKey: 'created_at',
    text: 'Created',
    type: 'date',
  },
  {
    category: 'metadata',
    queryKey: 'owner',
    text: 'Owner',
    type: 'string',
  },
];

export const SearchAndMenus: Story = {
  render: () => (
    <Stack gap={5} className="max-w-2xl">
      <form onSubmit={event => event.preventDefault()}>
        <SearchIconInput placeholder="Search assistants" />
      </form>
      <div className="flex items-center gap-3">
        <OptionMenu
          options={[
            { option: <span>Open</span>, onActionClick: () => {} },
            {
              option: <OptionMenuItem type="danger">Delete</OptionMenuItem>,
              onActionClick: () => {},
            },
          ]}
        />
        <CardOptionMenu
          options={[
            { option: <span>Rename</span>, onActionClick: () => {} },
            { option: <span>Archive</span>, onActionClick: () => {} },
          ]}
        />
      </div>
    </Stack>
  ),
};

export const Filters: Story = {
  render: function Render() {
    const [filters, setFilters] = useState(new Set(['llm']));

    return (
      <div className="flex items-center gap-3">
        <DateFilter onApply={() => {}} onReset={() => {}} />
        <TableToolbarFilter
          filters={[
            { id: 'llm', label: 'LLM' },
            { id: 'tool', label: 'Tool' },
            { id: 'webhook', label: 'Webhook' },
          ]}
          activeFilters={filters}
          onApplyFilter={setFilters}
          onResetFilter={() => setFilters(new Set())}
        />
      </div>
    );
  },
};

export const Tags: Story = {
  render: function Render() {
    const [tags, setTags] = useState(['production']);

    return (
      <div className="max-w-lg">
        <TagInput
          tags={tags}
          allTags={['production', 'billing', 'support', 'routing']}
          addTag={tag => setTags(current => [...new Set([...current, tag])])}
          removeTag={tag =>
            setTags(current => current.filter(currentTag => currentTag !== tag))
          }
        />
      </div>
    );
  },
};

export const Steps: Story = {
  render: function Render() {
    const [activeTab, setActiveTab] = useState('profile');

    return (
      <div className="h-[520px] border border-gray-200 dark:border-gray-800">
        <TabForm
          activeTab={activeTab}
          onChangeActiveTab={setActiveTab}
          formHeading="Configure a reusable assistant workflow."
          form={[
            {
              code: 'profile',
              name: 'Profile',
              description: 'Name and ownership details.',
              body: <div className="p-6">Profile form content</div>,
              actions: [<Button key="profile-action">Continue</Button>],
            },
            {
              code: 'routing',
              name: 'Routing',
              description: 'Routing and fallback behavior.',
              body: <div className="p-6">Routing form content</div>,
              actions: [<Button key="routing-action">Save</Button>],
            },
          ]}
        />
      </div>
    );
  },
};

export const QueryBuilder: Story = {
  render: function Render() {
    const [value, setValue] = useState('status:active');

    return (
      <div className="max-w-3xl">
        <QuerySearch
          fields={queryFields}
          tabs={[
            { id: 'all', text: 'All' },
            { id: 'runtime', text: 'Runtime' },
            { id: 'metadata', text: 'Metadata' },
          ]}
          value={value}
          onChange={setValue}
          onApply={setValue}
        />
      </div>
    );
  },
};

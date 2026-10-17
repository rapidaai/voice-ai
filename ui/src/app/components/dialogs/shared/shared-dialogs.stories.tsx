import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  Information,
  Settings,
  Tag as TagIcon,
  TrashCan,
} from '@carbon/icons-react';
import { Button } from '@carbon/react';
import { useState } from 'react';
import {
  ColumnPreferencesDialog,
  ConfirmDeleteDialog,
  ConfirmDialog,
  CreateTagDialog,
  DisconnectDetailsDialog,
  UpdateDescriptionDialog,
} from '.';

const meta = {
  title: 'Dialogs/Shared/Dialogs',
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Reusable shared dialogs for confirmations, metadata editing, tagging, table preferences, and session details.',
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const ConfirmInfo: Story = {
  render: () => (
    <ConfirmDialog
      showing
      type="info"
      title="Create version"
      content="Create a new endpoint version from the current draft."
      onConfirm={() => undefined}
      onCancel={() => undefined}
      onClose={() => undefined}
    />
  ),
};

export const ConfirmDelete: Story = {
  render: () => (
    <ConfirmDeleteDialog
      showing
      title="Delete project"
      content="This removes the project and all workspace references."
      objectName="customer-support"
      onConfirm={() => undefined}
      onCancel={() => undefined}
      onClose={() => undefined}
    />
  ),
};

export const ColumnPreferences: Story = {
  render: function ColumnPreferencesStory() {
    const [open, setOpen] = useState(true);
    const [columns, setColumns] = useState([
      { name: 'Name', key: 'name', visible: true },
      { name: 'Status', key: 'status', visible: true },
      { name: 'Updated', key: 'updated', visible: false },
    ]);
    const [pageSize, setPageSize] = useState(25);

    return (
      <>
        <Button renderIcon={Settings} onClick={() => setOpen(true)}>
          Preferences
        </Button>
        <ColumnPreferencesDialog
          open={open}
          setOpen={setOpen}
          columns={columns}
          onChangeColumns={setColumns}
          defaultPageSize={[10, 25, 50]}
          pageSize={pageSize}
          onChangePageSize={setPageSize}
        />
      </>
    );
  },
};

export const TagEditor: Story = {
  render: () => (
    <CreateTagDialog
      modalOpen
      setModalOpen={() => undefined}
      title="Edit tags"
      tags={['production']}
      allTags={['production', 'support', 'voice']}
      onCreateTag={(_tags, _onError, onSuccess) =>
        onSuccess({ id: 'tag-1', name: 'production' } as never)
      }
    />
  ),
};

export const EditDetails: Story = {
  render: () => (
    <UpdateDescriptionDialog
      modalOpen
      setModalOpen={() => undefined}
      title="Edit endpoint details"
      name="Support endpoint"
      description="Answers support calls."
      onUpdateDescription={(_name, _description, _onError, onSuccess) =>
        onSuccess()
      }
    />
  ),
};

export const DisconnectDetails: Story = {
  render: () => (
    <DisconnectDetailsDialog
      modalOpen
      setModalOpen={() => undefined}
      details={[
        { label: 'Reason', value: 'Client disconnected' },
        { label: 'Session id', value: 'sess_1234567890' },
      ]}
    />
  ),
};

export const EmptyDisconnectDetails: Story = {
  render: () => (
    <DisconnectDetailsDialog
      modalOpen
      setModalOpen={() => undefined}
      details={[]}
    />
  ),
};

export const SharedUseCases: Story = {
  render: () => (
    <div className="grid gap-4 md:grid-cols-3">
      <Button renderIcon={Information}>Info confirm</Button>
      <Button kind="danger" renderIcon={TrashCan}>
        Delete confirm
      </Button>
      <Button kind="secondary" renderIcon={TagIcon}>
        Edit tags
      </Button>
    </div>
  ),
};

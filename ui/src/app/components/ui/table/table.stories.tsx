import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { Stack, Tag } from '@carbon/react';
import { ScrollableResizableTable } from './data-table';
import { LabelCell } from './label-cell';
import { Table } from './table';
import { TableBody } from './table-body';
import { TableCell } from './table-cell';
import { TableHead } from './table-head';
import { TableLink } from './table-link';
import { TablePagination } from './table-pagination';
import { TableRow } from './table-row';
import { UrlTableCell } from './url-table-cell';

const meta = {
  title: 'UI/Table',
  component: Table,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Table primitives and composed table controls used by list and detail screens.',
      },
    },
  },
} satisfies Meta<typeof Table>;

export default meta;

type Story = StoryObj<typeof meta>;

const columns = [
  { name: 'Assistant', key: 'assistant' },
  { name: 'Status', key: 'status' },
  { name: 'Endpoint', key: 'endpoint' },
];

export const BasicTable: Story = {
  render: () => (
    <Table className="w-full border border-gray-200 dark:border-gray-800">
      <TableHead columns={columns} isActionable />
      <TableBody>
        <TableRow>
          <TableCell>Billing assistant</TableCell>
          <TableCell>
            <Tag type="green">Active</Tag>
          </TableCell>
          <TableCell>
            <TableLink href="https://example.com/assistants/billing">
              View endpoint
            </TableLink>
          </TableCell>
          <TableCell>
            <span className="text-[var(--cds-text-secondary)]">Actions</span>
          </TableCell>
        </TableRow>
        <TableRow>
          <TableCell>Support assistant</TableCell>
          <LabelCell className="bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-200">
            Draft
          </LabelCell>
          <UrlTableCell url="https://api.example.com/support" />
          <TableCell>
            <span className="text-[var(--cds-text-secondary)]">Actions</span>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  ),
};

export const ResizableTable: Story = {
  render: () => (
    <div className="max-w-5xl">
      <ScrollableResizableTable
        clms={[
          { name: 'Name', key: 'name', width: 280 },
          { name: 'Owner', key: 'owner', width: 220 },
          { name: 'Last activity', key: 'activity', width: 260 },
        ]}
        isActionable
        isOptionable
        optionLabel="Menu"
      >
        <TableRow>
          <TableCell>Collections assistant</TableCell>
          <TableCell>Finance</TableCell>
          <TableCell>2 minutes ago</TableCell>
          <TableCell>Open</TableCell>
          <TableCell>...</TableCell>
        </TableRow>
        <TableRow>
          <TableCell>Routing assistant</TableCell>
          <TableCell>Operations</TableCell>
          <TableCell>1 hour ago</TableCell>
          <TableCell>Open</TableCell>
          <TableCell>...</TableCell>
        </TableRow>
      </ScrollableResizableTable>
    </div>
  ),
};

export const PaginationControls: Story = {
  render: function Render() {
    const [page, setPage] = useState(2);
    const [pageSize, setPageSize] = useState(10);
    const [columnsState, setColumnsState] = useState([
      { name: 'Assistant', key: 'assistant', visible: true },
      { name: 'Status', key: 'status', visible: true },
      { name: 'Endpoint', key: 'endpoint', visible: true },
    ]);

    return (
      <Stack gap={4} className="max-w-4xl">
        <TablePagination
          id="storybook-table-pagination"
          columns={columnsState}
          currentPage={page}
          defaultPageSize={[10, 20, 50]}
          onChangeColumns={setColumnsState}
          onChangeCurrentPage={setPage}
          onChangePageSize={setPageSize}
          pageSize={pageSize}
          totalItem={83}
        />
        <p className="text-sm text-[var(--cds-text-secondary)]">
          Page {page}, {pageSize} rows per page,{' '}
          {columnsState.filter(column => column.visible).length} visible columns
        </p>
      </Stack>
    );
  },
};

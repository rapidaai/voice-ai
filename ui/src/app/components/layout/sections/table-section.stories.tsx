import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  DataTable,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
} from '@carbon/react';
import { ScrollableTableSection, TableSection } from './table-section';

const rows = [
  { id: 'assistant', name: 'Assistant service', status: 'Active' },
  { id: 'endpoint', name: 'Endpoint gateway', status: 'Deploying' },
  { id: 'logging', name: 'Logging pipeline', status: 'Paused' },
];

const headers = [
  { key: 'name', header: 'Name' },
  { key: 'status', header: 'Status' },
];

const meta = {
  title: 'Layout/Sections/TableSection',
  component: TableSection,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Flexible table section wrappers used by page-level data tables.',
      },
    },
  },
} satisfies Meta<typeof TableSection>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <TableSection className="h-72 bg-background" aria-label="Service table">
      <ScrollableTableSection>
        <DataTable rows={rows} headers={headers}>
          {({ rows, headers, getHeaderProps, getRowProps }) => (
            <TableContainer title="Services">
              <Table>
                <TableHead>
                  <TableRow>
                    {headers.map(header => (
                      <TableHeader {...getHeaderProps({ header })}>
                        {header.header}
                      </TableHeader>
                    ))}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {rows.map(row => (
                    <TableRow {...getRowProps({ row })}>
                      {row.cells.map(cell => (
                        <TableCell key={cell.id}>{cell.value}</TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </DataTable>
      </ScrollableTableSection>
    </TableSection>
  ),
};

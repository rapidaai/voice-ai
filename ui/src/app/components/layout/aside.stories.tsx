import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  DataTable,
  Table,
  TableBody,
  TableCell,
  TableRow,
} from '@carbon/react';
import { SidebarProvider } from '@/context/sidebar-context';
import { Aside } from './aside';

const rows = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'assistants', label: 'Assistants' },
  { id: 'logs', label: 'Logs' },
];

const meta = {
  title: 'Layout/Aside',
  component: Aside,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Sidebar rail container that tracks sidebar hover state and applies shell theme tokens.',
      },
    },
  },
} satisfies Meta<typeof Aside>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Expanded: Story = {
  render: args => (
    <SidebarProvider defaultOpen>
      <div className="h-80 bg-background">
        <Aside {...args} aria-label="Primary navigation">
          <DataTable
            rows={rows}
            headers={[{ key: 'label', header: 'Navigation' }]}
          >
            {({ rows }) => (
              <Table size="sm">
                <TableBody>
                  {rows.map(row => (
                    <TableRow key={row.id}>
                      <TableCell>{row.cells[0].value}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </DataTable>
        </Aside>
      </div>
    </SidebarProvider>
  ),
};

export const Collapsed: Story = {
  render: args => (
    <SidebarProvider defaultOpen={false}>
      <div className="h-80 bg-background">
        <Aside {...args} aria-label="Primary navigation">
          <span className="sr-only">Collapsed navigation</span>
        </Aside>
      </div>
    </SidebarProvider>
  ),
};

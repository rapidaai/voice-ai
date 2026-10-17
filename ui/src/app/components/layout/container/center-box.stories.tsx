import type { Meta, StoryObj } from '@storybook/react-vite';
import { CenterBox } from './center-box';

const meta = {
  title: 'Layout/Container/CenterBox',
  component: CenterBox,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component: 'Centered page container for auth and narrow-form surfaces.',
      },
    },
  },
  args: {
    children: (
      <div className="space-y-3">
        <h2 className="text-xl font-semibold text-foreground">Sign in</h2>
        <p className="text-sm text-muted">
          Centered content uses the shell surface and border tokens.
        </p>
      </div>
    ),
  },
} satisfies Meta<typeof CenterBox>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

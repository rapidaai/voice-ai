import type { Meta, StoryObj } from '@storybook/react-vite';
import { SocialButtonGroup } from './social-button-group';

const meta = {
  title: 'UI/Primitives/SocialButtonGroup',
  component: SocialButtonGroup,
  tags: ['autodocs'],
  args: {
    google: true,
    linkedin: true,
    github: true,
    password: true,
  },
  parameters: {
    docs: {
      description: {
        component:
          'Authentication provider actions with labels matched to login or account creation.',
      },
    },
  },
} satisfies Meta<typeof SocialButtonGroup>;

export default meta;

type Story = StoryObj<typeof meta>;

export const LogIn: Story = {};

export const SignUp: Story = {
  args: {
    actionLabel: 'Sign up',
  },
};

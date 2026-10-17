import type { Meta, StoryObj } from '@storybook/react-vite';
import { ThemeProvider } from '@/theme/theme-provider';
import developmentConfig from '@/configs/config.development.json';
import { ThemeManifest } from '@/theme/types';
import { BrandedLogo } from './branded-logo';

const theme = developmentConfig.theme as unknown as ThemeManifest;
const textFallbackTheme: ThemeManifest = {
  ...theme,
  brand: {
    ...theme.brand,
    name: 'Tenant Voice',
    logos: undefined,
  },
};

const meta = {
  title: 'Layout/Brand/BrandedLogo',
  component: BrandedLogo,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Theme-aware brand mark that renders full or compact logo assets and falls back to brand text when logo assets are unavailable.',
      },
    },
  },
  args: {
    variant: 'full',
    className: 'h-8 w-auto max-w-48',
    textClassName: 'text-lg',
  },
  argTypes: {
    variant: {
      control: 'inline-radio',
      options: ['full', 'compact'],
    },
    colorMode: {
      control: 'inline-radio',
      options: ['light', 'dark'],
    },
  },
} satisfies Meta<typeof BrandedLogo>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Full: Story = {};

export const Compact: Story = {
  args: {
    variant: 'compact',
    className: 'h-8 w-8',
  },
};

export const ExplicitDarkLogo: Story = {
  args: {
    colorMode: 'dark',
  },
};

export const TextFallback: Story = {
  render: args => (
    <ThemeProvider theme={textFallbackTheme}>
      <BrandedLogo {...args} />
    </ThemeProvider>
  ),
};

import React from 'react';
import { render, screen } from '@testing-library/react';
import developmentConfig from '@/configs/config.development.json';
import { ThemeProvider } from '@/theme/theme-provider';
import { ThemeManifest } from '@/theme/types';
import { ConversationMessages } from '../index';

const theme = developmentConfig.theme as unknown as ThemeManifest;

const tenantTheme: ThemeManifest = {
  ...theme,
  id: 'tenant-voice',
  brand: {
    ...theme.brand,
    name: 'Tenant Voice',
    logos: {
      full: {
        light: '/tenant/full-light.svg',
        dark: '/tenant/full-dark.svg',
      },
      compact: {
        light: '/tenant/compact-light.svg',
        dark: '/tenant/compact-dark.svg',
      },
    },
  },
  defaultMode: 'light',
};

jest.mock('@uiw/react-markdown-preview', () => ({
  __esModule: true,
  default: ({ source }: { source: string }) => <div>{source}</div>,
}));

jest.mock('@/app/components/base/tooltip', () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/app/components/base/modal/message-feedback-modal', () => ({
  MessageFeedbackDialog: () => null,
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({ user: { name: 'Jordan' } }),
}));

jest.mock('react-router-dom', () => ({
  useSearchParams: () => [new URLSearchParams()],
}));

jest.mock('@rapidaai/react', () => ({
  Feedback: {
    Helpful: 'helpful',
    NotHelpful: 'not-helpful',
  },
  MessageRole: {
    User: 'user',
    Assistant: 'assistant',
  },
  MessageStatus: {
    Complete: 'complete',
  },
  useAgentMessages: () => ({
    messages: [
      {
        id: 'msg-1',
        role: 'assistant',
        time: new Date('2026-01-01T00:00:00Z'),
        messages: ['Hello from the assistant'],
        status: 'complete',
      },
    ],
  }),
  useMessageFeedback: () => ({
    handleHelpfulnessFeedback: jest.fn(),
    handleMessageFeedback: jest.fn(),
  }),
}));

describe('ConversationMessages', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
  });

  it('renders assistant identity from the configured theme', () => {
    render(
      <ThemeProvider theme={tenantTheme}>
        <ConversationMessages vag={{} as any} />
      </ThemeProvider>,
    );

    expect(screen.getByText('Tenant Voice')).toBeInTheDocument();
    expect(screen.getByAltText('Tenant Voice')).toHaveAttribute(
      'src',
      '/tenant/compact-light.svg',
    );
    expect(screen.queryByText('Rapida')).not.toBeInTheDocument();
  });
});

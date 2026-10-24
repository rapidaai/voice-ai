import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';

import { EditAssistant } from '@/app/pages/assistant/actions/edit-assistant';

const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

jest.mock('@rapidaai/react', () => {
  class ConnectionConfig {
    static WithDebugger(config: unknown) {
      return config;
    }
  }

  class AssistantDefinition {
    setAssistantid(_: string) {}
  }

  class GetAssistantRequest {
    setAssistantdefinition(_: unknown) {}
  }

  const GetAssistant = () =>
    Promise.resolve({
      getSuccess: () => true,
      getData: () => ({
        getName: () => 'Demo Assistant',
        getDescription: () => 'Demo Description',
      }),
    });

  return {
    ConnectionConfig,
    AssistantDefinition,
    GetAssistantRequest,
    GetAssistant,
    UpdateAssistantDetail: jest.fn(),
    DeleteAssistant: jest.fn(),
  };
});

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({ authId: 'u1', token: 't1', projectId: 'p1' }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToAssistantListing: jest.fn(),
  }),
}));

jest.mock(
  '@/app/pages/assistant/actions/hooks/use-delete-confirmation',
  () => ({
    useDeleteConfirmDialog: () => ({
      showDialog: jest.fn(),
      ConfirmDeleteDialogComponent: () => null,
    }),
  }),
);

jest.mock('@/app/components/ui/feedback/notification', () => ({
  Notification: ({ subtitle }: any) => <div>{subtitle}</div>,
}));

jest.mock('react-hot-toast/headless', () => ({
  success: jest.fn(),
  error: jest.fn(),
}));

describe('EditAssistant layout', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    window.scrollTo = jest.fn();
  });

  it('uses the shared surface token and exact General Settings heading', async () => {
    const { container } = render(<EditAssistant assistantId="assistant-1" />);

    await waitFor(() => {
      expect(
        screen.getByRole('heading', { name: 'General Settings', level: 1 }),
      ).toBeInTheDocument();
    });

    const pageRoot = container.firstElementChild as HTMLElement;
    expect(pageRoot).toHaveClass('bg-surface');
    expect(pageRoot).not.toHaveClass('bg-white', 'dark:bg-gray-900');
  });

  it('uses label toggletips instead of helper text for edit fields', async () => {
    const { container } = render(<EditAssistant assistantId="assistant-1" />);

    await waitFor(() => {
      expect(screen.getByDisplayValue('Demo Assistant')).toBeInTheDocument();
    });

    expect(
      screen.getByRole('button', { name: 'Assistant ID information' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Name information' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Description information' }),
    ).toBeInTheDocument();
    expect(container.querySelector('.cds--form__helper-text')).toBeNull();
  });

  it('shows assistant id as an inline copyable value instead of a wide input', async () => {
    render(<EditAssistant assistantId="assistant-1" />);

    await waitFor(() => {
      expect(screen.getByText('assistant-1')).toBeInTheDocument();
    });

    expect(screen.queryByDisplayValue('assistant-1')).not.toBeInTheDocument();
    expect(screen.getByText('assistant-1').parentElement).toHaveClass(
      'inline-flex',
    );
    expect(screen.getByText('assistant-1').parentElement).not.toHaveClass(
      'border',
    );
    expect(screen.getByRole('button', { name: 'Copy' })).toBeInTheDocument();
  });
});

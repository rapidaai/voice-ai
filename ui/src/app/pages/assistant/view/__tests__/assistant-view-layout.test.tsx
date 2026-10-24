import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AssistantViewLayout } from '../assistant-view.layout';
import { getAssistantById } from '@/clients/assistant.client';

jest.mock('@/clients/assistant.client', () => ({
  getAssistantById: jest.fn(),
}));

const mockClearAssistantState = jest.fn();
const mockSetCurrentAssistant = jest.fn();

jest.mock('@/stores/assistant/assistant.store', () => ({
  useAssistantPageStore: (selector?: (state: any) => unknown) => {
    const state = {
      currentAssistant: null,
      clear: mockClearAssistantState,
      onChangeCurrentAssistant: mockSetCurrentAssistant,
    };
    return selector ? selector(state) : state;
  },
}));

const mockHideLoader = jest.fn();
const mockShowLoader = jest.fn();

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    hideLoader: mockHideLoader,
    showLoader: mockShowLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

const mockGoToAssistantListing = jest.fn();

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToAssistantListing: mockGoToAssistantListing,
    goToAssistantPreview: jest.fn(),
    goToAssistantPreviewCall: jest.fn(),
  }),
}));

jest.mock('../assistant-side-nav', () => ({
  AssistantSideNav: ({ expanded }: { expanded: boolean }) => (
    <nav aria-label="Assistant navigation" data-expanded={String(expanded)} />
  ),
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/app/components/ui/feedback', () => ({
  ErrorContainer: ({ title, description, actionLabel, onAction }: any) => (
    <section>
      <h1>{title}</h1>
      <p>{description}</p>
      <button onClick={onAction}>{actionLabel}</button>
    </section>
  ),
}));

const mockedGetAssistantById = getAssistantById as jest.MockedFunction<
  typeof getAssistantById
>;

function renderLayout() {
  render(
    <MemoryRouter
      initialEntries={['/deployment/assistant/assistant-1/overview']}
    >
      <Routes>
        <Route
          path="/deployment/assistant/:assistantId"
          element={<AssistantViewLayout />}
        >
          <Route path="overview" element={<div>Assistant overview</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe('AssistantViewLayout', () => {
  const originalMatchMedia = window.matchMedia;

  beforeEach(() => {
    jest.clearAllMocks();
    window.matchMedia = originalMatchMedia;
  });

  it('renders the assistant content after a successful load', async () => {
    const assistant = { getId: () => 'assistant-1' } as any;
    mockedGetAssistantById.mockResolvedValue({
      getSuccess: () => true,
      getData: () => assistant,
    } as any);

    renderLayout();

    await waitFor(() =>
      expect(mockSetCurrentAssistant).toHaveBeenCalledWith(assistant),
    );
    expect(screen.getByText('Assistant overview')).toBeInTheDocument();
    expect(mockHideLoader).toHaveBeenCalled();
  });

  it('renders the unavailable state after a transport failure', async () => {
    mockedGetAssistantById.mockRejectedValue(new Error('network unavailable'));

    renderLayout();

    expect(
      await screen.findByRole('heading', { name: 'Assistant not available' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Back to assistants' }),
    ).toBeVisible();
    expect(mockHideLoader).toHaveBeenCalled();
  });

  it('starts with the assistant navigation collapsed on a small screen', async () => {
    window.matchMedia = jest.fn().mockReturnValue({
      matches: true,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
    });
    mockedGetAssistantById.mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({ getId: () => 'assistant-1' }),
    } as any);

    renderLayout();

    expect(
      await screen.findByRole('navigation', { name: 'Assistant navigation' }),
    ).toHaveAttribute('data-expanded', 'false');
  });
});

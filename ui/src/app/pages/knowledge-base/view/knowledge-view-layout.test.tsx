import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, useRoutes } from 'react-router-dom';
import { getKnowledgeBaseDetail } from '@/clients/knowledge.client';
import { useKnowledgePageStore } from '@/stores/knowledge/knowledge.store';
import { KnowledgeViewLayout } from './knowledge-view.layout';

const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockGoTo = jest.fn();

jest.mock('@/clients/knowledge.client', () => ({
  getKnowledgeBaseDetail: jest.fn(),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({ goTo: mockGoTo }),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  ErrorContainer: ({ title, actionLabel, onAction }: any) => (
    <section>
      <h1>{title}</h1>
      <button onClick={onAction}>{actionLabel}</button>
    </section>
  ),
}));

const mockedGetKnowledgeBaseDetail =
  getKnowledgeBaseDetail as jest.MockedFunction<typeof getKnowledgeBaseDetail>;

const KnowledgeRoutes = () =>
  useRoutes([
    {
      path: '/knowledge/:id',
      element: <KnowledgeViewLayout />,
      children: [
        {
          path: 'add-knowledge-file',
          element: <div>Upload knowledge file</div>,
        },
      ],
    },
  ]);

function renderFileRoute() {
  return render(
    <MemoryRouter
      initialEntries={['/knowledge/knowledge-1/add-knowledge-file']}
    >
      <KnowledgeRoutes />
    </MemoryRouter>,
  );
}

describe('KnowledgeViewLayout', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useKnowledgePageStore.setState({
      currentKnowledge: null,
      currentKnowledgeStatus: 'idle',
    });
  });

  it('loads the parent knowledge for a direct file-route visit', async () => {
    const knowledge = {
      getId: () => 'knowledge-1',
      getName: () => 'Product knowledge',
    } as any;
    mockedGetKnowledgeBaseDetail.mockImplementation(({ callback }: any) =>
      callback(null, {
        getSuccess: () => true,
        getData: () => knowledge,
      }),
    );

    renderFileRoute();

    expect(screen.getByText('Upload knowledge file')).toBeInTheDocument();
    await waitFor(() =>
      expect(useKnowledgePageStore.getState().currentKnowledge).toBe(knowledge),
    );
    expect(useKnowledgePageStore.getState().currentKnowledgeStatus).toBe(
      'ready',
    );
  });

  it('renders an unavailable state when the parent lookup fails', async () => {
    mockedGetKnowledgeBaseDetail.mockImplementation(({ callback }: any) =>
      callback(null, { getSuccess: () => false, getError: () => null }),
    );

    renderFileRoute();

    expect(
      await screen.findByRole('heading', { name: 'Knowledge not available' }),
    ).toBeInTheDocument();
    expect(useKnowledgePageStore.getState().currentKnowledgeStatus).toBe(
      'error',
    );
  });
});

import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  MemoryRouter,
  useNavigate,
  useOutletContext,
  useRoutes,
} from 'react-router-dom';
import { getKnowledgeBaseDetail } from '@/clients/knowledge.client';
import { useKnowledgePageStore } from '@/stores/knowledge/knowledge.store';
import { KnowledgeViewLayout } from './knowledge-view.layout';

const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockGoTo = jest.fn();
let mockUserId = 'user-1';
let mockToken = 'token-1';
let mockProjectId = 'project-1';

jest.mock('@/clients/knowledge.client', () => ({
  getKnowledgeBaseDetail: jest.fn(),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => [mockUserId, mockToken, mockProjectId],
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

function KnowledgeFilePage() {
  const navigate = useNavigate();
  const { onReload } = useOutletContext<{ onReload: () => void }>();
  return (
    <>
      <div>Upload knowledge file</div>
      <button type="button" onClick={onReload}>
        Reload knowledge
      </button>
      <button
        type="button"
        onClick={() => navigate('/knowledge/knowledge-2/add-knowledge-file')}
      >
        Open next knowledge
      </button>
    </>
  );
}

const KnowledgeRoutes = () =>
  useRoutes([
    {
      path: '/knowledge/:id',
      element: <KnowledgeViewLayout />,
      children: [
        {
          path: 'add-knowledge-file',
          element: <KnowledgeFilePage />,
        },
      ],
    },
  ]);

function KnowledgeFileRoute() {
  return (
    <MemoryRouter
      initialEntries={['/knowledge/knowledge-1/add-knowledge-file']}
    >
      <KnowledgeRoutes />
    </MemoryRouter>
  );
}

function renderFileRoute() {
  return render(<KnowledgeFileRoute />);
}

describe('KnowledgeViewLayout', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUserId = 'user-1';
    mockToken = 'token-1';
    mockProjectId = 'project-1';
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

  it('ignores a response from the previous knowledge route', async () => {
    const user = userEvent.setup();
    const callbacks: Array<(error: null, response: any) => void> = [];
    const knowledgeTwo = {
      getId: () => 'knowledge-2',
      getName: () => 'Second knowledge',
    } as any;
    mockedGetKnowledgeBaseDetail.mockImplementation(({ callback }: any) => {
      callbacks.push(callback);
    });

    renderFileRoute();
    await user.click(
      screen.getByRole('button', { name: 'Open next knowledge' }),
    );
    await waitFor(() => expect(callbacks).toHaveLength(2));

    await act(async () => {
      callbacks[1](null, {
        getSuccess: () => true,
        getData: () => knowledgeTwo,
      });
    });
    await act(async () => {
      callbacks[0](null, {
        getSuccess: () => false,
        getError: () => null,
      });
    });

    expect(useKnowledgePageStore.getState().currentKnowledge).toBe(
      knowledgeTwo,
    );
    expect(useKnowledgePageStore.getState().currentKnowledgeStatus).toBe(
      'ready',
    );
    expect(
      screen.queryByRole('heading', { name: 'Knowledge not available' }),
    ).not.toBeInTheDocument();
  });

  it('ignores a response after the knowledge layout unmounts', async () => {
    let callback: (error: null, response: any) => void = () => undefined;
    mockedGetKnowledgeBaseDetail.mockImplementation((request: any) => {
      callback = request.callback;
    });

    const view = renderFileRoute();
    await waitFor(() =>
      expect(mockedGetKnowledgeBaseDetail).toHaveBeenCalledTimes(1),
    );
    view.unmount();
    await act(async () => {
      callback(null, { getSuccess: () => false, getError: () => null });
    });

    expect(useKnowledgePageStore.getState().currentKnowledgeStatus).not.toBe(
      'error',
    );
    expect(mockHideLoader).toHaveBeenCalledTimes(1);
  });

  it('keeps the current page when a reload fails', async () => {
    const user = userEvent.setup();
    const knowledge = {
      getId: () => 'knowledge-1',
      getName: () => 'Product knowledge',
    } as any;
    mockedGetKnowledgeBaseDetail
      .mockImplementationOnce(({ callback }: any) =>
        callback(null, {
          getSuccess: () => true,
          getData: () => knowledge,
        }),
      )
      .mockImplementationOnce(({ callback }: any) =>
        callback(null, { getSuccess: () => false, getError: () => null }),
      );

    renderFileRoute();
    await waitFor(() =>
      expect(useKnowledgePageStore.getState().currentKnowledge).toBe(knowledge),
    );
    await user.click(screen.getByRole('button', { name: 'Reload knowledge' }));

    expect(screen.getByText('Upload knowledge file')).toBeInTheDocument();
    expect(useKnowledgePageStore.getState().currentKnowledge).toBe(knowledge);
    expect(useKnowledgePageStore.getState().currentKnowledgeStatus).toBe(
      'ready',
    );
  });

  it('clears cached knowledge when the project context changes', async () => {
    const knowledge = {
      getId: () => 'knowledge-1',
      getName: () => 'First project knowledge',
    } as any;
    mockedGetKnowledgeBaseDetail
      .mockImplementationOnce(({ callback }: any) =>
        callback(null, {
          getSuccess: () => true,
          getData: () => knowledge,
        }),
      )
      .mockImplementationOnce(({ callback }: any) =>
        callback(null, { getSuccess: () => false, getError: () => null }),
      );

    const view = renderFileRoute();
    await waitFor(() =>
      expect(useKnowledgePageStore.getState().currentKnowledge).toBe(knowledge),
    );

    mockProjectId = 'project-2';
    view.rerender(<KnowledgeFileRoute />);

    expect(
      await screen.findByRole('heading', { name: 'Knowledge not available' }),
    ).toBeInTheDocument();
    expect(useKnowledgePageStore.getState().currentKnowledge).toBeNull();
    expect(mockedGetKnowledgeBaseDetail).toHaveBeenLastCalledWith(
      expect.objectContaining({
        auth: expect.objectContaining({ projectId: 'project-2' }),
      }),
    );
  });
});

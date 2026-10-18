import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { ConfigureAssistantKnowledgePage } from './index';

const mockGetAssistantKnowledge = jest.fn();
const mockGoToCreateAssistantKnowledge = jest.fn();
const mockGoToCreateKnowledge = jest.fn();
const mockHideLoader = jest.fn();
const mockShowLoader = jest.fn();

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
  success: jest.fn(),
}));

jest.mock('react-router-dom', () => ({
  useParams: () => ({ assistantId: 'assistant-1' }),
}));

jest.mock('@/app/components/dialogs/shared/confirm-ui', () => () => null);

jest.mock('@/app/components/domain/cards/knowledge-card', () => ({
  SelectKnowledgeCard: () => <article>Knowledge card</article>,
}));

jest.mock('@/app/components/layout/blocks/page-header-block', () => ({
  PageHeaderBlock: ({ children }: any) => <header>{children}</header>,
}));

jest.mock('@/app/components/layout/blocks/page-title-block', () => ({
  PageTitleBlock: ({ children }: any) => <h1>{children}</h1>,
}));

jest.mock('@/app/components/ui/feedback/empty-state', () => ({
  EmptyState: ({ action, onAction, title }: any) => (
    <section>
      <h2>{title}</h2>
      <button type="button" onClick={onAction}>
        {action}
      </button>
    </section>
  ),
}));

jest.mock('@/app/components/ui/feedback/loaders/section-loader', () => ({
  SectionLoader: () => <div data-testid="section-loader" />,
}));

jest.mock('@/app/components/ui/table/table-pagination', () => ({
  TablePagination: () => <nav data-testid="table-pagination" />,
}));

jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-knowledge/create-assistant-knowledge',
  () => ({
    CreateKnowledge: () => <div>Create knowledge form</div>,
  }),
);

jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-knowledge/update-assistant-knowledge',
  () => ({
    UpdateKnowledge: () => <div>Update knowledge form</div>,
  }),
);

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    ConfirmDialogComponent: () => null,
    showDialog: (callback: () => void) => callback(),
  }),
}));

jest.mock('@/stores/assistant/actions', () => ({
  useAssistantKnowledgePageStore: () => ({
    deleteAssistantKnowledge: jest.fn(),
    getAssistantKnowledge: mockGetAssistantKnowledge,
    knowledges: [],
    page: 1,
    pageSize: 10,
    setColumns: jest.fn(),
    setPage: jest.fn(),
    setPageSize: jest.fn(),
    totalCount: 0,
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    hideLoader: mockHideLoader,
    loading: false,
    showLoader: mockShowLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    projectId: 'project-1',
    token: 'token-1',
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToCreateAssistantKnowledge: mockGoToCreateAssistantKnowledge,
    goToCreateKnowledge: mockGoToCreateKnowledge,
    goToEditAssistantKnowledge: jest.fn(),
  }),
}));

jest.mock('@carbon/icons-react', () => ({
  Add: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="add-icon"
    />
  ),
  Information: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="information-icon"
    />
  ),
  Launch: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="launch-icon"
    />
  ),
}));

describe('ConfigureAssistantKnowledgePage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders knowledge actions with Carbon icons', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<ConfigureAssistantKnowledgePage />);

    expect(screen.getByText('Configure Knowledge')).toBeInTheDocument();
    expect(screen.getByTestId('launch-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );
    expect(screen.getByTestId('add-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );
    expect(screen.getByTestId('information-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );

    await user.click(
      screen.getByRole('button', { name: /create new knowledge/i }),
    );
    await user.click(
      screen.getByRole('button', { name: /connect knowledge/i }),
    );

    expect(mockGoToCreateKnowledge).toHaveBeenCalledTimes(1);
    expect(mockGoToCreateAssistantKnowledge).toHaveBeenCalledWith(
      'assistant-1',
    );
    await waitFor(() => expect(mockGetAssistantKnowledge).toHaveBeenCalled());
  });

  it('routes empty-state action to assistant knowledge creation', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<ConfigureAssistantKnowledgePage />);

    await user.click(screen.getByRole('button', { name: /connect knowlege/i }));

    expect(mockGoToCreateAssistantKnowledge).toHaveBeenCalledWith(
      'assistant-1',
    );
  });
});

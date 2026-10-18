import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { KnowledgePage } from './index';

const mockGoToCreateKnowledge = jest.fn();
const mockGetAllKnowledge = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockSetPage = jest.fn();
const mockSetPageSize = jest.fn();

const mockKnowledgeActions = {
  criteria: [],
  getAllKnowledge: mockGetAllKnowledge,
  knowledgeBases: [
    {
      getId: () => 'knowledge-1',
      getName: () => 'Product knowledge',
    },
  ],
  page: 1,
  pageSize: 10,
  setPage: mockSetPage,
  setPageSize: mockSetPageSize,
  totalCount: 1,
};

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: ({ title }: any) => <div data-testid="helmet">{title}</div>,
}));

jest.mock(
  '@/app/components/dialogs/knowledge/how-knowledge-works-modal',
  () => ({
    HowKnowledgeWorksDialog: ({ modalOpen }: any) =>
      modalOpen ? <div data-testid="how-knowledge-works-dialog" /> : null,
  }),
);

jest.mock('@/app/components/domain/cards/knowledge-card', () => ({
  ClickableKnowledgeCard: ({ knowledge }: any) => (
    <article>{knowledge.getName()}</article>
  ),
}));

jest.mock('@/app/components/layout/blocks/page-header-block', () => ({
  PageHeaderBlock: ({ children }: any) => <header>{children}</header>,
}));

jest.mock('@/app/components/layout/blocks/page-title-block', () => ({
  PageTitleBlock: ({ children }: any) => <h1>{children}</h1>,
}));

jest.mock('@/app/components/layout/blocks/pagination-button-block', () => ({
  PaginationButtonBlock: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/layout/wrapper/blured-wrapper', () => ({
  BluredWrapper: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  GhostButton: ({
    children,
    hasIconOnly,
    iconDescription,
    renderIcon: Icon,
    size,
    ...props
  }: React.ComponentProps<'button'> & {
    hasIconOnly?: boolean;
    iconDescription?: string;
    renderIcon?: React.ElementType;
    size?: string;
  }) => (
    <button type="button" aria-label={iconDescription} {...props}>
      {Icon ? <Icon /> : null}
      {children}
    </button>
  ),
}));

jest.mock('@/app/components/ui/feedback/empty-state', () => ({
  EmptyState: ({ action, onAction, title }: any) => (
    <section>
      <h2>{title}</h2>
      {action ? (
        <button type="button" onClick={onAction}>
          {action}
        </button>
      ) : null}
    </section>
  ),
}));

jest.mock('@/app/components/ui/composites/icon-input', () => ({
  SearchIconInput: () => <input aria-label="Search" type="search" />,
}));

jest.mock('@/app/components/ui/feedback/loading', () => ({
  PageLoading: () => <div data-testid="page-loading" />,
}));

jest.mock('@/app/components/ui/table/table-pagination', () => ({
  TablePagination: () => <nav data-testid="table-pagination" />,
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    hideLoader: mockHideLoader,
    loading: false,
    showLoader: mockShowLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToCreateKnowledge: mockGoToCreateKnowledge,
  }),
}));

jest.mock('@/stores/knowledge/knowledge.store', () => ({
  useKnowledgePageStore: () => mockKnowledgeActions,
}));

jest.mock('@carbon/icons-react', () => ({
  Add: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="add-icon"
    />
  ),
  Renew: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="renew-icon"
    />
  ),
}));

describe('KnowledgePage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders knowledge actions with Carbon icons', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<KnowledgePage />);

    expect(screen.getByText('Product knowledge')).toBeInTheDocument();
    expect(screen.getByTestId('add-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );
    expect(screen.getByTestId('renew-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );

    await user.click(
      screen.getByRole('button', { name: /add new knowledge/i }),
    );

    expect(mockGoToCreateKnowledge).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(mockShowLoader).toHaveBeenCalledTimes(1));
  });

  it('refreshes knowledge bases from the labeled icon action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<KnowledgePage />);

    await waitFor(() => expect(mockGetAllKnowledge).toHaveBeenCalledTimes(1));

    await user.click(
      screen.getByRole('button', { name: 'Refresh knowledge bases' }),
    );

    expect(mockGetAllKnowledge).toHaveBeenLastCalledWith(
      'project-1',
      'token-1',
      'user-1',
      expect.any(Function),
      expect.any(Function),
    );
    expect(mockGetAllKnowledge).toHaveBeenCalledTimes(2);
  });
});

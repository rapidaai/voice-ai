import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { ViewKnowledgePage } from './index';

const mockHideLoader = jest.fn();
const mockOnChangeCurrentKnowledge = jest.fn();
const mockShowLoader = jest.fn();

let mockCurrentKnowledge: any = {
  getCreateddate: () => '2026-01-01T00:00:00Z',
  getDescription: () => 'Useful internal docs',
  getId: () => 'knowledge-1',
  getKnowledgetag: () => ({ getTagList: () => [] }),
  getName: () => 'Product knowledge',
};

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
}));

jest.mock('react-router-dom', () => ({
  useParams: () => ({ id: 'knowledge-1' }),
}));

jest.mock('@rapidaai/react', () => ({
  ConnectionConfig: {
    WithDebugger: jest.fn(metadata => metadata),
  },
  GetKnowledgeBase: jest.fn((_config, _id, callback) => {
    callback(null, {
      getData: () => mockCurrentKnowledge,
      getSuccess: () => true,
    });
  }),
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: ({ title }: any) => <div data-testid="helmet">{title}</div>,
}));

jest.mock(
  '@/app/components/dialogs/knowledge/create-knowledge-document-modal',
  () => ({
    CreateKnowledgeDocumentDialog: ({ knowledgeId, modalOpen }: any) =>
      modalOpen ? (
        <div data-testid="create-document-dialog">{knowledgeId}</div>
      ) : null,
  }),
);

jest.mock('@/app/components/dialogs/shared/create-tag-modal', () => ({
  CreateTagDialog: () => <div data-testid="create-tag-dialog" />,
}));

jest.mock('@/app/components/dialogs/shared/update-description-modal', () => ({
  UpdateDescriptionDialog: () => (
    <div data-testid="update-description-dialog" />
  ),
}));

jest.mock('@/app/components/layout/blocks/page-header-block', () => ({
  PageHeaderBlock: ({ children }: any) => <header>{children}</header>,
}));

jest.mock('@/app/components/layout/blocks/page-title-block', () => ({
  PageTitleBlock: ({ children }: any) => <h1>{children}</h1>,
}));

jest.mock('@/app/components/ui/primitives/tabs', () => ({
  Tab: ({ tabs }: any) => (
    <div>
      {tabs.map((tab: any) => (
        <section key={tab.label}>{tab.element}</section>
      ))}
    </div>
  ),
}));

jest.mock('@/app/pages/knowledge-base/view/document-segments', () => ({
  DocumentSegments: () => <div>Segments tab</div>,
}));

jest.mock('./documents', () => ({
  Documents: () => <div>Documents tab</div>,
}));

jest.mock('@/configs', () => ({
  connectionConfig: {},
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    hideLoader: mockHideLoader,
    showLoader: mockShowLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/stores/knowledge/knowledge.store', () => ({
  useKnowledgePageStore: () => ({
    currentKnowledge: mockCurrentKnowledge,
    editTagVisible: false,
    onChangeCurrentKnowledge: mockOnChangeCurrentKnowledge,
    onEditKnowledgeTag: jest.fn(),
    onHideEditTagVisible: jest.fn(),
    onHideUpdateDescription: jest.fn(),
    onUpdateKnowledgeDetail: jest.fn(),
    updateDescriptionVisible: false,
  }),
}));

jest.mock('@/utils/date', () => ({
  toHumanReadableRelativeTime: () => 'recently',
}));

jest.mock('@carbon/icons-react', () => ({
  Add: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="add-icon"
    />
  ),
}));

describe('ViewKnowledgePage', () => {
  beforeEach(() => {
    mockCurrentKnowledge = {
      getCreateddate: () => '2026-01-01T00:00:00Z',
      getDescription: () => 'Useful internal docs',
      getId: () => 'knowledge-1',
      getKnowledgetag: () => ({ getTagList: () => [] }),
      getName: () => 'Product knowledge',
    };
    jest.clearAllMocks();
  });

  it('renders the add document action with a Carbon icon', async () => {
    render(<ViewKnowledgePage />);

    expect(screen.getByRole('button', { name: /add new document/i }));
    expect(screen.getByTestId('add-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );

    await waitFor(() => expect(mockShowLoader).toHaveBeenCalledWith('overlay'));
  });

  it('opens create document dialog for the current knowledge', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<ViewKnowledgePage />);

    await user.click(screen.getByRole('button', { name: /add new document/i }));

    expect(screen.getByTestId('create-document-dialog')).toHaveTextContent(
      'knowledge-1',
    );
  });
});

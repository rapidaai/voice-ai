import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { DeleteKnowledgeDocumentSegment } from '@rapidaai/react';
import { CreateKnowledgeDocumentDialog } from '../create-knowledge-document-modal';
import { DeleteKnowledgeDocumentSegmentDialog } from '../delete-knowledge-document-segment-modal';
import { HowKnowledgeWorksDialog } from '../how-knowledge-works-modal';

const mockCreateStore = {
  clear: jest.fn(),
  onCreateKnowledgeDocument: jest.fn(),
};

const mockRapidaStore = {
  loading: false,
  showLoader: jest.fn(),
  hideLoader: jest.fn(),
};

jest.mock('@rapidaai/react', () => ({
  DeleteKnowledgeDocumentSegment: jest.fn(),
}));

jest.mock('@/configs', () => ({
  connectionConfig: { endpoint: 'test-endpoint' },
}));

jest.mock('@/stores/knowledge/create-knowledge-document.store', () => ({
  useCreateKnowledgeDocumentPageStore: () => mockCreateStore,
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
  useCurrentCredential: () => ({
    authId: 'auth-1',
    projectId: 'project-1',
    token: 'token-1',
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => mockRapidaStore,
}));

jest.mock(
  '@/app/pages/knowledge-base/action/components/datasource-uploader/manual-file',
  () => ({
    ManualFile: () => <div data-testid="manual-file">Manual upload</div>,
  }),
);

jest.mock('@/app/components/dialogs/shared', () => ({
  HowItWorksDialog: ({ modalOpen, steps, title = 'How it works' }: any) =>
    modalOpen ? (
      <section>
        <h2>{title}</h2>
        {steps.map((step: any) => (
          <article key={step.title}>
            {step.icon}
            <h3>{step.title}</h3>
            <p>{step.description}</p>
          </article>
        ))}
      </section>
    ) : null,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  DangerButton: ({ children, ...props }: any) => (
    <button data-kind="danger" {...props}>
      {children}
    </button>
  ),
  GhostButton: ({ children, ...props }: any) => (
    <button data-kind="ghost" {...props}>
      {children}
    </button>
  ),
  Modal: ({ open, children, onClose, danger }: any) =>
    open ? (
      <div data-danger={String(Boolean(danger))}>
        <button type="button" onClick={onClose}>
          Modal close
        </button>
        {children}
      </div>
    ) : null,
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalFooter: ({ children, danger }: any) => (
    <footer data-danger={String(Boolean(danger))}>{children}</footer>
  ),
  ModalHeader: ({ label, title, onClose }: any) => (
    <header>
      {label ? <p>{label}</p> : null}
      <h2>{title}</h2>
      <button type="button" onClick={onClose}>
        Header close
      </button>
    </header>
  ),
  PrimaryButton: ({ children, isLoading, ...props }: any) => (
    <button data-loading={String(Boolean(isLoading))} {...props}>
      {children}
    </button>
  ),
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  TextArea: ({ id, labelText, value, onChange, ...props }: any) => (
    <label htmlFor={id}>
      {labelText}
      <textarea id={id} value={value} onChange={onChange} {...props} />
    </label>
  ),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  Notification: ({ title, subtitle }: any) => (
    <div role="alert">
      {title}: {subtitle}
    </div>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  CloudUpload: () => <svg data-testid="upload-icon" />,
  DataBase: () => <svg data-testid="database-icon" />,
  Rocket: () => <svg data-testid="rocket-icon" />,
}));

const mockDeleteKnowledgeDocumentSegment =
  DeleteKnowledgeDocumentSegment as jest.Mock;

const makeSegment = () => ({
  getDocumentId: () => 'document-1',
  getIndex: () => 2,
});

describe('knowledge dialogs', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockRapidaStore.loading = false;
  });

  it('creates a knowledge document and reloads after success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    const onReload = jest.fn();
    mockCreateStore.onCreateKnowledgeDocument.mockImplementation(
      (_knowledgeId, _projectId, _token, _userId, onSuccess) => {
        onSuccess([]);
      },
    );

    render(
      <CreateKnowledgeDocumentDialog
        modalOpen
        setModalOpen={setModalOpen}
        knowledgeId="knowledge-1"
        onReload={onReload}
      />,
    );

    expect(mockCreateStore.clear).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('manual-file')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Create Document' }));

    expect(mockRapidaStore.showLoader).toHaveBeenCalledWith('overlay');
    expect(mockCreateStore.onCreateKnowledgeDocument).toHaveBeenCalledWith(
      'knowledge-1',
      'project-1',
      'token-1',
      'user-1',
      expect.any(Function),
      expect.any(Function),
    );
    expect(mockRapidaStore.hideLoader).toHaveBeenCalledTimes(1);
    expect(setModalOpen).toHaveBeenCalledWith(false);
    expect(onReload).toHaveBeenCalledTimes(1);
  });

  it('shows create document errors without closing', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    mockCreateStore.onCreateKnowledgeDocument.mockImplementation(
      (_knowledgeId, _projectId, _token, _userId, _onSuccess, onError) => {
        onError('Upload failed');
      },
    );

    render(
      <CreateKnowledgeDocumentDialog
        modalOpen
        setModalOpen={setModalOpen}
        knowledgeId="knowledge-1"
        onReload={jest.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Create Document' }));

    expect(screen.getByRole('alert')).toHaveTextContent('Error: Upload failed');
    expect(setModalOpen).not.toHaveBeenCalledWith(false);
  });

  it('requires a delete reason before deleting a segment', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <DeleteKnowledgeDocumentSegmentDialog
        segment={makeSegment() as any}
        onClose={jest.fn()}
        onDelete={jest.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Delete Segment' }));

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Error: Please provide a reason for deletion.',
    );
    expect(mockDeleteKnowledgeDocumentSegment).not.toHaveBeenCalled();
  });

  it('deletes a segment with credentials and closes after success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onClose = jest.fn();
    const onDelete = jest.fn();
    mockDeleteKnowledgeDocumentSegment.mockImplementation(
      (_config, _documentId, _index, _reason, callback) => callback(null, {}),
    );

    render(
      <DeleteKnowledgeDocumentSegmentDialog
        segment={makeSegment() as any}
        onClose={onClose}
        onDelete={onDelete}
      />,
    );

    await user.click(screen.getByLabelText('Reason'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String(' outdated '));
    await user.click(screen.getByRole('button', { name: 'Delete Segment' }));

    expect(mockDeleteKnowledgeDocumentSegment).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'document-1',
      '2',
      'outdated',
      expect.any(Function),
      {
        authorization: 'token-1',
        'x-project-id': 'project-1',
        'x-auth-id': 'auth-1',
      },
    );
    expect(onDelete).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('shows delete segment API errors', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockDeleteKnowledgeDocumentSegment.mockImplementation(
      (_config, _documentId, _index, _reason, callback) =>
        callback({ message: 'failed' }, null),
    );

    render(
      <DeleteKnowledgeDocumentSegmentDialog
        segment={makeSegment() as any}
        onClose={jest.fn()}
        onDelete={jest.fn()}
      />,
    );

    await user.click(screen.getByLabelText('Reason'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('outdated'));
    await user.click(screen.getByRole('button', { name: 'Delete Segment' }));

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Error: Failed to delete the segment. Please try again.',
    );
  });

  it('documents the knowledge workflow with Carbon icons', () => {
    render(<HowKnowledgeWorksDialog modalOpen setModalOpen={jest.fn()} />);

    expect(screen.getByText('Upload and Explore')).toBeInTheDocument();
    expect(screen.getByText('Build Knowledge Base')).toBeInTheDocument();
    expect(screen.getByText('Deploy and Integrate')).toBeInTheDocument();
    expect(screen.getByTestId('upload-icon')).toBeInTheDocument();
    expect(screen.getByTestId('database-icon')).toBeInTheDocument();
    expect(screen.getByTestId('rocket-icon')).toBeInTheDocument();
  });
});

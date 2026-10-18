import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { UpdateKnowledgeDocumentSegment } from '@rapidaai/react';
import {
  EditKnowledgeDocumentSegmentDialog,
  parseSegmentEntityList,
} from '../edit-knowledge-document-segment-modal';

jest.mock('@rapidaai/react', () => ({
  UpdateKnowledgeDocumentSegment: jest.fn(),
}));

jest.mock('@/configs', () => ({
  connectionConfig: { endpoint: 'test-endpoint' },
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'auth-1',
    projectId: 'project-1',
    token: 'token-1',
  }),
}));

jest.mock('@/app/components/ui/primitives', () => ({
  Modal: ({ open, children }: any) => (open ? <div>{children}</div> : null),
  ModalHeader: ({ title, onClose }: any) => (
    <header>
      <h2>{title}</h2>
      <button type="button" onClick={onClose}>
        Header close
      </button>
    </header>
  ),
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalFooter: ({ children }: any) => <footer>{children}</footer>,
  PrimaryButton: ({ children, renderIcon: Icon, ...props }: any) => (
    <button {...props}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: ({ labelText, id, ...props }: any) => (
    <label htmlFor={id}>
      {labelText}
      <input id={id} {...props} />
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
  Checkmark: () => <svg data-testid="update-icon" />,
}));

const mockUpdateKnowledgeDocumentSegment =
  UpdateKnowledgeDocumentSegment as jest.Mock;

const makeSegment = () => ({
  getDocumentId: () => 'document-1',
  getIndex: () => 7,
  getMetadata: () => ({
    getDocumentName: () => 'Original document',
  }),
  getEntities: () => ({
    getOrganizationsList: () => ['Org A'],
    getDatesList: () => ['2026-09-19'],
    getProductsList: () => [],
    getEventsList: () => [],
    getIndustriesList: () => [],
    getLocationsList: () => [],
    getPeopleList: () => [],
    getTimesList: () => [],
    getQuantitiesList: () => [],
  }),
});

describe('EditKnowledgeDocumentSegmentDialog', () => {
  beforeEach(() => {
    mockUpdateKnowledgeDocumentSegment.mockReset();
  });

  it('renders the segment fields with the standard update icon', () => {
    render(
      <EditKnowledgeDocumentSegmentDialog
        segment={makeSegment() as any}
        onClose={jest.fn()}
        onUpdate={jest.fn()}
      />,
    );

    expect(
      screen.getByRole('heading', { name: 'Edit Document Segment' }),
    ).toBeInTheDocument();
    expect(screen.getByDisplayValue('document-1')).toBeDisabled();
    expect(screen.getByDisplayValue('Original document')).toBeInTheDocument();
    expect(screen.getByTestId('update-icon')).toBeInTheDocument();
  });

  it('submits trimmed entity values and closes after a successful update', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onClose = jest.fn();
    const onUpdate = jest.fn();
    mockUpdateKnowledgeDocumentSegment.mockImplementation(
      (...args: unknown[]) => {
        const callback = args[13] as Function;
        callback(null, {});
      },
    );

    render(
      <EditKnowledgeDocumentSegmentDialog
        segment={makeSegment() as any}
        onClose={onClose}
        onUpdate={onUpdate}
      />,
    );

    await user.click(
      screen.getByPlaceholderText('Enter organizations separated by commas'),
    );
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Org A, Org B, '));
    await user.click(screen.getByRole('button', { name: 'Update document' }));

    expect(mockUpdateKnowledgeDocumentSegment).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'document-1',
      '7',
      ['Org A', 'Org B'],
      ['2026-09-19'],
      [],
      [],
      [],
      [],
      [],
      [],
      [],
      'Original document',
      expect.any(Function),
      {
        authorization: 'token-1',
        'x-project-id': 'project-1',
        'x-auth-id': 'auth-1',
      },
    );
    expect(onUpdate).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('shows an error when update fails', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockUpdateKnowledgeDocumentSegment.mockImplementation(
      (...args: unknown[]) => {
        const callback = args[13] as Function;
        callback({ message: 'failed' }, null);
      },
    );

    render(
      <EditKnowledgeDocumentSegmentDialog
        segment={makeSegment() as any}
        onClose={jest.fn()}
        onUpdate={jest.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Update document' }));

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Error: Failed to update the segment. Please try again.',
    );
  });

  it('parses comma-separated entity values', () => {
    expect(parseSegmentEntityList(' Org A, Org B ,, ')).toEqual([
      'Org A',
      'Org B',
    ]);
  });
});

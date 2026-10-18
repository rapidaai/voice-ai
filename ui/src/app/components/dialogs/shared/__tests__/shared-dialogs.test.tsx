import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import {
  ColumnPreferencesDialog,
  ConfirmDeleteDialog,
  ConfirmDialog,
  CreateTagDialog,
  DisconnectDetailsDialog,
  UpdateDescriptionDialog,
} from '..';

const mockRapidaStore = {
  loading: false,
  showLoader: jest.fn(),
  hideLoader: jest.fn(),
};

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => mockRapidaStore,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  Checkbox: ({ labelText, checked, onChange, id }: any) => (
    <label htmlFor={id}>
      <input id={id} type="checkbox" checked={checked} onChange={onChange} />
      {labelText}
    </label>
  ),
  DangerButton: ({ children, isLoading, ...props }: any) => (
    <button data-kind="danger" {...props}>
      {children}
    </button>
  ),
  FormGroup: ({ children, legendText }: any) => (
    <fieldset>
      <legend>{legendText}</legend>
      {children}
    </fieldset>
  ),
  Modal: ({ open, children, onClose, danger, size }: any) =>
    open ? (
      <div data-danger={String(Boolean(danger))} data-size={size}>
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
    <button data-kind="primary" {...props}>
      {children}
    </button>
  ),
  SecondaryButton: ({ children, isLoading, ...props }: any) => (
    <button data-kind="secondary" {...props}>
      {children}
    </button>
  ),
  Stack: ({ children }: any) => <div>{children}</div>,
  TextArea: ({ labelText, onChange, value, ...props }: any) => (
    <label>
      {labelText}
      <textarea onChange={onChange} value={value} {...props} />
    </label>
  ),
  TextInput: ({ labelText, onChange, value, ...props }: any) => (
    <label>
      {labelText}
      <input onChange={onChange} value={value} {...props} />
    </label>
  ),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  EmptyState: ({ title, subtitle }: any) => (
    <section>
      <h3>{title}</h3>
      <p>{subtitle}</p>
    </section>
  ),
  Notification: ({ title, subtitle }: any) => (
    <div role="alert">
      {title}: {subtitle}
    </div>
  ),
}));

jest.mock('@/app/components/ui/feedback/error-message', () => ({
  ErrorMessage: ({ message }: any) =>
    message ? <div role="alert">{message}</div> : null,
}));

jest.mock('@/app/components/ui/feedback/notification', () => ({
  Notification: ({ title, subtitle }: any) => (
    <div role="alert">
      {title}: {subtitle}
    </div>
  ),
}));

jest.mock('@/app/components/ui/composites/tag-input', () => ({
  TagInput: ({ tags, addTag, removeTag, allTags }: any) => (
    <div>
      {tags.map((tag: string) => (
        <button key={tag} type="button" onClick={() => removeTag(tag)}>
          Remove {tag}
        </button>
      ))}
      {allTags.map((tag: string) => (
        <button key={tag} type="button" onClick={() => addTag(tag)}>
          Add {tag}
        </button>
      ))}
    </div>
  ),
}));

jest.mock('@carbon/react', () => ({
  RadioButton: ({ labelText, value, onSelect }: any) => (
    <button type="button" onClick={() => onSelect(value)}>
      {labelText}
    </button>
  ),
  RadioButtonGroup: ({ children, onChange }: any) => {
    const React = require('react');

    return (
      <div>
        {React.Children.map(children, (child: React.ReactElement) =>
          React.cloneElement(child, {
            onSelect: (value: string | number) => onChange(value),
          }),
        )}
      </div>
    );
  },
  Table: ({ children }: any) => <table>{children}</table>,
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableCell: ({ children }: any) => <td>{children}</td>,
  TableHead: ({ children }: any) => <thead>{children}</thead>,
  TableHeader: ({ children }: any) => <th>{children}</th>,
  TableRow: ({ children }: any) => <tr>{children}</tr>,
}));

jest.mock('@carbon/icons-react', () => ({
  Information: () => <svg data-testid="information-icon" />,
}));

describe('shared dialogs', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockRapidaStore.loading = false;
  });

  it('renders confirm dialogs and dispatches each action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onConfirm = jest.fn();
    const onCancel = jest.fn();
    const onClose = jest.fn();

    render(
      <ConfirmDialog
        showing
        type="info"
        title="Create version"
        content="Create this version."
        onConfirm={onConfirm}
        onCancel={onCancel}
        onClose={onClose}
      />,
    );

    expect(
      screen.getByRole('heading', { name: 'Create version' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Create this version.')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Confirm' }));
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    await user.click(screen.getByRole('button', { name: 'Modal close' }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('uses danger treatment for warning confirms', () => {
    render(
      <ConfirmDialog
        showing
        type="warning"
        title="Delete endpoint"
        content="This cannot be undone."
        confirmText="Delete"
        onConfirm={jest.fn()}
        onCancel={jest.fn()}
        onClose={jest.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: 'Delete' })).toHaveAttribute(
      'data-kind',
      'danger',
    );
    expect(screen.getByText('Warning')).toBeInTheDocument();
  });

  it('requires the object name before confirming a delete', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onConfirm = jest.fn();

    render(
      <ConfirmDeleteDialog
        showing
        title="Delete project"
        content="Delete this project."
        objectName="support"
        onConfirm={onConfirm}
        onCancel={jest.fn()}
        onClose={jest.fn()}
      />,
    );

    const deleteButton = screen.getByRole('button', { name: 'Delete' });
    expect(deleteButton).toBeDisabled();

    await user.click(screen.getByPlaceholderText('support'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('wrong'));
    expect(deleteButton).toBeDisabled();

    await user.click(screen.getByPlaceholderText('support'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('support'));
    expect(deleteButton).toBeEnabled();

    await user.click(deleteButton);
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('resets delete confirmation input when cancelled or closed', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onCancel = jest.fn();
    const onClose = jest.fn();

    render(
      <ConfirmDeleteDialog
        showing
        title="Delete project"
        content="Delete this project."
        objectName="support"
        onConfirm={jest.fn()}
        onCancel={onCancel}
        onClose={onClose}
      />,
    );

    const input = screen.getByPlaceholderText('support');
    await user.click(input);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('support'));
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(input).toHaveValue('');
    expect(onCancel).toHaveBeenCalledTimes(1);

    await user.click(input);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('support'));
    await user.click(screen.getByRole('button', { name: 'Modal close' }));
    expect(input).toHaveValue('');
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('saves column preferences and page size selections', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeColumns = jest.fn();
    const onChangePageSize = jest.fn();
    const setOpen = jest.fn();

    render(
      <ColumnPreferencesDialog
        open
        setOpen={setOpen}
        defaultPageSize={[10, 25]}
        columns={[
          { name: 'Name', key: 'name', visible: true },
          { name: 'Status', key: 'status', visible: false },
        ]}
        pageSize={10}
        onChangeColumns={onChangeColumns}
        onChangePageSize={onChangePageSize}
      />,
    );

    await user.click(screen.getByRole('button', { name: '25 Items' }));
    await user.click(screen.getByLabelText('Status'));
    await user.click(screen.getByRole('button', { name: 'Save Preference' }));

    expect(onChangePageSize).toHaveBeenCalledWith(25);
    expect(onChangeColumns).toHaveBeenCalledWith([
      { name: 'Name', key: 'name', visible: true },
      { name: 'Status', key: 'status', visible: true },
    ]);
    expect(setOpen).toHaveBeenCalledWith(false);
  });

  it('blocks column preferences when every column is hidden', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeColumns = jest.fn();

    render(
      <ColumnPreferencesDialog
        open
        setOpen={jest.fn()}
        defaultPageSize={[10]}
        columns={[{ name: 'Name', key: 'name', visible: true }]}
        pageSize={10}
        onChangeColumns={onChangeColumns}
        onChangePageSize={jest.fn()}
      />,
    );

    await user.click(screen.getByLabelText('Name'));
    await user.click(screen.getByRole('button', { name: 'Save Preference' }));

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Select at least one visible column',
    );
    expect(onChangeColumns).not.toHaveBeenCalled();
  });

  it('creates tags with selected values and closes on success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    const onCreateTag = jest.fn((_tags, _onError, onSuccess) => onSuccess({}));

    render(
      <CreateTagDialog
        modalOpen
        setModalOpen={setModalOpen}
        title="Edit tags"
        tags={['existing']}
        allTags={['support']}
        onCreateTag={onCreateTag}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Add support' }));
    await user.click(screen.getByRole('button', { name: 'Remove existing' }));
    await user.click(screen.getByRole('button', { name: 'Save tags' }));

    expect(mockRapidaStore.showLoader).toHaveBeenCalledWith('overlay');
    expect(mockRapidaStore.hideLoader).toHaveBeenCalledTimes(1);
    expect(onCreateTag).toHaveBeenCalledWith(
      ['support'],
      expect.any(Function),
      expect.any(Function),
    );
    expect(setModalOpen).toHaveBeenCalledWith(false);
  });

  it('shows create tag errors and keeps the dialog open', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    const onCreateTag = jest.fn((_tags, onError) => onError('Tag failed'));

    render(
      <CreateTagDialog
        modalOpen
        setModalOpen={setModalOpen}
        title="Edit tags"
        tags={[]}
        allTags={['support']}
        onCreateTag={onCreateTag}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Save tags' }));

    expect(screen.getByRole('alert')).toHaveTextContent('Error: Tag failed');
    expect(setModalOpen).not.toHaveBeenCalledWith(false);
  });

  it('updates descriptions and closes on success', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    const onUpdateDescription = jest.fn(
      (_name, _description, _onError, onSuccess) => onSuccess(),
    );

    render(
      <UpdateDescriptionDialog
        modalOpen
        setModalOpen={setModalOpen}
        title="Edit endpoint details"
        name="Endpoint"
        description="Initial description"
        onUpdateDescription={onUpdateDescription}
      />,
    );

    await user.click(screen.getByLabelText('Name'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Updated endpoint'));
    await user.click(screen.getByLabelText('Description'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Updated description'));
    await user.click(screen.getByRole('button', { name: 'Save changes' }));

    expect(onUpdateDescription).toHaveBeenCalledWith(
      'Updated endpoint',
      'Updated description',
      expect.any(Function),
      expect.any(Function),
    );
    expect(setModalOpen).toHaveBeenCalledWith(false);
  });

  it('shows update description errors', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onUpdateDescription = jest.fn((_name, _description, onError) =>
      onError('Update failed'),
    );

    render(
      <UpdateDescriptionDialog
        modalOpen
        setModalOpen={jest.fn()}
        onUpdateDescription={onUpdateDescription}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Save changes' }));

    expect(screen.getByRole('alert')).toHaveTextContent('Error: Update failed');
  });

  it('renders disconnect detail rows and empty state', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    const { rerender } = render(
      <DisconnectDetailsDialog
        modalOpen
        setModalOpen={setModalOpen}
        details={[{ label: 'Reason', value: 'Client disconnected' }]}
      />,
    );

    expect(screen.getByText('Reason')).toBeInTheDocument();
    expect(screen.getByText('Client disconnected')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Modal close' }));
    await user.click(screen.getByRole('button', { name: 'Header close' }));
    expect(setModalOpen).toHaveBeenCalledWith(false);
    expect(setModalOpen).toHaveBeenCalledTimes(2);

    rerender(
      <DisconnectDetailsDialog
        modalOpen
        setModalOpen={setModalOpen}
        details={[]}
      />,
    );

    expect(screen.getByText('No disconnect details')).toBeInTheDocument();
  });
});

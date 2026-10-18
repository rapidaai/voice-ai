import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ProjectRoleTable } from '../project-role-table';

jest.mock('@carbon/icons-react', () => ({
  Add: () => <svg data-testid="add-icon" />,
  TrashCan: () => <svg data-testid="trash-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Button: ({ iconDescription, onClick }: any) => (
    <button type="button" aria-label={iconDescription} onClick={onClick}>
      {iconDescription}
    </button>
  ),
  Dropdown: ({
    className,
    id,
    items,
    itemToString,
    label,
    onChange,
    selectedItem,
  }: any) => (
    <label htmlFor={id}>
      {label}
      <select
        className={className}
        id={id}
        value={selectedItem?.value || ''}
        onChange={event =>
          onChange({
            selectedItem:
              items.find((item: any) => item.value === event.target.value) ||
              null,
          })
        }
      >
        <option value="">{label}</option>
        {items.map((item: any) => (
          <option key={item.value} value={item.value}>
            {itemToString(item)}
          </option>
        ))}
      </select>
    </label>
  ),
  Table: ({ children, className }: any) => (
    <table className={className}>{children}</table>
  ),
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableCell: ({ children, className, colSpan }: any) => (
    <td className={className} colSpan={colSpan}>
      {children}
    </td>
  ),
  TableContainer: ({ children, className, title }: any) => (
    <section className={className}>
      {title ? <h2>{title}</h2> : null}
      {children}
    </section>
  ),
  TableHead: ({ children }: any) => <thead>{children}</thead>,
  TableHeader: ({ children, className }: any) => (
    <th className={className}>{children}</th>
  ),
  TableRow: ({ children }: any) => <tr>{children}</tr>,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  TertiaryButton: ({ children, onClick }: any) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
}));

const projectOptions = [
  { name: 'Support operations', value: 'project-1' },
  { name: 'Sales enablement', value: 'project-2' },
];

const roleOptions = [
  { name: 'Admin', value: 'admin' },
  { name: 'Reader', value: 'reader' },
];

describe('ProjectRoleTable', () => {
  it('renders an empty state and adds a default project row', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ProjectRoleTable
        rows={[]}
        onChange={onChange}
        projectOptions={projectOptions}
        roleOptions={roleOptions}
        defaultProjectId="project-2"
        title="Project roles"
        addButtonLabel="Add role"
      />,
    );

    expect(screen.getByText('Project roles (0)')).toBeInTheDocument();
    expect(screen.getByText(/No project roles yet/)).toHaveTextContent(
      'No project roles yet. Click Add role below.',
    );

    await user.click(screen.getByRole('button', { name: 'Add role' }));

    expect(onChange).toHaveBeenCalledWith([
      { projectId: 'project-2', projectRole: '' },
    ]);
  });

  it('updates project and role selections for an existing row', async () => {
    const user = userEvent.setup();
    const onChange = jest.fn();

    render(
      <ProjectRoleTable
        rows={[{ projectId: 'project-1', projectRole: 'admin' }]}
        onChange={onChange}
        projectOptions={projectOptions}
        roleOptions={roleOptions}
      />,
    );

    await user.selectOptions(
      screen.getByLabelText('Select project'),
      String('project-2'),
    );
    expect(onChange).toHaveBeenLastCalledWith([
      { projectId: 'project-2', projectRole: 'admin' },
    ]);

    await user.selectOptions(
      screen.getByLabelText('Select role'),
      String('reader'),
    );
    expect(onChange).toHaveBeenLastCalledWith([
      { projectId: 'project-1', projectRole: 'reader' },
    ]);
  });

  it('renders dropdown rows as a full-cell table form', () => {
    render(
      <ProjectRoleTable
        rows={[{ projectId: 'project-1', projectRole: 'admin' }]}
        onChange={jest.fn()}
        projectOptions={projectOptions}
        roleOptions={roleOptions}
      />,
    );

    expect(screen.getByRole('table').parentElement).toHaveClass(
      '![padding-block-start:0]',
      'border-border-subtle',
    );
    expect(screen.getByLabelText('Select project')).toHaveClass('w-full');
    expect(screen.getByLabelText('Select project').closest('td')).toHaveClass(
      '!p-0',
    );
    expect(screen.getByLabelText('Select role').closest('td')).toHaveClass(
      '!p-0',
    );
    expect(screen.getByRole('columnheader', { name: 'Actions' })).toHaveClass(
      '!w-10',
      '!text-center',
    );
    expect(
      screen.getByRole('button', { name: 'Remove project role' }).closest('td'),
    ).toHaveClass('!w-10', '!p-0', '!text-center');
  });

  it('removes rows when the remove column is visible', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ProjectRoleTable
        rows={[
          { projectId: 'project-1', projectRole: 'admin' },
          { projectId: 'project-2', projectRole: 'reader' },
        ]}
        onChange={onChange}
        projectOptions={projectOptions}
        roleOptions={roleOptions}
      />,
    );

    await user.click(
      screen.getAllByRole('button', {
        name: 'Remove project role',
      })[0],
    );

    expect(onChange).toHaveBeenCalledWith([
      { projectId: 'project-2', projectRole: 'reader' },
    ]);
  });

  it('supports fixed project rows without add or remove actions', () => {
    render(
      <ProjectRoleTable
        rows={[{ projectId: 'project-1', projectRole: 'reader' }]}
        onChange={jest.fn()}
        projectOptions={projectOptions}
        roleOptions={roleOptions}
        showAddButton={false}
        showRemoveColumn={false}
      />,
    );

    expect(
      screen.queryByRole('button', { name: 'Add project role' }),
    ).toBeNull();
    expect(
      screen.queryByRole('button', { name: 'Remove project role' }),
    ).toBeNull();
    expect(screen.queryByText('Actions')).toBeNull();
  });

  it('adds an empty row when no default project or options are available', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ProjectRoleTable
        rows={[]}
        onChange={onChange}
        projectOptions={[]}
        roleOptions={roleOptions}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Add project role' }));

    expect(onChange).toHaveBeenCalledWith([{ projectId: '', projectRole: '' }]);
  });
});

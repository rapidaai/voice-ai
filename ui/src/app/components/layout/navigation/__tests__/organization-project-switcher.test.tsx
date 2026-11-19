import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { OrganizationProjectSwitcher } from '@/app/components/layout/navigation/organization-project-switcher';

jest.mock('@carbon/react', () => {
  const React = jest.requireActual('react');
  const TreeContext = React.createContext(undefined);

  return {
    Button: React.forwardRef(
      (
        {
          children,
          hasIconOnly: _hasIconOnly,
          iconDescription: _iconDescription,
          kind: _kind,
          renderIcon: Icon,
          ...props
        }: any,
        ref: any,
      ) => (
        <button ref={ref} {...props}>
          {children}
          {Icon ? <Icon /> : null}
        </button>
      ),
    ),
    ButtonSkeleton: () => <div data-testid="context-switcher-skeleton" />,
    Popover: ({ children, open }: any) => (
      <div data-open={String(open)}>{children}</div>
    ),
    PopoverContent: ({ children }: any) => <div>{children}</div>,
    Search: ({ labelText, onChange, placeholder, value }: any) => (
      <input
        aria-label={labelText}
        placeholder={placeholder}
        type="search"
        value={value}
        onChange={onChange}
      />
    ),
    TreeView: ({ children, label, onSelect, selected }: any) => (
      <TreeContext.Provider value={{ onSelect, selected }}>
        <ul aria-label={label} role="tree">
          {children}
        </ul>
      </TreeContext.Provider>
    ),
    TreeNode: ({ children, id, label, value }: any) => {
      const context = React.useContext(TreeContext) as any;
      return (
        <li
          aria-selected={context.selected.includes(id)}
          id={id}
          role="treeitem"
        >
          <button
            type="button"
            onClick={event => context.onSelect(event, { id, label, value })}
          >
            {label}
          </button>
          {children ? <ul role="group">{children}</ul> : null}
        </li>
      );
    },
  };
});

jest.mock('@carbon/icons-react', () => ({
  ChevronDown: () => <span>open</span>,
}));

const organizations = [
  {
    id: 'organization-1',
    name: 'Acme Corporation',
    rolesList: [],
    projectsList: [
      { id: 'project-1', name: 'Support API', rolesList: [] },
      { id: 'project-2', name: 'Sales Assistant', rolesList: [] },
    ],
  },
  {
    id: 'organization-2',
    name: 'Northwind Group',
    rolesList: [],
    projectsList: [],
  },
] as any;

describe('OrganizationProjectSwitcher', () => {
  it('selects a project with its parent organization', async () => {
    const user = userEvent.setup();
    const onSelect = jest.fn();

    render(
      <OrganizationProjectSwitcher
        organizations={organizations}
        currentOrganization={organizations[0]}
        currentProject={organizations[0].projectsList[0]}
        onSelect={onSelect}
      />,
    );

    expect(
      screen.getByRole('button', {
        name: 'Change organization or project. Current context: Acme Corporation / Support API',
      }),
    ).toHaveClass('w-64!');
    await user.click(
      screen.getByRole('button', {
        name: 'Change organization or project. Current context: Acme Corporation / Support API',
      }),
    );
    expect(
      screen.getByRole('tree', { name: 'Organizations and projects' }),
    ).toBeInTheDocument();
    expect(
      document.getElementById('project-organization-1-project-1'),
    ).toHaveAttribute('aria-selected', 'true');

    await user.click(screen.getByRole('button', { name: 'Sales Assistant' }));

    expect(onSelect).toHaveBeenCalledWith(
      organizations[0],
      organizations[0].projectsList[1],
    );
    expect(
      screen.queryByRole('tree', { name: 'Organizations and projects' }),
    ).not.toBeInTheDocument();
  });

  it('filters by project while preserving its organization and supports organization-only selection', async () => {
    const user = userEvent.setup();
    const onSelect = jest.fn();

    render(
      <OrganizationProjectSwitcher
        organizations={organizations}
        currentOrganization={organizations[0]}
        currentProject={organizations[0].projectsList[0]}
        onSelect={onSelect}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: /Change organization or project/ }),
    );
    await user.type(
      screen.getByRole('searchbox', {
        name: 'Search organizations or projects',
      }),
      'sales',
    );

    expect(screen.getByText('Acme Corporation')).toBeInTheDocument();
    expect(screen.getByText('Sales Assistant')).toBeInTheDocument();
    expect(screen.queryByText('Support API')).not.toBeInTheDocument();
    expect(screen.queryByText('Northwind Group')).not.toBeInTheDocument();

    await user.clear(
      screen.getByRole('searchbox', {
        name: 'Search organizations or projects',
      }),
    );
    await user.click(screen.getByRole('button', { name: 'Northwind Group' }));

    expect(onSelect).toHaveBeenCalledWith(organizations[1]);
  });

  it('shows an empty result without changing context', async () => {
    const user = userEvent.setup();
    const onSelect = jest.fn();

    render(
      <OrganizationProjectSwitcher
        organizations={organizations}
        currentOrganization={organizations[0]}
        currentProject={organizations[0].projectsList[0]}
        onSelect={onSelect}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: /Change organization or project/ }),
    );
    await user.type(
      screen.getByRole('searchbox', {
        name: 'Search organizations or projects',
      }),
      'missing workspace',
    );

    expect(
      screen.getByText('No organizations or projects found.'),
    ).toBeVisible();
    expect(onSelect).not.toHaveBeenCalled();
  });
});

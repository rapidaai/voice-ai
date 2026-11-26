import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { OrganizationProjectSelector } from '../organization-project-selector';

const organizations = [
  {
    id: 'organization-1',
    name: 'Acme Corporation',
    projects: [
      { id: 'project-1', name: 'Customer Support' },
      { id: 'project-2', name: 'Sales Automation' },
    ],
  },
  {
    id: 'organization-2',
    name: 'Northwind Group',
    projects: [{ id: 'project-3', name: 'Internal Tools' }],
  },
  {
    id: 'organization-3',
    name: 'Contoso Research',
    projects: [],
  },
];

describe('OrganizationProjectSelector', () => {
  it('presents the selected context and nested organization projects', async () => {
    const user = userEvent.setup();

    render(
      <OrganizationProjectSelector
        organizations={organizations}
        selectedOrganizationId="organization-1"
        selectedProjectId="project-1"
        isCompact
        onChange={jest.fn()}
      />,
    );

    const trigger = screen.getByRole('button', {
      name: 'Change organization or project. Current organization: Acme Corporation. Current project: Customer Support.',
    });
    expect(trigger).toHaveTextContent('Acme Corporation');
    expect(trigger).toHaveTextContent('Customer Support');
    expect(trigger.parentElement).toHaveClass('w-12');

    await user.click(trigger);

    expect(trigger.closest('.cds--popover-container')).toHaveClass(
      'cds--popover--auto-align',
    );
    expect(trigger.closest('.cds--popover-container')).not.toHaveClass(
      'cds--popover--caret',
    );
    expect(
      document.querySelector('.cds--contained-list--md'),
    ).toBeInTheDocument();
    expect(screen.getByText('Switch context')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Search by name')).toBeInTheDocument();
    expect(screen.getAllByText('Acme Corporation')).toHaveLength(2);
    expect(screen.getByText('2 projects')).toBeInTheDocument();
    expect(screen.queryByText('Project')).not.toBeInTheDocument();
    const selectedIndicator = screen.getByLabelText('Selected project');
    const selectedProject = selectedIndicator.closest('button');
    expect(selectedProject).toBeInTheDocument();
    expect(selectedProject).toContainElement(selectedIndicator);
  });

  it('returns the parent organization with the selected project', async () => {
    const user = userEvent.setup();
    const onChange = jest.fn();

    render(
      <OrganizationProjectSelector
        organizations={organizations}
        selectedOrganizationId="organization-1"
        selectedProjectId="project-1"
        onChange={onChange}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: /Change organization or project/ }),
    );
    await user.click(screen.getByText('Sales Automation').closest('button')!);

    expect(onChange).toHaveBeenCalledWith(
      organizations[0],
      organizations[0].projects[1],
    );
    expect(screen.queryByText('Switch context')).not.toBeInTheDocument();
  });

  it('filters projects with their organization and supports organization-only context', async () => {
    const user = userEvent.setup();
    const onChange = jest.fn();

    render(
      <OrganizationProjectSelector
        organizations={organizations}
        selectedOrganizationId="organization-1"
        selectedProjectId="project-1"
        onChange={onChange}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: /Change organization or project/ }),
    );
    const search = screen.getByPlaceholderText('Search by name');
    await user.type(search, 'internal');

    expect(screen.getByText('Northwind Group')).toBeInTheDocument();
    expect(screen.getByText('Internal Tools')).toBeInTheDocument();
    expect(screen.queryByText('Sales Automation')).not.toBeInTheDocument();

    await user.clear(search);
    await user.click(screen.getByText('Organization only').closest('button')!);

    expect(onChange).toHaveBeenCalledWith(organizations[2], undefined);
  });
});

import React from 'react';
import { render, screen, within } from '@testing-library/react';
import '@testing-library/jest-dom';
import { ProjectUserGroupAvatar } from '../project-user-group-avatar';

describe('ProjectUserGroupAvatar', () => {
  it('renders member initials with an accessible group label', () => {
    render(
      <ProjectUserGroupAvatar
        label="Workspace collaborators"
        size="md"
        members={[{ name: 'Priyanka Iyer' }, { name: 'Amara Shah' }]}
      />,
    );

    const group = screen.getByRole('list', {
      name: 'Workspace collaborators',
    });

    expect(within(group).getAllByRole('listitem')).toHaveLength(2);
    expect(
      screen.getByRole('img', { name: 'Priyanka Iyer' }),
    ).toHaveTextContent('PI');
    expect(
      within(screen.getByRole('img', { name: 'Priyanka Iyer' })).getByText(
        'PI',
      ),
    ).toHaveClass('cds--user-avatar', 'cds--user-avatar--md');
    expect(screen.getByRole('img', { name: 'Amara Shah' })).toHaveTextContent(
      'AS',
    );
  });

  it('renders an empty member list without placeholder content', () => {
    render(<ProjectUserGroupAvatar members={[]} />);

    expect(
      screen.getByRole('list', { name: 'Project members' }),
    ).toBeEmptyDOMElement();
  });
});

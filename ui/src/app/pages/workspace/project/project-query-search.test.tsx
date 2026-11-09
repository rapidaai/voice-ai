import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { ProjectQuerySearch } from './project-query-search';

function ProjectQuerySearchHarness() {
  const [value, setValue] = useState('');

  return (
    <ProjectQuerySearch onApply={jest.fn()} onChange={setValue} value={value} />
  );
}

describe('ProjectQuerySearch', () => {
  it('offers the project roles used by organization membership', async () => {
    const user = userEvent.setup();

    render(<ProjectQuerySearchHarness />);

    await user.click(
      screen.getByPlaceholderText(
        'Search for name, role, collaborator, status',
      ),
    );
    await user.click(screen.getByRole('button', { name: 'role string' }));

    expect(
      screen.getByRole('button', { name: 'super admin string' }),
    ).toBeVisible();
    expect(screen.getByRole('button', { name: 'admin string' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'writer string' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'reader string' })).toBeVisible();
    expect(screen.queryByRole('button', { name: 'owner string' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'member string' })).toBeNull();
  });

  it('only offers statuses returned by the project listing', async () => {
    const user = userEvent.setup();

    render(<ProjectQuerySearchHarness />);

    await user.click(
      screen.getByPlaceholderText(
        'Search for name, role, collaborator, status',
      ),
    );
    await user.click(screen.getByRole('button', { name: 'status string' }));

    expect(screen.getByRole('button', { name: 'active string' })).toBeVisible();
    expect(
      screen.queryByRole('button', { name: 'archived string' }),
    ).toBeNull();
  });
});

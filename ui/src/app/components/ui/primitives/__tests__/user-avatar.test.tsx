import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { UserAvatar } from '../user-avatar';

describe('UserAvatar', () => {
  it('renders IBM-compatible initials, size, and background classes', () => {
    render(
      <UserAvatar
        aria-label="Priyanka Iyer"
        backgroundColor="order-5-purple"
        name="Priyanka Iyer"
        role="img"
        size="sm"
      />,
    );

    expect(screen.getByRole('img', { name: 'Priyanka Iyer' })).toHaveClass(
      'cds--user-avatar',
      'cds--user-avatar--order-5-purple',
      'cds--user-avatar--sm',
    );
    expect(
      screen.getByRole('img', { name: 'Priyanka Iyer' }),
    ).toHaveTextContent('PI');
  });

  it('renders the default user icon when no identity is provided', () => {
    render(<UserAvatar aria-label="User" role="img" />);

    expect(
      screen.getByRole('img', { name: 'User' }).querySelector('svg'),
    ).not.toBeNull();
  });

  it('renders an accessible profile image instead of initials', () => {
    render(
      <UserAvatar
        image="/profiles/priyanka.png"
        imageDescription="Priyanka Iyer"
        name="Priyanka Iyer"
      />,
    );

    expect(screen.getByAltText('Priyanka Iyer')).toHaveAttribute(
      'src',
      '/profiles/priyanka.png',
    );
    expect(screen.queryByText('PI')).not.toBeInTheDocument();
  });
});

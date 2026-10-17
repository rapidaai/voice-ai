import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { Aside } from '../aside';

let mockOpen = true;
const mockSetOpen = jest.fn();

jest.mock('@/context/sidebar-context', () => ({
  useSidebar: () => ({
    open: mockOpen,
    setOpen: mockSetOpen,
  }),
}));

describe('Aside', () => {
  beforeEach(() => {
    mockOpen = true;
    mockSetOpen.mockClear();
  });

  it('renders the expanded semantic aside and forwards attributes', () => {
    render(
      <Aside
        aria-label="Primary navigation"
        className="custom-aside"
        data-testid="aside"
      >
        Navigation
      </Aside>,
    );

    expect(screen.getByRole('complementary')).toHaveAttribute(
      'aria-label',
      'Primary navigation',
    );
    expect(screen.getByTestId('aside')).toHaveClass(
      'custom-aside',
      'bg-shell',
      'border-border-subtle',
      'w-64',
      'absolute',
      'md:relative',
      'overflow-x-hidden',
    );
    expect(screen.getByTestId('aside')).toHaveTextContent('Navigation');
  });

  it('renders the collapsed rail width when the sidebar is closed', () => {
    mockOpen = false;

    render(
      <Aside aria-label="Primary navigation" data-testid="aside">
        Navigation
      </Aside>,
    );

    expect(screen.getByTestId('aside')).toHaveClass('relative', 'w-12');
    expect(screen.getByTestId('aside')).not.toHaveClass('w-64');
  });

  it('composes caller hover handlers with sidebar open state changes', () => {
    const onMouseEnter = jest.fn();
    const onMouseLeave = jest.fn();

    render(
      <Aside
        aria-label="Primary navigation"
        data-testid="aside"
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
      >
        Navigation
      </Aside>,
    );

    fireEvent.mouseEnter(screen.getByTestId('aside'));
    fireEvent.mouseLeave(screen.getByTestId('aside'));

    expect(onMouseEnter).toHaveBeenCalledTimes(1);
    expect(onMouseLeave).toHaveBeenCalledTimes(1);
    expect(mockSetOpen).toHaveBeenNthCalledWith(1, true);
    expect(mockSetOpen).toHaveBeenNthCalledWith(2, false);
  });
});

import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { RightSideModal } from '../right-side-modal';

const mockAnimate = jest.fn(() => Promise.resolve());
const mockDragStart = jest.fn();
const mockMotionValueGet = jest.fn(() => 0);

jest.mock('react-use-measure', () => ({
  __esModule: true,
  default: () => [jest.fn(), { width: 320 }],
}));

jest.mock('framer-motion', () => {
  const React = require('react');

  const MotionDiv = React.forwardRef(
    (
      {
        children,
        initial,
        animate,
        transition,
        drag,
        dragControls,
        dragListener,
        dragConstraints,
        dragElastic,
        style,
        ...props
      }: any,
      ref: any,
    ) => (
      <div ref={ref} {...props}>
        {children}
      </div>
    ),
  );
  MotionDiv.displayName = 'MotionDiv';

  return {
    motion: {
      div: MotionDiv,
    },
    useAnimate: () => [
      {
        current: null,
      },
      mockAnimate,
    ],
    useDragControls: () => ({
      start: mockDragStart,
    }),
    useMotionValue: () => ({
      get: mockMotionValueGet,
    }),
  };
});

jest.mock('@/app/components/ui/primitives/modal', () => ({
  ModalHeader: ({ label, title, onClose }: any) => (
    <header>
      <span>{label}</span>
      <h2>{title}</h2>
      <button type="button" onClick={onClose}>
        Header close
      </button>
    </header>
  ),
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  IconOnlyButton: ({
    iconDescription,
    kind,
    renderIcon: Icon,
    ...props
  }: any) => (
    <button
      type="button"
      aria-label={iconDescription}
      data-carbon-icon-button-kind={kind}
      {...props}
    >
      {Icon ? <Icon /> : null}
    </button>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  Close: () => <svg data-testid="close-icon" />,
}));

describe('RightSideModal', () => {
  beforeEach(() => {
    mockAnimate.mockClear();
    mockDragStart.mockClear();
    mockMotionValueGet.mockClear();
  });

  it('does not render drawer content when closed', () => {
    render(
      <RightSideModal modalOpen={false} setModalOpen={jest.fn()}>
        Drawer content
      </RightSideModal>,
    );

    expect(screen.queryByText('Drawer content')).not.toBeInTheDocument();
  });

  it('closes no-title drawers from the Carbon icon action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();

    render(
      <RightSideModal
        modalOpen
        setModalOpen={setModalOpen}
        className="custom-drawer"
        data-testid="drawer"
      >
        Drawer content
      </RightSideModal>,
    );

    expect(screen.getByRole('dialog', { name: 'Details' })).toHaveClass(
      'custom-drawer',
      'bg-layer',
      'text-foreground',
    );
    expect(screen.getByTestId('drawer')).toHaveAttribute('aria-modal', 'true');

    const closeButton = screen.getByRole('button', { name: 'Close' });
    expect(closeButton).toHaveAttribute(
      'data-carbon-icon-button-kind',
      'ghost',
    );
    expect(screen.getByTestId('close-icon')).toBeInTheDocument();

    await user.click(closeButton);

    await waitFor(() => expect(setModalOpen).toHaveBeenCalledWith(false));
  });

  it('renders title and label through the modal header', () => {
    render(
      <RightSideModal
        modalOpen
        setModalOpen={jest.fn()}
        title="Trace details"
        label="Endpoint"
      >
        Drawer content
      </RightSideModal>,
    );

    expect(
      screen.getByRole('dialog', { name: 'Trace details' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Endpoint')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Trace details' }),
    ).toBeInTheDocument();
  });

  it('starts drawer dragging from the resize handle', async () => {
    const user = userEvent.setup();
    render(
      <RightSideModal modalOpen setModalOpen={jest.fn()}>
        Drawer content
      </RightSideModal>,
    );

    await user.pointer({
      keys: '[MouseLeft>]',
      target: screen.getByRole('button', { name: 'Resize drawer' }),
    });

    expect(mockDragStart).toHaveBeenCalledTimes(1);
  });

  it('closes the drawer after dragging past the close threshold', async () => {
    const setModalOpen = jest.fn();
    mockMotionValueGet.mockReturnValue(120);

    render(
      <RightSideModal modalOpen setModalOpen={setModalOpen}>
        Drawer content
      </RightSideModal>,
    );

    act(() => {
      screen
        .getByRole('dialog')
        .dispatchEvent(new Event('dragend', { bubbles: true }));
    });

    await waitFor(() => expect(setModalOpen).toHaveBeenCalledWith(false));
  });

  it('keeps the drawer open when drag distance is below the close threshold', () => {
    const setModalOpen = jest.fn();
    mockMotionValueGet.mockReturnValue(20);

    render(
      <RightSideModal modalOpen setModalOpen={setModalOpen}>
        Drawer content
      </RightSideModal>,
    );

    act(() => {
      screen
        .getByRole('dialog')
        .dispatchEvent(new Event('dragend', { bubbles: true }));
    });

    expect(setModalOpen).not.toHaveBeenCalled();
  });
});

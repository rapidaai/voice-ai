import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { HowItWorksDialog } from '@/app/components/dialogs/shared/how-it-works-modal';

jest.mock('@/app/components/ui/primitives/modal', () => ({
  Modal: ({ open, children, containerClassName, onClose }: any) =>
    open ? (
      <div data-container-class={containerClassName} data-testid="modal">
        <button type="button" onClick={onClose}>
          Modal close
        </button>
        {children}
      </div>
    ) : null,
  ModalHeader: ({ title, onClose }: any) => (
    <header>
      <h2>{title}</h2>
      <button type="button" onClick={onClose}>
        Header close
      </button>
    </header>
  ),
  ModalBody: ({ children }: any) => <div>{children}</div>,
  ModalFooter: ({ children }: any) => <footer>{children}</footer>,
}));

jest.mock('@carbon/icons-react', () => ({
  Checkmark: () => <svg data-testid="checkmark-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Button: ({ children, kind, renderIcon: Icon, ...props }: any) => (
    <button data-carbon-button-kind={kind} {...props}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
}));

describe('HowItWorksDialog', () => {
  const steps = [
    {
      title: 'Connect source',
      icon: <span data-testid="step-icon" />,
      description: 'Pick the source data.',
    },
  ];

  it('renders the configured title and steps', () => {
    render(
      <HowItWorksDialog
        modalOpen
        setModalOpen={jest.fn()}
        title="Knowledge flow"
        steps={steps}
      />,
    );

    expect(
      screen.getByRole('heading', { name: 'Knowledge flow' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Connect source')).toBeInTheDocument();
    expect(screen.getByText('Pick the source data.')).toBeInTheDocument();
  });

  it('uses a Carbon primary button for the close action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    render(
      <HowItWorksDialog modalOpen setModalOpen={setModalOpen} steps={steps} />,
    );

    const closeButton = screen.getByRole('button', { name: 'Got it' });
    expect(closeButton).toHaveAttribute('data-carbon-button-kind', 'primary');
    expect(screen.getByTestId('checkmark-icon')).toBeInTheDocument();

    await user.click(closeButton);
    expect(setModalOpen).toHaveBeenCalledWith(false);
  });

  it('uses the default title and forwards container class to the modal', () => {
    render(
      <HowItWorksDialog
        modalOpen
        setModalOpen={jest.fn()}
        className="custom-modal"
        steps={steps}
      />,
    );

    expect(
      screen.getByRole('heading', { name: 'How it works' }),
    ).toBeInTheDocument();
    expect(screen.getByTestId('modal')).toHaveAttribute(
      'data-container-class',
      'custom-modal',
    );
  });

  it('closes from the modal and header close actions', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();

    render(
      <HowItWorksDialog modalOpen setModalOpen={setModalOpen} steps={steps} />,
    );

    await user.click(screen.getByRole('button', { name: 'Modal close' }));
    await user.click(screen.getByRole('button', { name: 'Header close' }));

    expect(setModalOpen).toHaveBeenCalledWith(false);
    expect(setModalOpen).toHaveBeenCalledTimes(2);
  });
});

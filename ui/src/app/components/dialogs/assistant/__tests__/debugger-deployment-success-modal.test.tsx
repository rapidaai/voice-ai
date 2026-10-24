import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { DebuggerDeploymentSuccessDialog } from '../debugger-deployment-success-modal';

jest.mock('@/app/components/ui/primitives/modal', () => ({
  Modal: ({ open, children }: any) => (open ? <div>{children}</div> : null),
  ModalHeader: ({ title }: any) => <h2>{title}</h2>,
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalFooter: ({ children }: any) => <footer>{children}</footer>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({ children, renderIcon: Icon, ...props }: any) => (
    <button {...props}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  Launch: () => <svg data-testid="preview-icon" />,
}));

describe('DebuggerDeploymentSuccessDialog', () => {
  const originalOpen = window.open;

  beforeEach(() => {
    window.open = jest.fn();
  });

  afterEach(() => {
    window.open = originalOpen;
  });

  it('closes from the secondary action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();

    render(
      <DebuggerDeploymentSuccessDialog
        modalOpen
        setModalOpen={setModalOpen}
        assistantId="assistant-1"
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Close' }));

    expect(setModalOpen).toHaveBeenCalledWith(false);
  });

  it('opens the assistant preview with the standard action icon', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <DebuggerDeploymentSuccessDialog
        modalOpen
        setModalOpen={jest.fn()}
        assistantId="assistant-1"
      />,
    );

    const previewButton = screen.getByRole('button', {
      name: 'Preview assistant',
    });
    expect(screen.getByTestId('preview-icon')).toBeInTheDocument();

    await user.click(previewButton);

    expect(window.open).toHaveBeenCalledWith(
      '/deployment/assistant/assistant-1/playground/chat',
      '_blank',
    );
  });
});

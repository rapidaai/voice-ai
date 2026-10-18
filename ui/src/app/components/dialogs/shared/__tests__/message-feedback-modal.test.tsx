import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { MessageFeedbackDialog } from '../message-feedback-modal';

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
  Checkmark: () => <svg data-testid="submit-icon" />,
}));

describe('MessageFeedbackDialog', () => {
  it('keeps submit disabled until feedback has text', () => {
    render(
      <MessageFeedbackDialog
        modalOpen
        setModalOpen={jest.fn()}
        onSubmitFeedback={jest.fn()}
      />,
    );

    expect(
      screen.getByRole('button', { name: 'Submit feedback' }),
    ).toBeDisabled();
  });

  it('submits feedback with the standard action icon', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    const onSubmitFeedback = jest.fn();

    render(
      <MessageFeedbackDialog
        modalOpen
        setModalOpen={setModalOpen}
        onSubmitFeedback={onSubmitFeedback}
      />,
    );

    await user.click(screen.getByPlaceholderText('Your feedback...'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Needs better source detail'));

    const submitButton = screen.getByRole('button', {
      name: 'Submit feedback',
    });
    expect(submitButton).toBeEnabled();
    expect(screen.getByTestId('submit-icon')).toBeInTheDocument();

    await user.click(submitButton);

    expect(setModalOpen).toHaveBeenCalledWith(false);
    expect(onSubmitFeedback).toHaveBeenCalledWith('Needs better source detail');
  });
});

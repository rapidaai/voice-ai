import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ConfigureAssistantTemplateDialog } from '../configure-assistant-template-modal';

jest.mock('@carbon/react', () => {
  const React = require('react');

  return {
    ContentSwitcher: ({ children, onChange }: any) => (
      <div>
        {React.Children.map(children, child =>
          React.cloneElement(child, {
            onClick: () => onChange({ name: child.props.name }),
          }),
        )}
      </div>
    ),
    SelectableTile: ({ children, onClick, selected }: any) => (
      <button
        type="button"
        aria-pressed={selected}
        data-selected={String(Boolean(selected))}
        onClick={onClick}
      >
        {children}
      </button>
    ),
    Switch: ({ name, onClick, text }: any) => (
      <button type="button" name={name} onClick={onClick}>
        {text}
      </button>
    ),
    Tag: ({ children }: any) => <span>{children}</span>,
  };
});

jest.mock('@/app/components/ui/primitives', () => ({
  Modal: ({ children, open }: any) =>
    open ? <section>{children}</section> : null,
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalFooter: ({ children }: any) => <footer>{children}</footer>,
  ModalHeader: ({ label, title }: any) => (
    <header>
      <p>{label}</p>
      <h2>{title}</h2>
    </header>
  ),
  PrimaryButton: ({ children, disabled, onClick }: any) => (
    <button type="button" disabled={disabled} onClick={onClick}>
      {children}
    </button>
  ),
  SecondaryButton: ({ children, onClick }: any) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
}));

describe('ConfigureAssistantTemplateDialog', () => {
  const selectCustomerSupportTemplate = async () => {
    const user = userEvent.setup();
    const heading = screen.getByRole('heading', {
      name: 'Customer Support Agent',
    });
    const tile = heading.closest('button');
    if (!tile) throw new Error('Customer Support Agent tile was not found');
    await user.click(tile);
  };

  it('selects an assistant template and returns it', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onSelectTemplate = jest.fn();
    const setModalOpen = jest.fn();

    render(
      <ConfigureAssistantTemplateDialog
        modalOpen
        setModalOpen={setModalOpen}
        onSelectTemplate={onSelectTemplate}
      />,
    );

    expect(screen.getByRole('button', { name: 'Use template' })).toBeDisabled();

    await selectCustomerSupportTemplate();
    await user.click(screen.getByRole('button', { name: 'Use template' }));

    expect(onSelectTemplate).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Customer Support Agent' }),
    );
    expect(setModalOpen).toHaveBeenCalledWith(false);
  });

  it('closes after selecting a template when no selection callback is provided', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();

    render(
      <ConfigureAssistantTemplateDialog
        modalOpen
        setModalOpen={setModalOpen}
      />,
    );

    await selectCustomerSupportTemplate();
    await user.click(screen.getByRole('button', { name: 'Use template' }));

    expect(setModalOpen).toHaveBeenCalledWith(false);
  });

  it('filters templates by category and clears the active selection', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onSelectTemplate = jest.fn();

    render(
      <ConfigureAssistantTemplateDialog
        modalOpen
        setModalOpen={jest.fn()}
        onSelectTemplate={onSelectTemplate}
      />,
    );

    await selectCustomerSupportTemplate();
    expect(screen.getByText('Selected:')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Sales' }));

    expect(
      screen.queryByText('Customer Support Agent'),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Sales Development Representative/ }),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Use template' })).toBeDisabled();
  });

  it('closes without selecting a template', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onSelectTemplate = jest.fn();
    const setModalOpen = jest.fn();

    render(
      <ConfigureAssistantTemplateDialog
        modalOpen
        setModalOpen={setModalOpen}
        onSelectTemplate={onSelectTemplate}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onSelectTemplate).not.toHaveBeenCalled();
    expect(setModalOpen).toHaveBeenCalledWith(false);
  });
});

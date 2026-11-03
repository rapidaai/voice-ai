import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { AssistantWebwidgetDeploymentDialog } from '../assistant-instruction-modal';

jest.mock('@/app/components/dialogs/shared', () => ({
  RightSideModal: ({
    children,
    className,
    label,
    modalOpen,
    setModalOpen,
    title,
  }: any) =>
    modalOpen ? (
      <aside className={className} data-testid="widget-instruction-drawer">
        <span>{label}</span>
        <h2>{title}</h2>
        <button type="button" onClick={() => setModalOpen(false)}>
          Close
        </button>
        {children}
      </aside>
    ) : null,
}));

jest.mock(
  '@/app/components/domain/integration-document/widget-integration',
  () => ({
    WidgetIntegration: ({ assistantId }: any) => (
      <section>Widget integration for {assistantId}</section>
    ),
  }),
);

describe('AssistantWebwidgetDeploymentDialog', () => {
  it('uses the shared responsive deployment drawer and integration document', () => {
    render(
      <AssistantWebwidgetDeploymentDialog
        modalOpen
        setModalOpen={jest.fn()}
        assistantId="assistant-123"
      />,
    );

    expect(screen.getByTestId('widget-instruction-drawer')).toHaveClass(
      'w-[580px]',
      'max-w-[calc(100vw-2rem)]',
    );
    expect(screen.getByText('Web Widget Deployment')).toBeInTheDocument();
    expect(screen.getByText('Deployment completed')).toBeInTheDocument();
    expect(
      screen.getByText('Widget integration for assistant-123'),
    ).toBeInTheDocument();
  });

  it('closes through the platform drawer action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();
    render(
      <AssistantWebwidgetDeploymentDialog
        modalOpen
        setModalOpen={setModalOpen}
        assistantId="assistant-123"
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Close' }));
    expect(setModalOpen).toHaveBeenCalledWith(false);
  });
});

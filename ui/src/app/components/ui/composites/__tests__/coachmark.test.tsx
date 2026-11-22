import React, { createRef, useRef, useState } from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Coachmark } from '../coachmark';
import { CoachmarkContent } from '../coachmark-content';

jest.mock(
  '@carbon/ibm-products/lib/components/Coachmark/next/Coachmark/Coachmark',
  () => {
    const MockCoachmark = ({ align, children, className, onClose }: any) => (
      <div className={className} data-align={align}>
        {children}
        <button
          type="button"
          onClick={() => {
            onClose();
            onClose();
          }}
        >
          Carbon close
        </button>
      </div>
    );
    MockCoachmark.Content = ({ children, className, ...props }: any) => (
      <section className={className} {...props}>
        {children}
      </section>
    );
    MockCoachmark.ContentHeader = () => null;
    MockCoachmark.ContentBody = ({ children, className }: any) => (
      <div className={className}>{children}</div>
    );
    return { Coachmark: MockCoachmark };
  },
);

function FocusRestorationHarness({ onDismiss }: { onDismiss: () => void }) {
  const [open, setOpen] = useState(true);
  const launcherButtonRef = useRef<HTMLButtonElement>(null);
  const dismiss = () => {
    onDismiss();
    setOpen(false);
  };

  if (!open) {
    return (
      <button ref={launcherButtonRef} type="button">
        Target
      </button>
    );
  }

  return (
    <CoachmarkContent
      actionLabel="Done"
      align="bottom"
      description="Description"
      launcherButtonRef={launcherButtonRef}
      onAction={dismiss}
      onClose={dismiss}
      title="Title"
    >
      <button ref={launcherButtonRef} type="button">
        Target
      </button>
    </CoachmarkContent>
  );
}

describe('Coachmark', () => {
  it('renders only its target while closed', () => {
    render(
      <Coachmark
        actionLabel="Next"
        align="bottom-end"
        description="Description"
        launcherButtonRef={createRef<HTMLButtonElement>()}
        onAction={jest.fn()}
        onClose={jest.fn()}
        open={false}
        title="Title"
      >
        <button type="button">Target</button>
      </Coachmark>,
    );

    expect(screen.getByRole('button', { name: 'Target' })).toBeVisible();
    expect(screen.queryByRole('region')).not.toBeInTheDocument();
  });

  it('uses the Carbon content layout without overriding its padding', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onAction = jest.fn();
    const onClose = jest.fn();
    const launcherButtonRef = createRef<HTMLButtonElement>();

    render(
      <CoachmarkContent
        actionLabel="Done"
        align="top-start"
        description="Collapse or expand the navigation."
        launcherButtonRef={launcherButtonRef}
        onAction={onAction}
        onClose={onClose}
        title="Control the navigation"
      >
        <button ref={launcherButtonRef} type="button">
          Collapse sidebar
        </button>
      </CoachmarkContent>,
    );

    const region = await screen.findByRole('region', {
      name: 'Control the navigation coachmark',
    });
    expect(region).toHaveClass('app-coachmark__content');
    expect(region.parentElement).not.toHaveClass('inline-flex', 'h-full');
    expect(region.parentElement).toHaveAttribute('data-align', 'top-start');
    expect(screen.getByRole('heading')).not.toHaveAttribute('class');
    expect(
      screen.getByText('Collapse or expand the navigation.'),
    ).not.toHaveAttribute('class');
    expect(screen.getByRole('heading').parentElement).not.toHaveClass(
      'gap-2',
      'pb-4',
    );
    expect(screen.getByRole('button', { name: 'Done' })).toHaveClass(
      'app-coachmark__action',
    );
    expect(screen.getByRole('button', { name: 'Done' })).not.toHaveClass(
      'my-4',
      'self-end',
    );

    await user.click(screen.getByRole('button', { name: 'Done' }));
    await user.click(screen.getByRole('button', { name: 'Carbon close' }));
    expect(onAction).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it.each([
    ['action', 'Done'],
    ['close control', 'Carbon close'],
  ])('restores launcher focus after the %s dismisses it', async (_, label) => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onDismiss = jest.fn();

    render(<FocusRestorationHarness onDismiss={onDismiss} />);
    await user.click(screen.getByRole('button', { name: label }));

    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Target' })).toHaveFocus(),
    );
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });
});

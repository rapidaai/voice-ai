import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { AssistantSettingsShellHeader } from '../settings-shell-header';

describe('AssistantSettingsShellHeader', () => {
  it('uses the shared shell header surface and preserves its actions', async () => {
    const user = userEvent.setup();
    const onReload = jest.fn();
    const onPrimaryAction = jest.fn();

    render(
      <AssistantSettingsShellHeader
        title="Authentication"
        href="/deployment/assistant/assistant-1/configure-authentication"
        onReload={onReload}
        reloadLabel="Reload authentication"
        primaryAction={{
          label: 'Configure authentication',
          onClick: onPrimaryAction,
        }}
      />,
    );

    const heading = screen.getByRole('heading', {
      name: 'Authentication',
      level: 1,
    });
    const header = heading.closest('[aria-label="Authentication page header"]');

    expect(header).toHaveClass(
      'h-12',
      'shrink-0',
      'border-b',
      'border-border-subtle',
      'bg-shell',
    );
    expect(header).not.toHaveClass('cds--header');
    expect(heading.closest('a')).toHaveAttribute(
      'href',
      '/deployment/assistant/assistant-1/configure-authentication',
    );

    await user.click(
      within(header as HTMLElement).getByRole('button', {
        name: 'Reload authentication',
      }),
    );
    await user.click(
      within(header as HTMLElement).getByRole('button', {
        name: 'Configure authentication',
      }),
    );

    expect(onReload).toHaveBeenCalledTimes(1);
    expect(onPrimaryAction).toHaveBeenCalledTimes(1);
  });

  it('renders a heading without an empty actions container', () => {
    render(
      <AssistantSettingsShellHeader
        title="General Settings"
        href="/deployment/assistant/assistant-1/edit-assistant"
      />,
    );

    const heading = screen.getByRole('heading', {
      name: 'General Settings',
      level: 1,
    });
    const header = heading.closest(
      '[aria-label="General Settings page header"]',
    );

    expect(header).toBeInTheDocument();
    expect(within(header as HTMLElement).queryByRole('button')).toBeNull();
  });
});

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { Switch, SwitchWithLabel } from '../switch';

jest.mock('@carbon/react', () => ({
  Toggle: ({ id, labelText, hideLabel, name, onToggle, toggled }: any) => (
    <button
      type="button"
      id={id}
      name={name}
      role="switch"
      aria-checked={toggled}
      aria-label={hideLabel ? labelText : undefined}
      data-design-system-toggle
      onClick={() => onToggle(!toggled)}
    >
      {!hideLabel ? labelText : null}
    </button>
  ),
}));

describe('Switch', () => {
  it('renders the Carbon toggle with the current enabled state', () => {
    render(
      <Switch
        id="reranking"
        name="reranking"
        label="Enable reranking"
        enable
        setEnable={jest.fn()}
      />,
    );

    const toggle = screen.getByRole('switch', { name: 'Enable reranking' });
    expect(toggle).toHaveAttribute('data-design-system-toggle');
    expect(toggle).toHaveAttribute('aria-checked', 'true');
    expect(toggle).toHaveAttribute('name', 'reranking');
  });

  it('keeps the existing boolean setter contract', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setEnable = jest.fn();

    render(
      <Switch
        id="reranking"
        label="Enable reranking"
        enable={false}
        setEnable={setEnable}
      />,
    );

    await user.click(screen.getByRole('switch', { name: 'Enable reranking' }));

    expect(setEnable).toHaveBeenCalledWith(true);
  });

  it('renders the labeled row with a Carbon toggle action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setEnable = jest.fn();

    render(
      <SwitchWithLabel
        id="grounding"
        label="Enable grounding check"
        enable={false}
        setEnable={setEnable}
      />,
    );

    expect(screen.getByText('Enable grounding check')).toBeInTheDocument();

    await user.click(
      screen.getByRole('switch', { name: 'Enable grounding check' }),
    );

    expect(setEnable).toHaveBeenCalledWith(true);
  });
});

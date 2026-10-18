import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { Slider } from '../slider';
import userEvent from '@testing-library/user-event';
import { setRangeValue } from '@/test-utils/set-range-value';

jest.mock('@carbon/react', () => {
  const React = require('react');

  return {
    Slider: ({
      ariaLabelInput,
      className,
      hideLabel,
      id,
      inputType,
      labelText,
      max,
      min,
      name,
      onChange,
      step,
      value,
    }: any) => (
      <label>
        <span className={hideLabel ? 'cds--visually-hidden' : undefined}>
          {labelText}
        </span>
        <input
          aria-label={ariaLabelInput}
          className={className}
          data-design-system-slider
          data-input-type={inputType}
          id={id}
          max={max}
          min={min}
          name={name}
          onChange={event =>
            onChange({ value: Number(event.currentTarget.value) })
          }
          step={step}
          type="range"
          value={value}
        />
      </label>
    ),
  };
});

describe('Slider', () => {
  it('renders Carbon Slider with converted numeric values and onSlide support', async () => {
    const user = userEvent.setup();
    const onSlide = jest.fn();

    render(
      <Slider
        name="volume"
        min="0"
        max="1"
        step="0.01"
        value="0.25"
        onSlide={onSlide}
        className="max-w-48"
        type="range"
      />,
    );

    const slider = screen.getByRole('slider', { name: 'volume' });

    expect(slider).toHaveAttribute('data-design-system-slider');
    expect(slider).toHaveAttribute('id', 'volume');
    expect(slider).toHaveAttribute('name', 'volume');
    expect(slider).toHaveAttribute('min', '0');
    expect(slider).toHaveAttribute('max', '1');
    expect(slider).toHaveAttribute('step', '0.01');
    expect(slider).toHaveAttribute('data-input-type', 'range');
    expect(slider).toHaveValue('0.25');
    expect(slider).toHaveClass('w-full');
    expect(slider).toHaveClass('max-w-48');

    await user.click(slider);
    setRangeValue(slider as HTMLInputElement, 0.5);

    expect(onSlide).toHaveBeenCalledWith(0.5);
  });

  it('keeps Carbon-style onChange and visible labels', async () => {
    const user = userEvent.setup();
    const onChange = jest.fn();

    render(
      <Slider
        id="transfer-delay"
        labelText="Transfer Delay"
        hideLabel={false}
        min={500}
        max={3000}
        step={50}
        value={1000}
        onChange={onChange}
      />,
    );

    const slider = screen.getByRole('slider', { name: 'Transfer Delay' });

    expect(slider).toHaveAttribute('id', 'transfer-delay');
    expect(screen.getByText('Transfer Delay')).toBeVisible();

    await user.click(slider);
    setRangeValue(slider as HTMLInputElement, 1500);

    expect(onChange).toHaveBeenCalledWith({ value: 1500 });
  });

  it('generates an id and defaults the value to min', () => {
    render(<Slider aria-label="Fallback slider" min={10} max={20} />);

    const slider = screen.getByRole('slider', { name: 'Fallback slider' });

    expect(slider.id).toMatch(/^slider-/);
    expect(slider).toHaveValue('10');
  });
});

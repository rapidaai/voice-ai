import { act } from '@testing-library/react';

export function setRangeValue(input: HTMLInputElement, value: number) {
  const setValue = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    'value',
  )?.set;

  if (!setValue) {
    throw new Error('Range input value setter is unavailable');
  }

  act(() => {
    setValue.call(input, String(value));
    input.dispatchEvent(new Event('change', { bubbles: true }));
  });
}

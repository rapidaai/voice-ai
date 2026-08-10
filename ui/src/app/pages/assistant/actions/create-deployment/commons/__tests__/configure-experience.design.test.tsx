import React from 'react';
import { render } from '@testing-library/react';
import {
  ConfigureExperience,
  DEFAULT_IDEAL_TIMEOUT,
} from '../configure-experience';

const mockSlider = jest.fn();

jest.mock('@/app/components/carbon/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextArea: ({ labelText, value }: any) => (
    <textarea aria-label={labelText} value={value} readOnly />
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  Information: () => null,
}));

jest.mock('@carbon/react', () => ({
  Button: ({ children }: any) => <button>{children}</button>,
  ComboBox: ({ 'aria-label': ariaLabel, selectedItem }: any) => (
    <input aria-label={ariaLabel} value={selectedItem || ''} readOnly />
  ),
  FormLabel: ({ children, id }: any) => <label id={id}>{children}</label>,
  Slider: (props: any) => {
    mockSlider(props);
    return (
      <input
        aria-label={props.labelText}
        data-max={props.max}
        data-min={props.min}
        data-value={props.value}
        readOnly
      />
    );
  },
  Toggletip: ({ children }: any) => <div>{children}</div>,
  ToggletipActions: ({ children }: any) => <div>{children}</div>,
  ToggletipButton: ({ children, label }: any) => (
    <button aria-label={label}>{children}</button>
  ),
  ToggletipContent: ({ children }: any) => <div>{children}</div>,
  Toggle: ({ toggled }: any) => (
    <input aria-label="toggle" checked={toggled} readOnly type="checkbox" />
  ),
}));

describe('ConfigureExperience', () => {
  beforeEach(() => {
    mockSlider.mockClear();
  });

  it('defaults idle timeout to 10 seconds within backend validation limits', () => {
    render(
      <ConfigureExperience
        experienceConfig={{}}
        setExperienceConfig={jest.fn()}
      />,
    );

    const idleTimeoutSlider = mockSlider.mock.calls
      .map(([props]) => props)
      .find(props => props.id === 'experience-idle-timeout');

    expect(DEFAULT_IDEAL_TIMEOUT).toBe('10');
    expect(idleTimeoutSlider).toMatchObject({
      min: 5,
      max: 120,
      value: 10,
    });
  });
});

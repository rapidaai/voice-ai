import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { HelpToggletip } from '../help-label';
import userEvent from '@testing-library/user-event';
import {
  BargeInTriggerControl,
  LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY,
  MICROPHONE_BARGE_IN_TRIGGER_KEY,
} from '../microphone/barge-in-trigger-control';

jest.mock('@carbon/icons-react', () => ({
  Information: () => <svg data-testid="information-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Select: ({ id, labelText, value, onChange, children }: any) => (
    <div>
      <label htmlFor={id}>{labelText}</label>
      <select id={id} value={value} onChange={onChange}>
        {children}
      </select>
    </div>
  ),
  SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
  Toggletip: ({ children }: any) => <span>{children}</span>,
  ToggletipButton: ({ children, label }: any) => (
    <button aria-label={label} type="button">
      {children}
    </button>
  ),
  ToggletipContent: ({ children }: any) => <span>{children}</span>,
}));

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

const getMetadataValue = (parameters: Metadata[], key: string) =>
  parameters.find(parameter => parameter.getKey() === key)?.getValue();

describe('provider controls', () => {
  it('renders help toggletip content only when help text is available', () => {
    const { container, rerender } = render(
      <HelpToggletip label="Provider" helpText="Provider help" />,
    );

    expect(
      screen.getByRole('button', { name: 'Provider information' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Provider help')).toBeInTheDocument();
    expect(screen.getByTestId('information-icon')).toBeInTheDocument();

    rerender(<HelpToggletip label="Provider" />);

    expect(container).toBeEmptyDOMElement();
  });

  it('defaults the barge-in trigger to VAD and writes the current metadata key', async () => {
    const user = userEvent.setup();
    const onChangeParameter = jest.fn();

    render(
      <BargeInTriggerControl
        parameters={[createMetadata('unrelated', 'keep-me')]}
        onChangeParameter={onChangeParameter}
      />,
    );

    const select = screen.getByRole('combobox');
    expect(select).toHaveValue('vad');

    await user.selectOptions(select, String('word'));

    const nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(getMetadataValue(nextParameters, 'unrelated')).toBe('keep-me');
    expect(
      getMetadataValue(nextParameters, MICROPHONE_BARGE_IN_TRIGGER_KEY),
    ).toBe('word');
  });

  it('prefers the current barge-in metadata key over legacy values', () => {
    render(
      <BargeInTriggerControl
        parameters={[
          createMetadata(LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY, 'word'),
          createMetadata(MICROPHONE_BARGE_IN_TRIGGER_KEY, 'vad'),
        ]}
        onChangeParameter={() => undefined}
      />,
    );

    expect(screen.getByRole('combobox')).toHaveValue('vad');
  });

  it('uses legacy barge-in metadata and removes it after changes', async () => {
    const user = userEvent.setup();
    const onChangeParameter = jest.fn();

    render(
      <BargeInTriggerControl
        parameters={[
          createMetadata(LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY, 'word'),
        ]}
        onChangeParameter={onChangeParameter}
      />,
    );

    const select = screen.getByRole('combobox');
    expect(select).toHaveValue('word');

    await user.selectOptions(select, String('vad'));

    const nextParameters = onChangeParameter.mock.calls[0][0] as Metadata[];
    expect(
      getMetadataValue(
        nextParameters,
        LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY,
      ),
    ).toBeUndefined();
    expect(
      getMetadataValue(nextParameters, MICROPHONE_BARGE_IN_TRIGGER_KEY),
    ).toBe('vad');
  });
});

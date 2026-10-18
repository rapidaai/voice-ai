import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { ConfigureTransferCall } from '../transfer-call';
import userEvent from '@testing-library/user-event';
import { setRangeValue } from '@/test-utils/set-range-value';

jest.mock('@/app/components/ui/primitives/form', () => {
  const React = require('react');
  return {
    FormGroup: ({ legendText, children }: any) =>
      React.createElement(
        'fieldset',
        null,
        legendText ? React.createElement('legend', null, legendText) : null,
        children,
      ),
    Stack: ({ children }: any) => React.createElement('div', null, children),
    TextArea: ({
      id,
      labelText,
      value,
      onChange,
      helperText,
      placeholder,
    }: any) =>
      React.createElement(
        'div',
        null,
        labelText
          ? React.createElement('label', { htmlFor: id }, labelText)
          : null,
        helperText ? React.createElement('small', null, helperText) : null,
        React.createElement('textarea', {
          id,
          'data-testid': id,
          value: value ?? '',
          onChange,
          placeholder,
        }),
      ),
  };
});

jest.mock('@carbon/react', () => {
  const React = require('react');
  return {
    Slider: ({ id, labelText, value, onChange, min, max, step }: any) =>
      React.createElement(
        'div',
        null,
        labelText
          ? React.createElement('label', { htmlFor: id }, labelText)
          : null,
        React.createElement('input', {
          id,
          type: 'range',
          min,
          max,
          step,
          value: value ?? 0,
          'data-testid': id,
          onChange: (e: any) => onChange?.({ value: Number(e.target.value) }),
        }),
      ),
    Select: ({ id, labelText, value, onChange, children }: any) =>
      React.createElement(
        'div',
        null,
        labelText
          ? React.createElement('label', { htmlFor: id }, labelText)
          : null,
        React.createElement(
          'select',
          { id, value: value ?? '', 'data-testid': id, onChange },
          children,
        ),
      ),
    SelectItem: ({ value, text }: any) =>
      React.createElement('option', { value }, text),
    Tooltip: ({ children }: any) => React.createElement('span', null, children),
  };
});

jest.mock('@/app/components/ui/primitives/input-group', () => {
  const React = require('react');
  return {
    InputGroup: ({ title, children }: any) =>
      React.createElement(
        'section',
        null,
        title ? React.createElement('h3', null, title) : null,
        children,
      ),
  };
});

jest.mock(
  '@/app/components/domain/configuration/config-var/config-select',
  () => {
    const React = require('react');
    return ({ options, label }: any) =>
      React.createElement(
        'div',
        { 'data-testid': 'transfer-destination-select' },
        `${label}: ${(options || []).join(',')}`,
      );
  },
);

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

describe('ConfigureTransferCall', () => {
  it('loads and updates transfer message and transfer delay for save payload', async () => {
    const user = userEvent.setup();
    const onParameterChange = jest.fn();
    const params = [
      createMetadata('tool.transfer_to', '+14155551234'),
      createMetadata(
        'tool.transfer_message',
        'Please hold while I transfer your call.',
      ),
      createMetadata('tool.transfer_delay', '700'),
      createMetadata('tool.post_transfer_action', 'end_call'),
      createMetadata('tool.ringtone', 'ring-ring'),
    ];

    render(
      <ConfigureTransferCall
        parameters={params}
        onParameterChange={onParameterChange}
      />,
    );

    const message = screen.getByTestId('transfer-message');
    const delay = screen.getByTestId('transfer-delay');
    const postTransferAction = screen.getByTestId('post-transfer-action');
    const ringtone = screen.getByTestId('transfer-ringtone');

    expect(message).toHaveValue('Please hold while I transfer your call.');
    expect(delay).toHaveValue('700');
    expect(postTransferAction).toHaveValue('end_call');
    expect(ringtone).toHaveValue('ring-ring');

    await user.click(message);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Connecting you now to our support specialist.'));

    const afterMessageUpdate = onParameterChange.mock.calls.at(
      -1,
    )?.[0] as Metadata[];
    expect(
      afterMessageUpdate
        .find(m => m.getKey() === 'tool.transfer_message')
        ?.getValue(),
    ).toBe('Connecting you now to our support specialist.');
    expect(
      afterMessageUpdate
        .find(m => m.getKey() === 'tool.transfer_delay')
        ?.getValue(),
    ).toBe('700');

    await user.click(delay);
    setRangeValue(delay as HTMLInputElement, 650);

    const afterDelayUpdate = onParameterChange.mock.calls.at(
      -1,
    )?.[0] as Metadata[];
    expect(
      afterDelayUpdate
        .find(m => m.getKey() === 'tool.transfer_delay')
        ?.getValue(),
    ).toBe('650');
    expect(
      afterDelayUpdate
        .find(m => m.getKey() === 'tool.transfer_message')
        ?.getValue(),
    ).toBe('Please hold while I transfer your call.');

    await user.selectOptions(postTransferAction, String('resume_ai'));

    const afterActionUpdate = onParameterChange.mock.calls.at(
      -1,
    )?.[0] as Metadata[];
    expect(
      afterActionUpdate
        .find(m => m.getKey() === 'tool.post_transfer_action')
        ?.getValue(),
    ).toBe('resume_ai');

    await user.selectOptions(ringtone, String('transfer-music'));

    const afterRingtoneUpdate = onParameterChange.mock.calls.at(
      -1,
    )?.[0] as Metadata[];
    expect(
      afterRingtoneUpdate.find(m => m.getKey() === 'tool.ringtone')?.getValue(),
    ).toBe('transfer-music');
  });

  it('shows defaults for transfer delay and ringtone when parameters are missing', () => {
    render(
      <ConfigureTransferCall
        parameters={[createMetadata('tool.transfer_to', '+14155551234')]}
        onParameterChange={jest.fn()}
      />,
    );

    const delay = screen.getByTestId('transfer-delay');
    const ringtone = screen.getByTestId('transfer-ringtone');

    expect(delay).toHaveValue('500');
    expect(ringtone).toHaveValue('default');
  });
});

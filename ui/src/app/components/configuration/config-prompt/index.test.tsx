import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { ConfigPrompt } from '@/app/components/configuration/config-prompt';

jest.mock('random-words', () => ({
  generate: () => 'stub-word',
}));

jest.mock('@carbon/react', () => {
  const actual = jest.requireActual('@carbon/react');
  return {
    ...actual,
    Toggletip: ({ children }: any) => <span>{children}</span>,
    ToggletipButton: ({ children, label }: any) => (
      <button type="button" aria-label={label}>
        {children || label}
      </button>
    ),
    ToggletipContent: ({ children }: any) => <span>{children}</span>,
  };
});

jest.mock(
  '@/app/components/configuration/config-prompt/advanced-prompt-input',
  () => ({
    __esModule: true,
    default: (props: any) => (
      <div data-testid="advanced-message-input">
        <button
          type="button"
          onClick={() =>
            props.onChange(
              'Hello {{assistant.name}} and {{customer_name}} {{args.city}}',
            )
          }
        >
          trigger-change
        </button>
        <button
          type="button"
          onClick={() => props.onChange('Hello {{customer_name}}')}
        >
          trigger-change-custom-only
        </button>
      </div>
    ),
  }),
);

describe('ConfigPrompt argument hints', () => {
  const basePrompt = {
    prompt: [{ role: 'system', content: 'You are {{assistant.name}}' }],
    variables: [{ name: 'assistant.name', type: 'text', defaultvalue: '' }],
  };

  it('shows runtime argument hint text when hints are provided', () => {
    render(
      <ConfigPrompt
        existingPrompt={basePrompt}
        showRuntimeReplacementHint
        onChange={() => {}}
      />,
    );

    expect(
      screen.getByRole('button', { name: /Reserved Variables/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /These variables are preserved and replaced at runtime/i,
      ),
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole('button', { name: /Reserved Variables/i }),
    );

    expect(screen.getByText(/Runtime value/i)).toBeInTheDocument();
    expect(
      screen.getAllByText('{{system.current_date}}').length,
    ).toBeGreaterThan(0);
    expect(screen.getAllByText('{{assistant.name}}').length).toBeGreaterThan(0);
    expect(screen.getAllByText('{{client.phone}}').length).toBeGreaterThan(0);
    expect(screen.getAllByText('{{client.provider_call_id}}').length).toBeGreaterThan(0);
    expect(screen.queryByText('{{session.mode}}')).not.toBeInTheDocument();
    expect(
      screen.queryByText('{{conversation.updated_date}}'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('{{conversation.duration}}'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('{{assistant.language}}'),
    ).not.toBeInTheDocument();
  });

  it('does not show runtime argument hint text when hints are not provided', () => {
    render(<ConfigPrompt existingPrompt={basePrompt} onChange={() => {}} />);
    expect(
      screen.queryByText(/Reserved Variables/i),
    ).not.toBeInTheDocument();
  });

  it('shows arguments guidance even when template-specific variable list is empty', () => {
    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'system', content: 'hello' }],
          variables: [],
        }}
        showRuntimeReplacementHint
        onChange={() => {}}
      />,
    );

    expect(screen.getByText('Arguments')).toBeInTheDocument();
    expect(
      screen.getByText(
        /Reserved variables are preserved and replaced at runtime/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/No template-specific variables yet/i),
    ).toBeInTheDocument();
  });

  it('keeps reserved variables in arguments list with runtime hint enabled', () => {
    const onChange = jest.fn();
    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'system', content: '' }],
          variables: [],
        }}
        showRuntimeReplacementHint
        onChange={onChange}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'trigger-change' }));

    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1][0];
    expect(lastCall.variables).toEqual([
      { name: 'assistant.name', type: 'text', defaultvalue: '' },
      { name: 'customer_name', type: 'text', defaultvalue: '' },
      { name: 'args.city', type: 'text', defaultvalue: '' },
    ]);
  });

  it('marks reserved variables with Reserved label in arguments list', () => {
    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'system', content: '' }],
          variables: [
            { name: 'assistant.name', type: 'text', defaultvalue: '' },
            { name: 'customer_name', type: 'text', defaultvalue: '' },
          ],
        }}
        showRuntimeReplacementHint
        onChange={() => {}}
      />,
    );

    expect(screen.getAllByText('Reserved').length).toBe(1);
  });

  it('removes reserved variables from arguments once removed from prompt content', () => {
    const onChange = jest.fn();
    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'system', content: '' }],
          variables: [],
        }}
        showRuntimeReplacementHint
        onChange={onChange}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'trigger-change' }));
    fireEvent.click(
      screen.getByRole('button', { name: 'trigger-change-custom-only' }),
    );

    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1][0];
    expect(lastCall.variables).toEqual([
      { name: 'customer_name', type: 'text', defaultvalue: '' },
    ]);
  });

  it('keeps all extracted variables when Rapida filtering is disabled', () => {
    const onChange = jest.fn();
    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'system', content: '' }],
          variables: [],
        }}
        onChange={onChange}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'trigger-change' }));

    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1][0];
    expect(lastCall.variables).toEqual([
      { name: 'assistant.name', type: 'text', defaultvalue: '' },
      { name: 'customer_name', type: 'text', defaultvalue: '' },
      { name: 'args.city', type: 'text', defaultvalue: '' },
    ]);
  });
});

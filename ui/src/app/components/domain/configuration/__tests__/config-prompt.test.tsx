import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { MAX_PROMPT_MESSAGE_LENGTH } from '@/configs';
import { ConfigPrompt } from '../config-prompt';

jest.mock('random-words', () => ({
  generate: () => 'stub-word',
}));

jest.mock('@carbon/react', () => {
  const actual = jest.requireActual('@carbon/react');
  return {
    ...actual,
    Button: ({
      children,
      className: _className,
      disabled,
      hasIconOnly: _hasIconOnly,
      iconDescription,
      kind,
      renderIcon: Icon,
      size: _size,
      tooltipPosition: _tooltipPosition,
      ...props
    }: any) => (
      <button
        type="button"
        aria-label={iconDescription}
        data-design-system-button-kind={kind}
        disabled={disabled}
        {...props}
      >
        {children}
        {Icon ? <Icon /> : null}
      </button>
    ),
    ComposedModal: ({ children, open }: any) =>
      open ? <div role="dialog">{children}</div> : null,
    ModalBody: ({ children }: any) => <section>{children}</section>,
    ModalFooter: ({ children }: any) => <footer>{children}</footer>,
    ModalHeader: ({ buttonOnClick, label, title }: any) => (
      <header>
        {label ? <p>{label}</p> : null}
        <h2>{title}</h2>
        <button type="button" aria-label="Close" onClick={buttonOnClick}>
          Close
        </button>
      </header>
    ),
    Table: ({ children }: any) => <table>{children}</table>,
    TableBody: ({ children }: any) => <tbody>{children}</tbody>,
    TableCell: ({ children, className, colSpan }: any) => (
      <td className={className} colSpan={colSpan}>
        {children}
      </td>
    ),
    TableContainer: ({ children, className }: any) => (
      <section className={className}>{children}</section>
    ),
    TableHead: ({ children }: any) => <thead>{children}</thead>,
    TableHeader: ({ children }: any) => <th>{children}</th>,
    TableRow: ({ children }: any) => <tr>{children}</tr>,
    Toggletip: ({ children }: any) => <span>{children}</span>,
    ToggletipButton: ({ children, label }: any) => (
      <button type="button" aria-label={label}>
        {children || label}
      </button>
    ),
    ToggletipContent: ({ children }: any) => <span>{children}</span>,
  };
});

jest.mock('@carbon/icons-react', () => {
  const actual = jest.requireActual('@carbon/icons-react');
  return {
    ...actual,
    Add: () => <svg data-testid="add-message-icon" />,
  };
});

jest.mock(
  '@/app/components/domain/configuration/config-prompt/advanced-prompt-input',
  () => ({
    __esModule: true,
    default: (props: any) => (
      <div
        data-testid="advanced-message-input"
        data-can-delete={String(props.canDelete)}
        data-enable-reserved={String(props.enableReservedVariableSuggestions)}
        data-instance-id={props.instanceId}
        data-type={props.type}
      >
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
        <button type="button" onClick={() => props.onTypeChange('assistant')}>
          trigger-type-change
        </button>
        <button
          type="button"
          disabled={!props.canDelete}
          onClick={props.onDelete}
        >
          trigger-delete
        </button>
      </div>
    ),
  }),
);

jest.mock(
  '@/app/components/domain/configuration/config-prompt/type-of-variable',
  () => ({
    TypeOfVariable: ({
      allType,
      'aria-label': ariaLabel = 'Variable type',
      id,
      onChange,
      type,
    }: any) => (
      <select
        aria-label={ariaLabel}
        id={id}
        value={type}
        onChange={event => onChange(event.target.value)}
      >
        {allType.map((item: string) => (
          <option key={item} value={item}>
            {item}
          </option>
        ))}
      </select>
    ),
  }),
);

describe('ConfigPrompt', () => {
  const basePrompt = {
    prompt: [{ role: 'system', content: 'You are {{assistant.name}}' }],
    variables: [{ name: 'assistant.name', type: 'text', defaultvalue: '' }],
  };

  it('shows runtime argument hint text when hints are provided', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <ConfigPrompt
        existingPrompt={basePrompt}
        showRuntimeReplacementHint
        onChange={() => {}}
      />,
    );

    expect(
      screen.getByRole('button', { name: /Reserved Variables/i }),
    ).toHaveAttribute('data-design-system-button-kind', 'ghost');

    const reservedButton = screen.getByRole('button', {
      name: /Reserved Variables/i,
    });
    expect(reservedButton.firstElementChild).toHaveClass('flex-col');
    expect(
      screen.getByText(/These variables are preserved and replaced at runtime/i)
        .parentElement,
    ).toBe(reservedButton.firstElementChild);

    expect(
      screen.getByText(
        /These variables are preserved and replaced at runtime/i,
      ),
    ).toBeInTheDocument();

    await user.click(
      screen.getByRole('button', { name: /Reserved Variables/i }),
    );

    expect(screen.getByText(/Runtime value/i)).toBeInTheDocument();
    expect(
      screen.getAllByText('{{system.current_date}}').length,
    ).toBeGreaterThan(0);
    expect(screen.getAllByText('{{assistant.name}}').length).toBeGreaterThan(0);
    expect(screen.getAllByText('{{client.phone}}').length).toBeGreaterThan(0);
    expect(
      screen.getAllByText('{{client.provider_call_id}}').length,
    ).toBeGreaterThan(0);
    expect(screen.queryByText('{{session.mode}}')).not.toBeInTheDocument();
  });

  it('does not show runtime argument hint text when hints are not provided', () => {
    render(<ConfigPrompt existingPrompt={basePrompt} onChange={() => {}} />);

    expect(screen.queryByText(/Reserved Variables/i)).not.toBeInTheDocument();
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

  it('can hide the arguments runtime hint while still showing arguments', () => {
    render(
      <ConfigPrompt
        existingPrompt={basePrompt}
        showRuntimeReplacementHint
        hideArgumentRuntimeHint
        onChange={() => {}}
      />,
    );

    expect(screen.getByText('Arguments')).toBeInTheDocument();
    expect(
      screen.queryByText(/Add only your template-specific variables here/i),
    ).not.toBeInTheDocument();
  });

  it('keeps reserved variables in arguments list with runtime hint enabled', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
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

    await user.click(screen.getByRole('button', { name: 'trigger-change' }));

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

  it('removes reserved variables from arguments once removed from prompt content', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
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

    await user.click(screen.getByRole('button', { name: 'trigger-change' }));
    await user.click(
      screen.getByRole('button', { name: 'trigger-change-custom-only' }),
    );

    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1][0];
    expect(lastCall.variables).toEqual([
      { name: 'customer_name', type: 'text', defaultvalue: '' },
    ]);
  });

  it('keeps all extracted variables when runtime hints are disabled', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
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

    await user.click(screen.getByRole('button', { name: 'trigger-change' }));

    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1][0];
    expect(lastCall.variables).toEqual([
      { name: 'assistant.name', type: 'text', defaultvalue: '' },
      { name: 'customer_name', type: 'text', defaultvalue: '' },
      { name: 'args.city', type: 'text', defaultvalue: '' },
    ]);
  });

  it('preserves existing variable configuration when prompt content still references it', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();
    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'system', content: '' }],
          variables: [
            {
              name: 'customer_name',
              type: 'paragraph',
              defaultvalue: 'Existing customer',
            },
          ],
        }}
        onChange={onChange}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: 'trigger-change-custom-only' }),
    );

    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1][0];
    expect(lastCall.variables).toEqual([
      {
        name: 'customer_name',
        type: 'paragraph',
        defaultvalue: 'Existing customer',
      },
    ]);
  });

  it('updates one prompt message without changing the other messages', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();
    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [
            { role: 'system', content: 'system message' },
            { role: 'user', content: 'user message' },
          ],
          variables: [],
        }}
        onChange={onChange}
      />,
    );

    await user.click(
      screen.getAllByRole('button', { name: 'trigger-change' })[0],
    );

    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1][0];
    expect(lastCall.prompt).toEqual([
      {
        role: 'system',
        content: 'Hello {{assistant.name}} and {{customer_name}} {{args.city}}',
      },
      { role: 'user', content: 'user message' },
    ]);
  });

  it('adds a new message with the next alternating role', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'user', content: 'hello' }],
          variables: [],
        }}
        onChange={onChange}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Add new message' }));

    expect(onChange).toHaveBeenCalledWith({
      prompt: [
        { role: 'user', content: 'hello' },
        { role: 'assistant', content: '' },
      ],
      variables: [],
    });
  });

  it('adds a user message after an assistant message', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'assistant', content: 'hello' }],
          variables: [],
        }}
        onChange={onChange}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Add new message' }));

    expect(onChange).toHaveBeenCalledWith({
      prompt: [
        { role: 'assistant', content: 'hello' },
        { role: 'user', content: '' },
      ],
      variables: [],
    });
  });

  it('does not show add message after the maximum message count', () => {
    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: Array.from(
            { length: MAX_PROMPT_MESSAGE_LENGTH },
            (_, index) => ({
              role: index % 2 === 0 ? 'user' : 'assistant',
              content: `message ${index}`,
            }),
          ),
          variables: [],
        }}
        onChange={() => {}}
      />,
    );

    expect(
      screen.queryByRole('button', { name: 'Add new message' }),
    ).not.toBeInTheDocument();
  });

  it('updates message role and deletes prompt messages', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [
            { role: 'user', content: 'hello' },
            { role: 'assistant', content: 'hi' },
          ],
          variables: [],
        }}
        instanceId="prompt"
        enableReservedVariableSuggestions
        onChange={onChange}
      />,
    );

    expect(screen.getAllByTestId('advanced-message-input')[0]).toHaveAttribute(
      'data-instance-id',
      'prompt-user-0',
    );
    expect(screen.getAllByTestId('advanced-message-input')[0]).toHaveAttribute(
      'data-enable-reserved',
      'true',
    );

    await user.click(
      screen.getAllByRole('button', { name: 'trigger-type-change' })[0],
    );
    expect(onChange).toHaveBeenLastCalledWith({
      prompt: [
        { role: 'assistant', content: 'hello' },
        { role: 'assistant', content: 'hi' },
      ],
      variables: [],
    });

    await user.click(
      screen.getAllByRole('button', { name: 'trigger-delete' })[0],
    );
    expect(onChange).toHaveBeenLastCalledWith({
      prompt: [{ role: 'assistant', content: 'hi' }],
      variables: [],
    });
  });

  it('edits argument type and default value inline from its table row action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'system', content: 'Hello {{customer_name}}' }],
          variables: [
            { name: 'customer_name', type: 'text', defaultvalue: 'Priyanka' },
            { name: 'account_id', type: 'text', defaultvalue: '123' },
          ],
        }}
        onChange={onChange}
      />,
    );

    expect(screen.getByText('Variable')).toBeInTheDocument();
    expect(screen.getByText('Type')).toBeInTheDocument();
    expect(screen.getByText('Default value')).toBeInTheDocument();
    expect(screen.getByText('Action')).toBeInTheDocument();
    expect(screen.getByRole('table').parentElement).toHaveClass(
      '![padding-block-start:0]',
      'border-border-subtle',
    );

    const editButton = screen.getByRole('button', {
      name: 'Edit customer_name',
    });
    expect(editButton.closest('td')).toHaveClass('!text-center');

    await user.click(editButton);

    const variableInput = screen.getByLabelText('Variable customer_name');
    expect(variableInput).toHaveValue('customer_name');
    expect(variableInput).toBeDisabled();
    expect(variableInput.closest('td')).toHaveClass('!p-0');

    const typeSelect = screen.getByLabelText('Type for customer_name');
    expect(typeSelect).toHaveAttribute('id', 'argument-type-customer_name');
    expect(typeSelect.closest('td')).toHaveClass('!p-0');

    await user.selectOptions(typeSelect, String('number'));

    const defaultInput = screen.getByPlaceholderText('Optional default value');
    expect(defaultInput.closest('td')).toHaveClass('!p-0');
    await user.click(defaultInput);
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('42'));
    expect(onChange).not.toHaveBeenCalled();

    const saveButton = screen.getByRole('button', {
      name: 'Save customer_name',
    });
    expect(saveButton.parentElement).toHaveClass('justify-center');

    await user.click(saveButton);

    expect(onChange).toHaveBeenLastCalledWith({
      prompt: [{ role: 'system', content: 'Hello {{customer_name}}' }],
      variables: [
        { name: 'customer_name', type: 'number', defaultvalue: '42' },
        { name: 'account_id', type: 'text', defaultvalue: '123' },
      ],
    });
  });

  it('cancels inline argument edits without changing the prompt', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ConfigPrompt
        existingPrompt={{
          prompt: [{ role: 'system', content: 'Hello {{customer_name}}' }],
          variables: [
            { name: 'customer_name', type: 'text', defaultvalue: 'Priyanka' },
          ],
        }}
        onChange={onChange}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: 'Edit customer_name' }),
    );
    await user.click(screen.getByPlaceholderText('Optional default value'));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('Changed'));
    await user.click(
      screen.getByRole('button', { name: 'Cancel customer_name' }),
    );

    expect(onChange).not.toHaveBeenCalled();
    expect(screen.queryByPlaceholderText('Optional default value')).toBeNull();
    expect(screen.getByText('Priyanka')).toBeInTheDocument();
  });

  it('uses the standard add icon for adding a prompt message', () => {
    render(<ConfigPrompt existingPrompt={basePrompt} onChange={() => {}} />);

    expect(screen.getByTestId('add-message-icon')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Add new message' }),
    ).toHaveAttribute('data-design-system-button-kind', 'tertiary');
  });
});

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ConfigureAssistantToolDialog } from '../assistant-configure-tool-modal';

const mockValidateToolDefaultOptions = jest.fn();

jest.mock('@/configs', () => ({
  CONFIG: {
    workspace: {
      features: {
        knowledge: true,
      },
    },
  },
}));

jest.mock('@/app/components/ui/primitives/modal', () => ({
  Modal: ({ open, children }: any) => (open ? <div>{children}</div> : null),
  ModalHeader: ({ title }: any) => <div>{title}</div>,
  ModalBody: ({ children }: any) => <div>{children}</div>,
  ModalFooter: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@/app/components/ui/feedback/notification', () => ({
  Notification: ({ subtitle }: any) => <div>{subtitle}</div>,
}));

jest.mock('@/app/components/domain/tools/tool-registry', () => ({
  BuildinTool: ({
    onChangeToolDefinition,
    onChangeBuildinTool,
    onChangeConfig,
    config,
  }: any) => (
    <div>
      <button
        onClick={() =>
          onChangeToolDefinition({
            name: 'tool_name',
            description: 'tool_description',
            parameters: '{"k":"v"}',
          })
        }
      >
        Set Valid Definition
      </button>
      <button
        onClick={() =>
          onChangeToolDefinition({
            name: 'tool_name',
            description: 'tool_description',
            parameters: '{bad-json',
          })
        }
      >
        Set Invalid JSON Definition
      </button>
      <button
        onClick={() =>
          onChangeToolDefinition({
            name: '',
            description: 'tool_description',
            parameters: '{"k":"v"}',
          })
        }
      >
        Set Missing Name
      </button>
      <button
        onClick={() =>
          onChangeToolDefinition({
            name: 'bad-name',
            description: 'tool_description',
            parameters: '{"k":"v"}',
          })
        }
      >
        Set Invalid Name
      </button>
      <button
        onClick={() =>
          onChangeToolDefinition({
            name: 'tool_name',
            description: '',
            parameters: '{"k":"v"}',
          })
        }
      >
        Set Missing Description
      </button>
      <button
        onClick={() =>
          onChangeToolDefinition({
            name: 'tool_name',
            description: 'tool_description',
            parameters: '',
          })
        }
      >
        Set Missing Parameters
      </button>
      <button onClick={() => onChangeBuildinTool('mcp')}>Use MCP</button>
      <button
        onClick={() =>
          onChangeConfig({
            ...config,
            code: 'mcp',
            parameters: [
              {
                getKey: () => 'mcp.server_url',
                getValue: () => 'https://mcp.example.com',
              },
            ],
          })
        }
      >
        Set MCP Config
      </button>
    </div>
  ),
  BuildinToolConfig: {},
  GetDefaultToolDefintion: (code: string, existing?: any) => {
    if (existing?.name && existing?.description && existing?.parameters) {
      return existing;
    }
    if (code === 'mcp') {
      return {
        name: 'mcp_tool',
        description: 'MCP tool',
        parameters: '{"type":"object"}',
      };
    }
    return {
      name: 'knowledge_tool',
      description: 'Knowledge tool',
      parameters: '{"type":"object"}',
    };
  },
  GetDefaultToolConfigIfInvalid: (code: string, params: any[]) => params || [],
  ValidateToolDefaultOptions: (...args: unknown[]) =>
    mockValidateToolDefaultOptions(...args),
}));

describe('ConfigureAssistantToolDialog', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockValidateToolDefaultOptions.mockReturnValue(undefined);
  });

  it('submits valid create payload via onChange', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
        onChange={onChange}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: 'Set Valid Definition' }),
    );
    await user.click(screen.getByRole('button', { name: 'Save tool' }));

    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange.mock.calls[0][0]).toMatchObject({
      name: 'tool_name',
      description: 'tool_description',
      fields: '{"k":"v"}',
    });
  });

  it('shows validation error for invalid JSON parameters', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: 'Set Invalid JSON Definition' }),
    );
    await user.click(screen.getByRole('button', { name: 'Save tool' }));

    expect(
      screen.getByText(
        'Please provide a valid parameter, parameter must be a valid JSON.',
      ),
    ).toBeInTheDocument();
  });

  it('shows validation errors for missing required definition fields', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const { rerender } = render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Set Missing Name' }));
    await user.click(screen.getByRole('button', { name: 'Save tool' }));
    expect(
      screen.getByText('Please provide a valid name for tool.'),
    ).toBeInTheDocument();

    rerender(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );
    await user.click(screen.getByRole('button', { name: 'Set Invalid Name' }));
    await user.click(screen.getByRole('button', { name: 'Save tool' }));
    expect(
      screen.getByText(
        'Name should only contain letters, numbers, and underscores.',
      ),
    ).toBeInTheDocument();

    rerender(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );
    await user.click(
      screen.getByRole('button', { name: 'Set Missing Description' }),
    );
    await user.click(screen.getByRole('button', { name: 'Save tool' }));
    expect(
      screen.getByText('Please provide a valid description for the tool.'),
    ).toBeInTheDocument();

    rerender(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );
    await user.click(
      screen.getByRole('button', { name: 'Set Missing Parameters' }),
    );
    await user.click(screen.getByRole('button', { name: 'Save tool' }));
    expect(
      screen.getByText('Please provide a valid parameters for the tool.'),
    ).toBeInTheDocument();
  });

  it('surfaces tool option validation errors', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockValidateToolDefaultOptions.mockReturnValue('Select a valid tool.');

    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: 'Set Valid Definition' }),
    );
    await user.click(screen.getByRole('button', { name: 'Save tool' }));

    expect(screen.getByText('Select a valid tool.')).toBeInTheDocument();
  });

  it('surfaces parent onValidateConfig error and blocks submit', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();

    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
        onChange={onChange}
        onValidateConfig={() => 'Please provide a unique tool name for tools.'}
      />,
    );

    await user.click(
      screen.getByRole('button', { name: 'Set Valid Definition' }),
    );
    await user.click(screen.getByRole('button', { name: 'Save tool' }));

    expect(
      screen.getByText('Please provide a unique tool name for tools.'),
    ).toBeInTheDocument();
    expect(onChange).not.toHaveBeenCalled();
  });

  it('loads initial data on edit and validates using its tool code', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={{
          name: 'old_tool',
          description: 'old desc',
          fields: '{"x":"y"}',
          buildinToolConfig: {
            code: 'knowledge_retrieval',
            parameters: [],
          },
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Save tool' }));
    expect(mockValidateToolDefaultOptions).toHaveBeenCalledWith(
      'knowledge_retrieval',
      [],
    );
  });

  it('falls back to the default tool code when edit data has no code', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={
          {
            name: 'old_tool',
            description: 'old desc',
            fields: '{"x":"y"}',
            buildinToolConfig: {
              parameters: [],
            },
          } as any
        }
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Save tool' }));
    expect(mockValidateToolDefaultOptions).toHaveBeenCalledWith(
      'knowledge_retrieval',
      [],
    );
  });

  it('resets local validation state after the dialog is closed', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const { rerender } = render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Set Missing Name' }));
    await user.click(screen.getByRole('button', { name: 'Save tool' }));
    expect(
      screen.getByText('Please provide a valid name for tool.'),
    ).toBeInTheDocument();

    rerender(
      <ConfigureAssistantToolDialog
        modalOpen={false}
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );
    rerender(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={jest.fn()}
        initialData={null}
      />,
    );

    expect(
      screen.queryByText('Please provide a valid name for tool.'),
    ).not.toBeInTheDocument();
  });

  it('updates the built-in tool configuration and closes from cancel', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChange = jest.fn();
    const setModalOpen = jest.fn();

    render(
      <ConfigureAssistantToolDialog
        modalOpen
        setModalOpen={setModalOpen}
        initialData={null}
        onChange={onChange}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Use MCP' }));
    await user.click(screen.getByRole('button', { name: 'Set MCP Config' }));
    await user.click(screen.getByRole('button', { name: 'Save tool' }));

    expect(onChange.mock.calls[0][0]).toMatchObject({
      name: 'mcp_tool',
      buildinToolConfig: {
        code: 'mcp',
      },
    });

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(setModalOpen).toHaveBeenCalledWith(false);
  });
});
